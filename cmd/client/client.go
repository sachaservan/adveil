package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/rpc"
	"sync"

	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/cmd/api"
	"github.com/sachaservan/adveil/cmd/sealpir"

	"github.com/sachaservan/vec"
)

// RuntimeExperiment captures all the information needed to
// evaluate a live deployment
type RuntimeExperiment struct {
	NumAds                   int     `json:"num_ads"`
	NumFeatures              int     `json:"num_features"`
	AdSizeKB                 int     `json:"ad_size_kilobytes"`
	NumTables                int     `json:"num_tables"`
	GetBucketServerMS        []int64 `json:"get_bucket_server_ms"`
	GetBucketClientMS        []int64 `json:"get_bucket_client_ms"`
	GetBucketBandwidthDownB  []int64 `json:"get_bucket_bandwidth_down_bytes"`
	GetBucketBandwidthUpB    []int64 `json:"get_bucket_bandwidth_up_bytes"`
	GetBucketBandwidthNaiveB []int64 `json:"get_bucket_bandwidth_naive_bytes"`
	GetAdServerMS            []int64 `json:"get_ad_server_ms"`
	GetAdClientMS            []int64 `json:"get_ad_client_ms"`
	GetAdBandwidthB          []int64 `json:"get_ad_bandwidth_bytes"`
	PrivateGetAdServerMS     []int64 `json:"private_get_ad_server_ms"`
	PrivateGetAdDPFServerMS  []int64 `json:"private_get_ad_dpf_server_ms"`
	PrivateGetAdClientMS     []int64 `json:"private_get_ad_client_ms"`
	PrivateGetAdBandwidthB   []int64 `json:"private_get_ad_bandwidth_bytes"`
}

const BrokerServerID int = 0

// Client is used to store all relevant client information
type Client struct {
	serverAddr    string
	serverPort    string
	sessionParams *api.SessionParameters

	// SealPIR related
	// NOTE: "client" here refers to the PIR client in SealPIR
	// and is a bridge between Go and C++ code
	tablePIRClient     *sealpir.Client     // clients used to query each tables
	tablePIRKeys       *sealpir.GaloisKeys // keys used to query each hash table
	tableNumBuckets    map[int]int         // number of hash buckets in each table
	tableHashFunctions map[int]*anns.LSH   // LSH functions used to query tables

	adPIRParams *sealpir.Params     // SealPIR params for the ad database
	adPIRClient *sealpir.Client     // client used to query ad database
	adPIRKeys   *sealpir.GaloisKeys // ad database SealPIR keys

	// client's profile feature vector
	profile    *vec.Vec
	experiment *RuntimeExperiment

	mu sync.Mutex
}

// WaitForExperimentStart completes once the servers are ready
// to start the experiments
func (client *Client) WaitForExperimentStart() {
	args := api.WaitForExperimentArgs{}
	res := api.WaitForExperimentResponse{}

	if !client.call("Server.WaitForExperiment", &args, &res) {
		panic("failed to make RPC call")
	}
}

// InitSession creates a new API session with the server
func (client *Client) InitSession() {

	args := &api.InitSessionArgs{}
	res := &api.InitSessionResponse{}

	if !client.call("Server.InitSession", &args, &res) {
		panic("failed to make RPC call")
	}

	if res.TablePIRParams != nil {
		// initialize the SealPIR clients used to query each hash table
		// using the params provided by the server
		c := sealpir.InitClient(sealpir.DeserializeParams(res.TablePIRParams), 0)
		keys := c.GenGaloisKeys()

		client.tablePIRClient = c
		client.tablePIRKeys = keys

		client.tableNumBuckets = res.TableNumBuckets
		client.tableHashFunctions = res.TableHashFunctions
	}

	// SealPIR ad database client and keys
	adC := sealpir.InitClient(sealpir.DeserializeParams(res.AdPIRParams), 0)
	client.adPIRKeys = adC.GenGaloisKeys()
	client.adPIRClient = adC

	client.sessionParams = &api.SessionParameters{
		SessionID:   res.SessionID,
		NumFeatures: res.NumFeatures,
		NumTables:   res.NumTables,
		NumAds:      res.NumAds,
	}

	// TODO: this is kind of a hack that is only ok for experiments
	// gen profile here once the client knows how many features the server is running
	client.profile = vec.NewRandomVec(res.NumFeatures, -50, 50)

	// init the experiment
	client.experiment.NumAds = res.NumAds
	client.experiment.AdSizeKB = res.AdSizeKB
	client.experiment.NumFeatures = res.NumFeatures
	client.experiment.NumTables = res.NumTables
}

