package main

import (
	"encoding/gob"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/token"

	"github.com/alexflint/go-arg"
)

// MetricsExperiment contains results for verifying tokens
type MetricsExperiment struct {
	NumReports                int     `json:"num_reports"`
	TokenStorage              int64   `json:"token_storage_bytes"`
	RedeemPublicProcessingMS  []int64 `json:"oken_redeem_public_processing_ms"`
	RedeemPrivateProcessingMS []int64 `json:"token_redeem_private_processing_ms"`
}

func main() {

	gob.Register(&anns.GaussianHash{})

	// command-line arguments to the server
	var args struct {
		// port on which to run
		Port string `default:"8000"`

		// number of cores to use
		NumProcs int `default:"40"`

		// database parameters
		NumAds      int `default:"10000"`
		AdSizeBytes int `default:"1000"`

		NoANNS bool `default:"false"`

		// knn parameters
		NumFeatures     int `default:"50"`
		NumTables       int `default:"10"`
		NumProjections  int `default:"5"`
		DataMin         int `default:"-50"`
		DataMax         int `default:"50"`
		ProjectionWidth int `default:"300"`

		// only for reporting experiment
		JustReporting       bool   `default:"false"`
		NumTrials           int    `default:"1"`
		NumReports          int    `default:"1024"`
		ExperimentNumTrials int    `default:"1"`
		ExperimentSaveFile  string `default:"output.json"`
	}

	// parse the command line arguments
	arg.MustParse(&args)

	// construct the parameter struct for KNN data structure
	params := &anns.LSHParams{}
	params.NumFeatures = args.NumFeatures
	params.NumProjections = args.NumProjections
	params.ProjectionWidth = float64(args.ProjectionWidth)
	params.NumTables = args.NumTables
	params.Metric = anns.EuclideanDistance

	// TODO: don't have magic constants
	params.ApproximationFactor = 2
	params.BucketSize = 1
	params.HashBytes = 4

	// make the server struct
	server := &Server{
		Sessions:  make(map[int64]*ClientSession),
		KnnParams: params,
		Ready:     false,
		NumAds:    args.NumAds,
		AdSize:    args.AdSizeBytes,
		ANNS:      !args.NoANNS,
		NumProcs:  args.NumProcs,
	}

	go func(server *Server) {
		// hack to ensure server starts before this completes
		time.Sleep(100 * time.Millisecond)

		if server.ANNS {
			log.Println("[Server]: loading feature vectors")
			server.loadFeatureVectors(args.NumAds, args.NumFeatures, args.DataMin, args.DataMax)

			log.Println("[Server]: building KNN data struct")
			server.buildKNNDataStructure()
		}

		log.Println("[Server]: building Ad databases")
		server.buildAdDatabase()

		log.Println("[Server]: server is ready")
		server.Ready = true

	}(server)

	// start the server in the background
	// will set ready=true when ready to take API calls
	go killLoop(server)
	startServer(server, args.Port)
}

// kill server when Killed flag set
func killLoop(server *Server) {
	for !server.Killed {
		time.Sleep(100 * time.Millisecond)
	}

	server.Listener.Close()
}

func startServer(server *Server, port string) {

	rpc.HandleHTTP()
	rpc.RegisterName("Server", server)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	log.Println("[Server]: waiting for clients on port " + port)

	server.Listener = listener
	http.Serve(listener, nil)
}

func (server *Server) loadFeatureVectors(dbSize, numFeatures, min, max int) {

	log.Printf("[Server]: generating synthetic dataset of size %v with %v features\n", dbSize, numFeatures)

	// TODO: don't use magic constants?
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

	server.KnnValues = dbValues

}

// for timing purposes only
func (server *Server) genFakeReportingPrivateToken() *token.SignedToken {

	tokenPk := token.PublicKey{
		Pk: server.RPk.Pk,
	}

	tokenSk := token.SecretKey{
		Sk: server.RSk.Sk,
	}

	t, T, _, _, err := tokenPk.NewPrivateMDToken()
	if err != nil {
		panic(err)
	}

	W := tokenSk.PrivateMDSign(T, false)
	return &token.SignedToken{T: t, W: W}
}

// for timing purposes only
func (server *Server) genFakeReportingPublicToken() *token.SignedToken {

	tokenPk := token.PublicKey{
		Pk: server.RPk.Pk,
	}

	tokenSk := token.SecretKey{
		Sk: server.RSk.Sk,
	}

	md := make([]byte, 4)
	t, T, _, err := tokenPk.NewPublicMDToken(md)
	if err != nil {
		panic(err)
	}

	W, proof := tokenSk.PublicMDSign(T, md)
	return &token.SignedToken{T: t, W: W, Proof: proof}
}
