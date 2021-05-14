package main

import (
	"adveil/anns"
	"adveil/cmd/api"
	"adveil/cmd/sealpir"
	"adveil/elgamal"
	"crypto/elliptic"
	"log"
	"math"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/sachaservan/vec"
)

// Server maintains all the necessary server state
type Server struct {
	OtherServerAddr string
	OtherServerPort string
	Sessions        map[int64]*ClientSession
	NumProcs        int
	KnnParams       *anns.LSHParams
	KnnValues       []*vec.Vec
	Knn             *anns.LSHBasedKNN

	IDtoVecRedundancy int                 // redundancy required for batch-PIR accuracy
	IDtoVecDB         []*sealpir.Database // mapping of ID to vector
	IDtoVecParams     []*sealpir.Params

	TableDBs    []*sealpir.Database // array of databases; one for each hash table
	TableParams []*sealpir.Params   // array of SealPIR params; one for each hash table

	AdDb   *sealpir.Database // database of ads
	AdSize int
	NumAds int

	ANNS bool // set to false to not build ANNS data structure

	// reporting public/secret keys
	RPk *elgamal.PublicKey
	RSk *elgamal.SecretKey

	Listener net.Listener
	Ready    bool // true when server has initialized
	Killed   bool // true if server killed
}

// WaitForExperiment is used to signal to a waiting client that the server has finishied initializing
func (server *Server) WaitForExperiment(args *api.WaitForExperimentArgs, reply *api.WaitForExperimentResponse) error {

	for !server.Ready {
		time.Sleep(1 * time.Second)
	}

	return nil
}

// PrivateBucketQuery performs a PIR query for the items in the database
func (server *Server) PrivateBucketQuery(args *api.BucketQueryArgs, reply *api.BucketQueryResponse) error {

	start := time.Now()
	reply.StartsStartTime = time.Now().Unix()

	log.Printf("[Server]: received request to PrivateBucketQuery\n")

	reply.Answers = make([][]*sealpir.Answer, server.KnnParams.NumTables)

	var wg sync.WaitGroup
	for tableIndex := 0; tableIndex < server.KnnParams.NumTables; tableIndex++ {
		wg.Add(1)
		go func(tableIndex int) {
			defer wg.Done()
			reply.Answers[tableIndex] = server.TableDBs[tableIndex].Server.GenAnswer(args.Queries[tableIndex])
		}(tableIndex)
	}

	wg.Wait()

	reply.StatsTotalTimeInMS = time.Now().Sub(start).Milliseconds()
	log.Printf("[Server]: processed PrivateBucketQuery request in %v ms", reply.StatsTotalTimeInMS)

	return nil
}

// PrivateMappingQuery performs a batch PIR query to recover the mapping of IDs to vectors
func (server *Server) PrivateMappingQuery(args *api.MappingQueryArgs, reply *api.MappingQueryResponse) error {

	start := time.Now()
	reply.StartsStartTime = time.Now().Unix()

	log.Printf("[Server]: received request to PrivateMappingQuery\n")

	reply.Answers = make([][]*sealpir.Answer, server.IDtoVecRedundancy)

	var wg sync.WaitGroup
	for m := 0; m < server.IDtoVecRedundancy; m++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			reply.Answers[i] = server.IDtoVecDB[i].Server.GenAnswer(args.Queries[i])
		}(m)
	}

	wg.Wait()

	reply.StatsTotalTimeInMS = time.Now().Sub(start).Milliseconds()

	log.Printf("[Server]: processed PrivateMappingQuery request in %v ms", reply.StatsTotalTimeInMS)

	return nil
}

// PrivateAdQuery performs a PIR query for the items in the database
func (server *Server) PrivateAdQuery(args *api.AdQueryArgs, reply *api.AdQueryResponse) error {

	start := time.Now()
	log.Printf("[Server]: received request to PrivateAdQuery")

	reply.Answer = server.AdDb.Server.GenAnswer(args.Query)
	reply.StatsTotalTimeInMS = time.Now().Sub(start).Milliseconds()

	log.Printf("[Server]: processed single server PrivateAdQuery request in %v ms", reply.StatsTotalTimeInMS)

	return nil
}

