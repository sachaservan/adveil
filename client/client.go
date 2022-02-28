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
	NumCategories            int     `json:"num_categories"`
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
		SessionID:     res.SessionID,
		NumFeatures:   res.NumFeatures,
		NumTables:     res.NumTables,
		NumCategories: res.NumCategories,
		NumProbes:     res.NumProbes,
		NumTableDBs:   res.NumTableDBs,
	}

	// TODO: this is kind of a hack that is only ok for experiments
	// gen profile here once the client knows how many features the server is running
	client.Profile = vec.NewRandomVec(res.NumFeatures, -50, 50)

	// init the experiment
	client.Experiment.NumCategories = res.NumCategories
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

	allIndices := make([]int64, 0)

	// query each hash table for the bucket that collides with the
	// client's profile feature vector under the server-provided LSH function
	qargs.Queries = make(map[int]*sealpir.Query)
	for tableIndex := 0; tableIndex < client.SessionParams.NumTables; tableIndex++ {

		h := client.TableHashFunctions[tableIndex]

		numProbes := client.SessionParams.NumProbes
		for probeIndex := 0; probeIndex < numProbes; probeIndex++ {

			// simulate LSH-multiprobing perturbation
			q := client.Profile.Copy()
			noise := vec.NewRandomVec(client.Profile.Size(), 0, 255)
			q.Add(noise)

			elemIndex := h.Digest(q).Int64()
			allIndices = append(allIndices, elemIndex)

			c := client.TablePIRClient
			index := c.GetFVIndex(elemIndex)
			query := c.GenQuery(index)
			qargs.Queries[tableIndex*numProbes+probeIndex] = query
		}
	}

	// extra databases introduce to account for partial batch retrieval failures
	numQueries := client.SessionParams.NumTables * client.SessionParams.NumProbes
	diff := client.SessionParams.NumTableDBs - numQueries
	for extra := 0; extra < diff; extra++ {

		// simulate LSH-multiprobing perturbation
		q := client.Profile.Copy()
		noise := vec.NewRandomVec(client.Profile.Size(), 0, 255)
		q.Add(noise)

		h := client.TableHashFunctions[0]
		elemIndex := h.Digest(q).Int64()
		allIndices = append(allIndices, elemIndex)

		c := client.TablePIRClient
		index := c.GetFVIndex(elemIndex)
		query := c.GenQuery(index)
		qargs.Queries[numQueries+extra] = query
	}

	if !client.call("Server.PrivateBucketQuery", &qargs, &qres) {
		panic("failed to make RPC call")
	}

	// recover the result
	// TODO: actually use the recovered result(s) to recover the NN
	for dbIndex := 0; dbIndex < client.SessionParams.NumTableDBs; dbIndex++ {
		elemIndex := allIndices[dbIndex]
		c := client.TablePIRClient
		offset := c.GetFVOffset(elemIndex)
		c.Recover(qres.Answers[dbIndex][0], offset)
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
