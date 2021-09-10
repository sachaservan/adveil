package client

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/rpc"

	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/api"
	"github.com/sachaservan/adveil/sealpir"

	"github.com/sachaservan/vec"
)

// RuntimeExperiment captures all the information needed to
// evaluate a live deployment
type RuntimeExperiment struct {
	NumAds                   int     `json:"num_ads"`
	NumFeatures              int     `json:"num_features"`
	NumTables                int     `json:"num_tables"`
	GetBucketServerMS        []int64 `json:"get_bucket_server_ms"`
	GetBucketClientMS        []int64 `json:"get_bucket_client_ms"`
	GetBucketBandwidthDownB  []int64 `json:"get_bucket_bandwidth_down_bytes"`
	GetBucketBandwidthUpB    []int64 `json:"get_bucket_bandwidth_up_bytes"`
	GetBucketBandwidthNaiveB []int64 `json:"get_bucket_bandwidth_naive_bytes"`
}

const BrokerServerID int = 0

// Client is used to store all relevant client information
type Client struct {
	ServerAddr    string
	ServerPort    string
	SessionParams *api.SessionParameters

	// SealPIR related
	// NOTE: "client" here refers to the PIR client in SealPIR
	// and is a bridge between Go and C++ code
	TablePIRClient     *sealpir.Client     // clients used to query each tables
	TablePIRKeys       *sealpir.GaloisKeys // keys used to query each hash table
	TableNumBuckets    map[int]int         // number of hash buckets in each table
	TableHashFunctions map[int]*anns.LSH   // LSH functions used to query tables

	// client's profile feature vector
	Profile    *vec.Vec
	Experiment *RuntimeExperiment
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

		client.TablePIRClient = c
		client.TablePIRKeys = keys

		client.TableNumBuckets = res.TableNumBuckets
		client.TableHashFunctions = res.TableHashFunctions
	} else {
		panic("no table PIR params provided")
	}

	client.SessionParams = &api.SessionParameters{
		SessionID:   res.SessionID,
		NumFeatures: res.NumFeatures,
		NumTables:   res.NumTables,
		NumAds:      res.NumAds,
	}

	// TODO: this is kind of a hack that is only ok for experiments
	// gen profile here once the client knows how many features the server is running
	client.Profile = vec.NewRandomVec(res.NumFeatures, -50, 50)

	// init the experiment
	client.Experiment.NumAds = res.NumAds
	client.Experiment.NumFeatures = res.NumFeatures
	client.Experiment.NumTables = res.NumTables
}

func (client *Client) SendPIRKeys() {

	args := &api.SetKeysArgs{}
	res := &api.SetKeysResponse{}

	args.TableDBGaloisKeys = client.TablePIRKeys

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

	client.TablePIRClient.Free()

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
	for tableIndex := 0; tableIndex < client.SessionParams.NumTables; tableIndex++ {

		h := client.TableHashFunctions[tableIndex]
		elemIndex := h.Digest(client.Profile).Int64()

		c := client.TablePIRClient
		index := c.GetFVIndex(elemIndex)
		query := c.GenQuery(index)

		qargs.Queries[tableIndex] = query
	}

	if !client.call("Server.PrivateBucketQuery", &qargs, &qres) {
		panic("failed to make RPC call")
	}

	// recover the result
	// TODO: actually use the recovered result(s) to recover the NN
	for tableIndex := 0; tableIndex < client.SessionParams.NumTables; tableIndex++ {

		h := client.TableHashFunctions[tableIndex]
		elemIndex := h.Digest(client.Profile).Int64()

		c := client.TablePIRClient
		offset := c.GetFVOffset(elemIndex)
		c.Recover(qres.Answers[tableIndex][0], offset)
	}

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

	cli, err := rpc.DialHTTP("tcp", client.ServerAddr+":"+client.ServerPort)
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