func (client *Client) SendPIRKeys() {

	args := &api.SetKeysArgs{}
	res := &api.SetKeysResponse{}

	args.AdDBGaloisKeys = client.adPIRKeys
	args.TableDBGaloisKeys = client.tablePIRKeys

	if !client.call("Server.SetPIRKeys", &args, &res) {
		panic("failed to make RPC call")
	}

}

// TerminateSessions ends the client session on both servers
func (client *Client) TerminateSessions() {
	args := api.TerminateSessionArgs{}
	res := api.TerminateSessionResponse{}

	if !client.call("Server.TerminateSession", &args, &res) {
		panic("failed to make RPC call")
	}

	client.adPIRClient.Free()
	client.tablePIRClient.Free()

}

// PrivateQueryAd privately retrieves the ad at the index
func (client *Client) PrivateQueryAd(index int64) ([]byte, int64, int64, int64) {

	args := &api.AdQueryArgs{}
	res := &api.AdQueryResponse{}

	c := client.adPIRClient
	idx := c.GetFVIndex(index)
	offset := c.GetFVOffset(idx)
	query := c.GenQuery(idx)

	args.Query = query

	if !client.call("Server.PrivateAdQuery", &args, &res) {
		panic("failed to make RPC call")
	}

	// recover the result
	c.Recover(res.Answer[0], offset)

	bandwidth := getSizeInBytes(args) + getSizeInBytes(res)
	return nil, res.StatsTotalTimeInMS, res.StatsTotalTimeInMS, bandwidth
}

// QueryAd retrieves the ad at the index with no privacy
func (client *Client) QueryAd(index int64) ([]byte, int64, int64) {

	args := &api.AdQueryArgs{}
	res := &api.AdQueryResponse{}

	args.Index = index
	if !client.call("Server.AdQuery", &args, &res) {
		panic("failed to make RPC call")
	}

	bandwidth := getSizeInBytes(args) + getSizeInBytes(res)

	return nil, res.StatsTotalTimeInMS, bandwidth

}

// QueryBuckets privately queries LSH tables held by the server
// by first hashing the client's profile vector and then retrieving the corresponding
// hash from the hash table
func (client *Client) QueryBuckets() ([][]int, int64, int64, int64, int64) {

	qargs := &api.BucketQueryArgs{}
	qres := &api.BucketQueryResponse{}

	// query each hash table for the bucket that collides with the
	// client's profile feature vector under the server-provided LSH function
	qargs.Queries = make(map[int]*sealpir.Query)
	for tableIndex := 0; tableIndex < client.sessionParams.NumTables; tableIndex++ {

		h := client.tableHashFunctions[tableIndex]
		elemIndex := h.Digest(client.profile).Int64()

		c := client.tablePIRClient
		index := c.GetFVIndex(elemIndex)
		query := c.GenQuery(index)

		qargs.Queries[tableIndex] = query
	}

	if !client.call("Server.PrivateBucketQuery", &qargs, &qres) {
		panic("failed to make RPC call")
	}

	// recover the result
	// TODO: actually use the recovered result(s) to recover the NN
	// for tableIndex := 0; tableIndex < client.sessionParams.NumTables; tableIndex++ {

	// 	h := client.tableHashFunctions[tableIndex]
	// 	elemIndex := h.Digest(client.profile).Int64()

	// 	c := client.tablePIRClients[tableIndex]
	// 	offset := c.GetFVOffset(elemIndex)
	// 	c.Recover(qres.Answers[tableIndex][0], offset)
	// }

	bandwidthNaive := qres.StatsNaiveBandwidthBytes
	bandwidthUp := getSizeInBytes(qargs)
	bandwidthDown := getSizeInBytes(qres)
	serverMS := qres.StatsTotalTimeInMS

	return nil, serverMS, bandwidthUp, bandwidthDown, bandwidthNaive
}

func getSizeInBytes(s interface{}) int64 {
	var b bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&b) // Will write to network.
	err := enc.Encode(s)
	if err != nil {
		panic(err)
	}

	return int64(len(b.Bytes()))
}

// send an RPC request to the master, wait for the response
func (client *Client) call(rpcname string, args interface{}, reply interface{}) bool {

	cli, err := rpc.DialHTTP("tcp", client.serverAddr+":"+client.serverPort)
	if err != nil {
		log.Fatal("dialing:", err)
	}

	defer cli.Close()

	err = cli.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)

	return false
}
