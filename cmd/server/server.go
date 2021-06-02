package main

import (
	"log"
	"math"
	"net"
	"sync"
	"time"

	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/cmd/api"
	"github.com/sachaservan/adveil/cmd/sealpir"
	"github.com/sachaservan/adveil/token"

	"github.com/sachaservan/vec"
)

// Server maintains all the necessary server state
type Server struct {
	Sessions  map[int64]*ClientSession
	NumProcs  int
	KnnParams *anns.LSHParams
	KnnValues []*vec.Vec
	Knn       *anns.LSHBasedKNN

	IDtoVecDB     map[int]*sealpir.Database // each database is a mapping of ID (index) to vector
	IDtoVecParams map[int]*sealpir.Params

	TableDBs    map[int]*sealpir.Database // array of databases; one for each hash table
	TableParams map[int]*sealpir.Params   // array of SealPIR params; one for each hash table

	BucketCountProof map[int][]uint8 // bucket counts for each hash table

	AdDb   *sealpir.Database // database of ads
	AdSize int
	NumAds int

	ANNS bool // set to false to not build ANNS data structure

	// reporting public/secret keys
	RPk *token.PublicKey
	RSk *token.SecretKey

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

	log.Printf("[Server]: received request to PrivateBucketQuery\n")

	reply.Answers = make(map[int][]*sealpir.Answer)
	reply.BucketCountProof = server.BucketCountProof

	var wg sync.WaitGroup
	var mu sync.Mutex
	for tableIndex := 0; tableIndex < server.KnnParams.NumTables; tableIndex++ {
		wg.Add(1)
		go func(tableIndex int) {
			defer wg.Done()

			mu.Lock()
			query := args.Queries[tableIndex]
			db := server.TableDBs[tableIndex]
			mu.Unlock()

			res := db.Server.GenAnswer(query)

			mu.Lock()
			reply.Answers[tableIndex] = res
			mu.Unlock()

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

	reply.Answers = make(map[int][]*sealpir.Answer)

	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < server.KnnParams.NumTables; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			mu.Lock()
			query := args.Queries[i]
			db := server.IDtoVecDB[i]
			mu.Unlock()

			res := db.Server.GenAnswer(query)

			mu.Lock()
			reply.Answers[i] = res
			mu.Unlock()

		}(i)
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

	var wg sync.WaitGroup

	// (optimization): instead of computing a Merkle proof over each hash table
	// just send back the counts for each bucket; coupled with the Merkle proofs
	// for the vectors and the fact that checking bucket membership can be performed
	// using the LSH functions, this saves computation time (and communication)
	// in most practical cases as it avoids PIR over Merkle proofs
	server.BucketCountProof = make(map[int][]uint8)
	for t := 0; t < numTables; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			server.BucketCountProof[t] = make([]uint8, numBuckets)
		}(t)
	}
	wg.Wait()

	// compute the min number of bytes needed to represent a bucket
	// size of each vector is dim * 8 in bits (assuming 8 bits per entry)

	// size of a feature vector ID
	vecIDBits := int(math.Ceil(math.Log2(float64(len(server.KnnValues)))))

	// size of each feature vector
	vecBits := server.KnnValues[0].Size() * 8 // 1 byte per coordinate

	// contents of bucket
	bucketBits := vecIDBits * server.KnnParams.BucketSize

	// MerkleProof for N elements
	sigBits := vecIDBits * 256 // 256 bits for SHA256 hash

	// divide by 8 to convert to bytes
	bytesPerBucket := (bucketBits) / 8

	// divide by 8 to convert to bytes
	bytesPerMapping := (vecBits + sigBits) / 8

	// SealPIR databases and params for each hash table
	server.TableDBs = make(map[int]*sealpir.Database)
	server.TableParams = make(map[int]*sealpir.Params)

	params := sealpir.InitParams(
		numBuckets,
		bytesPerBucket,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		server.NumProcs,
	)

	for t := 0; t < numTables; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			_, db := sealpir.InitRandomDB(params)
			server.TableParams[t] = params
			server.TableDBs[t] = db
		}(t)
	}
	wg.Wait()

	// SealPIR database for the mapping from ID to feature vector
	params = sealpir.InitParams(
		len(server.KnnValues),
		bytesPerMapping,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		server.NumProcs, // divide database into NumTables separate databases
	)

	server.IDtoVecDB = make(map[int]*sealpir.Database)
	server.IDtoVecParams = make(map[int]*sealpir.Params)

	for t := 0; t < numTables; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			_, db := sealpir.InitRandomDB(params)
			server.IDtoVecParams[t] = params
			server.IDtoVecDB[t] = db
		}(t)
	}

	wg.Wait()
}

func newError(err error) api.Error {
	return api.Error{
		Msg: err.Error(),
	}

}
