package main

import (
	"log"
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

	TableDBs    map[int]*sealpir.Database // array of databases; one for each hash table
	TableParams map[int]*sealpir.Params   // array of SealPIR params; one for each hash table

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
	var mu sync.Mutex

	// compute the min number of bytes needed to represent a bucket
	// size of each vector is dim * 8 in bits (assuming 8 bits per entry)

	// size of each feature vector
	vecBits := server.KnnValues[0].Size() * 8 // 1 byte per coordinate

	// contents of bucket
	bucketBits := vecBits * server.KnnParams.BucketSize

	// Vector commitment proof for dictionary keys
	// using bilinear scheme of LY10 requires 48 bytes (@128 bit security)
	// see https://eprint.iacr.org/2020/419.pdf for details
	// Trusted setup required to sign all public elements (1 element per key);
	// This is required solely for efficiency since we don't want the client
	// to download all public parameters.
	// In total:
	// 		48 bytes for proof for the vector commitment over the dictionary keys;
	//      48 bytes for public params element associated with the dictionary key;
	//      32 bytes for signature on public params element.
	proofBits := (48 + 48 + 32) * 8

	// divide by 8 to convert to bytes
	bytesPerBucket := (bucketBits + proofBits) / 8

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
			// TODO: this is where actual data would be used
			_, db := sealpir.InitRandomDB(params)
			mu.Lock()
			server.TableParams[t] = params
			server.TableDBs[t] = db
			mu.Unlock()
		}(t)
	}
	wg.Wait()

	// SealPIR database for the mapping from ID to feature vector
	params = sealpir.InitParams(
		len(server.KnnValues),
		bytesPerBucket,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		server.NumProcs, // divide database into NumTables separate databases
	)
}

func newError(err error) api.Error {
	return api.Error{
		Msg: err.Error(),
	}

}
