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

// Server maintains all the necessary server state
type Server struct {
	Sessions  map[int64]*ClientSession
	NumProcs  int
	KnnParams *anns.LSHParams
	KnnValues []*vec.Vec
	Knn       *anns.LSHBasedKNN

	TableDBs    map[int]*sealpir.Database // array of databases; one for each hash table
	TableParams *sealpir.Params           // array of SealPIR params; one for each hash table

	AdDb   *sealpir.Database // database of ads
	AdSize int
	NumAds int

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
	for tableIndex := 0; tableIndex < serv.KnnParams.NumTables; tableIndex++ {
		wg.Add(1)
		go func(tableIndex int) {
			defer wg.Done()

			mu.Lock()
			query := args.Queries[tableIndex]
			db := serv.TableDBs[tableIndex]
			mu.Unlock()

			res := db.Server.GenAnswer(query)

			mu.Lock()
			reply.Answers[tableIndex] = res
			mu.Unlock()

		}(tableIndex)
	}

	wg.Wait()

	idBits := math.Ceil(math.Log2(float64(serv.NumAds)))          // bits needed to describe each ad ID
	bucketSizeBits := idBits * float64(serv.KnnParams.BucketSize) // bits needed per table bucket
	idMappingBits := serv.NumAds * serv.KnnParams.NumFeatures * 8 // assume each feature is 1 byte

	// bandwidth required to send: all hash tables + mapping from ID to vector
	// observe that this is much better than sending the tables with the full vectors in each bucket
	naiveBandwidth := int64(serv.KnnParams.NumTables)*int64(bucketSizeBits)*int64(serv.NumAds) + int64(idMappingBits)
	naiveBandwidth = naiveBandwidth / 8 // bits to bytes

	reply.StatsNaiveBandwidthBytes = naiveBandwidth
	reply.StatsTotalTimeInMS = time.Now().Sub(start).Milliseconds()
	log.Printf("[Server]: processed PrivateBucketQuery request in %v ms", reply.StatsTotalTimeInMS)

	return nil
}

// PrivateAdQuery performs a PIR query for the items in the database
func (serv *Server) PrivateAdQuery(args *api.AdQueryArgs, reply *api.AdQueryResponse) error {

	start := time.Now()
	log.Printf("[Server]: received request to PrivateAdQuery")

	reply.Answer = serv.AdDb.Server.GenAnswer(args.Query)
	reply.StatsTotalTimeInMS = time.Now().Sub(start).Milliseconds()

	log.Printf("[Server]: processed single server PrivateAdQuery request in %v ms", reply.StatsTotalTimeInMS)

	return nil
}

// AdQuery performs a PIR query for the items in the database
func (serv *Server) AdQuery(args *api.AdQueryArgs, reply *api.AdQueryResponse) error {

	log.Printf("[Server]: received request to AdQuery")

	size := serv.AdDb.Server.Params.ItemBytes
	idx := int(args.Index)
	reply.Item = serv.AdDb.Bytes[size*idx : size*idx+size]

	log.Printf("[Server]: processed AdQuery request")

	return nil
}

func (serv *Server) BuildAdDatabase() {

	// SealPIR DB containing ad slots
	params := sealpir.InitParams(
		serv.NumAds,
		serv.AdSize,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		serv.NumProcs,
	)

	_, db := sealpir.InitRandomDB(params)

	serv.AdDb = db
}

// BuildKNNDataStructure initializes the KNN data structure hash tables
// and the SealPIR databases used to privately query them
func (serv *Server) BuildKNNDataStructure() {

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
	numBuckets := int(len(serv.KnnValues))

	var wg sync.WaitGroup
	var mu sync.Mutex

	// compute the min number of bytes needed to represent a bucket
	// size of each vector is dim * 8 in bits (assuming 8 bits per entry)

	// size of each feature vector
	vecBits := serv.KnnValues[0].Size() * 8 // 1 byte per coordinate

	// contents of bucket
	bucketBits := vecBits * serv.KnnParams.BucketSize

	// Vector commitment proof for dictionary keys (assume keys are 1...n)
	// using bilinear scheme of LY10 requires 48 bytes (@128 bit security)
	// see https://eprint.iacr.org/2020/419.pdf for details.
	proofBits := (48) * 8

	// divide by 8 to convert to bytes
	bytesPerBucket := (bucketBits + proofBits) / 8

	// SealPIR databases and params for each hash table
	serv.TableDBs = make(map[int]*sealpir.Database)

	serv.TableParams = sealpir.InitParams(
		numBuckets,
		bytesPerBucket,
		sealpir.DefaultSealPolyDegree,
		sealpir.DefaultSealLogt,
		sealpir.DefaultSealRecursionDim,
		serv.NumProcs,
	)

	// TODO: this is where actual data would be used
	_, db := sealpir.InitRandomDB(serv.TableParams)

	for t := 0; t < numTables; t++ {
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

func (serv *Server) LoadFeatureVectors(dbSize, numFeatures, min, max int) {

	log.Printf("[Server]: generating synthetic dataset of size %v with %v features\n", dbSize, numFeatures)

	// TODO: don't use magic constants
	// It doesn't really matter for runtime experiments but a complete system
	// should use a "real" query from the dataset because this isn't guaranteed to
	// generate a query that has any neigbors ...
	var err error
	dbValues, _, _, err := anns.GenerateRandomDataWithPlantedQueries(
		dbSize,
		numFeatures,
		float64(-50), // min value
		float64(50),  // max value
		10,           // num queries
		10,           // num NN per query
		anns.EuclideanDistance,
		20, // max distance to a neighbor
	)

	if err != nil {
		panic(err)
	}

	serv.KnnValues = dbValues

}

// for timing purposes only
func (serv *Server) GenFakeReportingToken() ([]byte, *token.SignedBlindToken) {

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
func (serv *Server) GenFakeReportingPublicMDToken() ([]byte, *token.SignedBlindTokenWithMD) {

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

func newError(err error) api.Error {
	return api.Error{
		Msg: err.Error(),
	}

}