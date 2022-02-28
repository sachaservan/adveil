package server

import (
	"log"
	"math"
	"net"
	"sync"
	"time"

	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/api"
	"github.com/sachaservan/adveil/sealpir"
	"github.com/sachaservan/adveil/token"

	"github.com/sachaservan/vec"
)

// Server maintains all the necessary state
type Server struct {
	Sessions  map[int64]*ClientSession
	NumProcs  int
	KnnParams *anns.LSHParams
	KnnValues []*vec.Vec
	Knn       *anns.LSHBasedKNN

	TableDBs    map[int]*sealpir.Database // array of databases; one for each hash table
	TableParams *sealpir.Params           // array of SealPIR params; one for each hash table

	NumCategories int

	// reporting public/secret keys
	RPk *token.PublicKey
	RSk *token.SecretKey

	Listener net.Listener
	Ready    bool // true when server has initialized
	Killed   bool // true if server killed
}

// WaitForExperiment is used to signal to a waiting client that the server has finishied initializing
func (serv *Server) WaitForExperiment(args *api.WaitForExperimentArgs, reply *api.WaitForExperimentResponse) error {

	for !serv.Ready {
		time.Sleep(1 * time.Second)
	}

	return nil
}

// PrivateBucketQuery performs a PIR query for the items in the database
func (serv *Server) PrivateBucketQuery(args *api.BucketQueryArgs, reply *api.BucketQueryResponse) error {

	start := time.Now()

	log.Printf("[Server]: received request to PrivateBucketQuery\n")

	reply.Answers = make(map[int][]*sealpir.Answer)

	var wg sync.WaitGroup
	var mu sync.Mutex
	for dbIndex := 0; dbIndex < len(serv.TableDBs); dbIndex++ {
		wg.Add(1)
		go func(dbIndex int) {
			defer wg.Done()

			mu.Lock()
			query := args.Queries[dbIndex]
			db := serv.TableDBs[dbIndex]
			mu.Unlock()

			res := db.Server.GenAnswer(query)

			mu.Lock()
			reply.Answers[dbIndex] = res
			mu.Unlock()

		}(dbIndex)
	}

	wg.Wait()

	idBits := math.Ceil(math.Log2(float64(serv.NumCategories)))          // bits needed to describe each ad ID
	bucketSizeBits := idBits * float64(serv.KnnParams.BucketSize)        // bits needed per table bucket
	idMappingBits := serv.NumCategories * serv.KnnParams.NumFeatures * 8 // assume each feature is 1 byte

	// bandwidth required to send: all hash tables + mapping from ID to vector
	// observe that this is much better than sending the tables with the full vectors in each bucket
	naiveBandwidth := int64(serv.KnnParams.NumTables)*int64(bucketSizeBits)*int64(serv.NumCategories) + int64(idMappingBits)
	naiveBandwidth = naiveBandwidth / 8 // bits to bytes

	reply.StatsNaiveBandwidthBytes = naiveBandwidth
	reply.StatsTotalTimeInMS = time.Since(start).Milliseconds()
	log.Printf("[Server]: processed PrivateBucketQuery request in %v ms", reply.StatsTotalTimeInMS)

	return nil
}

// buildKNNDataStructure initializes the KNN data structure hash tables
// and the SealPIR databases used to privately query them
func BuildKNNDataStructure(serv *Server) {

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
	knn, err := anns.NewLSHBased(serv.KnnParams)
	if err != nil {
		panic(err)
	}

	serv.Knn = knn
	numTables := serv.KnnParams.NumTables
	numProbes := serv.KnnParams.NumProbes
	numBuckets := serv.NumCategories

	var wg sync.WaitGroup
	var mu sync.Mutex

	// compute the min number of bytes needed to represent a bucket
	// size of each vector is dim * 8 in bits (assuming 8 bits per entry)

	// size of each feature vector
	vecBits := serv.KnnParams.NumFeatures * 8 // 1 byte per coordinate

	// contents of bucket
	bucketBits := vecBits * serv.KnnParams.BucketSize

	// Vector commitment proof for dictionary keys (assume keys are 1...n)
	// using bilinear scheme of LY10 requires 48 bytes (@128 bit security)
	// see https://eprint.iacr.org/2020/419.pdf for details.
	proofBits := (48) * 8

	// numProbes
	// Number of multiprobes to retrieve in each hash table
	// this impacts the number of PIR queries (and communication) but
	// amortizes the server-side processing cost of retrieving multiple
	// candidates per table.
	// By partitioning the table key space, we can query each partition
	// and retrieve an element from it (if it happens to fall into the partition).

	// each partition now becomes its own (smaller) hash table
	numTableDBs := numTables * numProbes

	// number of buckets in each partition decreases by the number of multiprobes
	numBuckets = int(math.Ceil(float64(numBuckets) / float64(numProbes)))

	// divide by 8 to convert to bytes
	bytesPerBucket := (bucketBits + proofBits) / 8

	// SealPIR databases and params for each hash table
	serv.TableDBs = make(map[int]*sealpir.Database)

	serv.TableParams = sealpir.InitParams(
		numBuckets, // size of each partition table
		bytesPerBucket,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		serv.NumProcs,
	)

	// TODO: this is where actual data would be used
	_, db := sealpir.InitRandomDB(serv.TableParams)

	for t := 0; t < numTableDBs; t++ {
		wg.Add(1)
		go func(t int) {
			defer wg.Done()
			mu.Lock()
			serv.TableDBs[t] = db
			mu.Unlock()
		}(t)
	}
	wg.Wait()
}

// for timing purposes only
func GenFakeReportingToken(serv *Server) ([]byte, *token.SignedBlindToken) {

	tokenPk := token.PublicKey{
		Pks: serv.RPk.Pks,
		Pkr: serv.RPk.Pkr,
	}

	tokenSk := token.SecretKey{
		Sks: serv.RSk.Sks,
		Skr: serv.RSk.Skr,
	}

	t, T, _, _, err := tokenPk.NewToken()
	if err != nil {
		panic(err)
	}

	W := tokenSk.Sign(T, false)
	return t, W
}

// for timing purposes only
func GenFakeReportingPublicMDToken(serv *Server) ([]byte, *token.SignedBlindTokenWithMD) {

	tokenPk := token.PublicKey{
		Pks: serv.RPk.Pks,
		Pkr: serv.RPk.Pkr,
	}

	tokenSk := token.SecretKey{
		Sks: serv.RSk.Sks,
		Skr: serv.RSk.Skr,
	}

	md := make([]byte, 4)
	t, T, _, err := tokenPk.NewPublicMDToken()
	if err != nil {
		panic(err)
	}

	W := tokenSk.PublicMDSign(T, md)
	return t, W
}
