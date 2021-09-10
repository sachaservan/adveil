package main

import (
	"encoding/gob"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"time"

	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/server"

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
	serv := &server.Server{
		Sessions:  make(map[int64]*server.ClientSession),
		KnnParams: params,
		Ready:     false,
		NumAds:    args.NumAds,
		AdSize:    args.AdSizeBytes,
		NumProcs:  args.NumProcs,
	}

	go func(serv *server.Server) {
		// hack to ensure server starts before this completes
		time.Sleep(100 * time.Millisecond)

		log.Println("[Server]: loading feature vectors")
		serv.LoadFeatureVectors(args.NumAds, args.NumFeatures, args.DataMin, args.DataMax)

		log.Println("[Server]: building KNN data struct")
		serv.BuildKNNDataStructure()

		log.Println("[Server]: building Ad databases")
		serv.BuildAdDatabase()

		log.Println("[Server]: server is ready")
		serv.Ready = true

	}(serv)

	// start the server in the background
	// will set ready=true when ready to take API calls
	go killLoop(serv)
	startServer(serv, args.Port)
}

// kill server when Killed flag set
func killLoop(serv *server.Server) {
	for !serv.Killed {
		time.Sleep(100 * time.Millisecond)
	}

	serv.Listener.Close()
}

func startServer(serv *server.Server, port string) {

	rpc.HandleHTTP()
	rpc.RegisterName("Server", serv)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	log.Println("[Server]: waiting for clients on port " + port)

	serv.Listener = listener
	http.Serve(listener, nil)
}