// AdQuery performs a PIR query for the items in the database
func (server *Server) AdQuery(args *api.AdQueryArgs, reply *api.AdQueryResponse) error {

	log.Printf("[Server]: received request to AdQuery")

	size := server.AdDb.Server.Params.ItemBytes
	idx := int(args.Index)
	reply.Item = server.AdDb.Bytes[size*idx : size*idx+size]

	log.Printf("[Server]: processed AdQuery request")

	return nil
}

func (server *Server) buildAdDatabase() {

	// SealPIR DB containing ad slots
	params := sealpir.InitParams(
		server.NumAds,
		server.AdSize,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		server.NumProcs,
	)

	_, db := sealpir.InitRandomDB(params)

	server.AdDb = db
}

// buildKNNDataStructure initializes the KNN data structure hash tables
// and the SealPIR databases used to privately query them
func (server *Server) buildKNNDataStructure() {

	// TODO: build the PIR databases based on the actual hash tables constructed
	// by the ANNS data structure.
	//
	// Currently not implemented for evaluation purposes because it does not result
	// in worst-case data (for PIR performance) given that the hash tables are likely going to be smaller
	// (assuming capped bucket sizes) than the case where there is 1 item per bucket and n buckets.
	//
	// To evaluate over real hash tables, use the knn.BuildWithData
	// function which will populate the hash tables.

	// build a new LSH-based ANN data structure for the values
	knn, err := anns.NewLSHBased(server.KnnParams)
	if err != nil {
		panic(err)
	}

	server.Knn = knn
	numTables := server.KnnParams.NumTables
	numBuckets := int(len(server.KnnValues))

	// compute the min number of bytes needed to represent a bucket
	// size of each vector is dim * 8 in bits (assuming 8 bits per entry)

	// size of a feature vector ID
	vecIDBits := int(math.Ceil(math.Log2(float64(len(server.KnnValues)))))
	// size of each feature vector
	vecBits := server.KnnValues[0].Size() * 8
	// contents of bucket
	bucketBits := vecIDBits * server.KnnParams.BucketSize
	// CoA signature on the bucket (one curve element)
	sigBits := 256
	// divide by 8 to convert to bytes
	bytesPerBucket := (bucketBits + sigBits) / 8
	// divide by 8 to convert to bytes
	bytesPerMapping := (vecBits + sigBits) / 8

	// SealPIR databases and params for each hash table
	server.TableDBs = make([]*sealpir.Database, numTables)
	server.TableParams = make([]*sealpir.Params, numTables)

	for t := 0; t < numTables; t++ {
		params := sealpir.InitParams(
			numBuckets,
			bytesPerBucket,
			sealpir.DefaultSealPolyDegree,
			sealpir.DefaultSealLogt,
			sealpir.DefaultSealRecursionDim,
			server.NumProcs,
		)

		_, db := sealpir.InitRandomDB(params)
		server.TableParams[t] = params
		server.TableDBs[t] = db
	}

	// SealPIR database for the mapping from ID to feature vector
	params := sealpir.InitParams(
		len(server.KnnValues),
		bytesPerMapping,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		server.KnnParams.NumTables, // divide database into NumTables separate databases
	)

	server.IDtoVecDB = make([]*sealpir.Database, server.IDtoVecRedundancy)
	server.IDtoVecParams = make([]*sealpir.Params, server.IDtoVecRedundancy)

	for r := 0; r < server.IDtoVecRedundancy; r++ {
		_, db := sealpir.InitRandomDB(params)
		server.IDtoVecParams[r] = params
		server.IDtoVecDB[r] = db
	}

}

// initialize ElGamal keys used in the shuffle
func (server *Server) initElGamal() {
	curve := elliptic.P256()
	server.RPk, server.RSk = elgamal.KeyGen(curve)
}

// send an RPC request to the master, wait for the response
func (server *Server) call(rpcname string, args interface{}, reply interface{}) bool {

	cli, err := rpc.DialHTTP("tcp", server.OtherServerAddr+":"+server.OtherServerPort)
	if err != nil {
		log.Printf("RPC error %v\n", err)
		return false
	}

	defer cli.Close()

	err = cli.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	log.Printf("RPC error %v\n", err)

	return false
}

func newError(err error) api.Error {
	return api.Error{
		Msg: err.Error(),
	}

}
