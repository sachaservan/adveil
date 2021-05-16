package main

import (
	"adveil/anns"
	"bufio"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/alexflint/go-arg"
)

// command-line arguments to run the server
var args struct {
	ServerAddr          string
	ServerPort          string
	SecurityBits        int    `default:"1024"` // e.g., 1024 RSA security; 128 for secret-sharing security
	ExperimentNumTrials int    `default:"1"`    // number of times to run this experiment configuration
	ExperimentSaveFile  string `default:"output.json"`
	EvaluatePrivateANN  bool   `default:"false"` // run ANN search protocol
	EvaluateAdRetrieval bool   `default:"false"` // retrieve an ad using PIR
	AutoCloseClient     bool   `default:"true"`  // close client when done
}

func main() {

	gob.Register(&anns.GaussianHash{})

	arg.MustParse(&args)

	client := &Client{}
	client.serverAddr = args.ServerAddr
	client.serverPort = args.ServerPort
	client.experiment = &RuntimeExperiment{}

	// init experiment
	client.experiment.GetBucketServerMS = make([]int64, 0)
	client.experiment.GetBucketClientMS = make([]int64, 0)
	client.experiment.GetAdServerMS = make([]int64, 0)
	client.experiment.GetAdClientMS = make([]int64, 0)
	client.experiment.PrivateGetAdDPFServerMS = make([]int64, 0)

	log.Printf("[Client]: waiting for server to initialize \n")

	// wait for the server(s) to finish initializing
	client.WaitForExperimentStart()

	log.Printf("[Client]: starting experiment \n")

	log.Printf("[Client]: initializing session \n")

	client.InitSession()

	log.Printf("[Client]: sending PIR keys to the server \n")

	client.SendPIRKeys()

	log.Printf("[Client]: session initialized (SID = %v)\n", client.sessionParams.SessionID)

	experimentsToDiscard := 2 // discard first couple experiments which are always slower due to server warmup
	for i := 0; i < args.ExperimentNumTrials+experimentsToDiscard; i++ {

		if args.EvaluatePrivateANN {
			start := time.Now()
			_, serverMS, bandwidth := client.QueryBuckets()

			if i >= experimentsToDiscard {
				client.experiment.GetBucketClientMS = append(client.experiment.GetBucketClientMS, time.Now().Sub(start).Milliseconds())
				client.experiment.GetBucketServerMS = append(client.experiment.GetBucketServerMS, serverMS)
				client.experiment.GetBucketBandwidthB = append(client.experiment.GetBucketBandwidthB, bandwidth)

				log.Printf("[Client]: bucket query took %v seconds\n", time.Now().Sub(start).Seconds())
			}
		}

		if args.EvaluateAdRetrieval {
			start := time.Now()
			_, serverMS, serverDPFMS, bandwidth := client.PrivateQueryAd(0)

			if i >= experimentsToDiscard {
				client.experiment.PrivateGetAdClientMS = append(client.experiment.PrivateGetAdClientMS, time.Now().Sub(start).Milliseconds())
				client.experiment.PrivateGetAdServerMS = append(client.experiment.PrivateGetAdServerMS, serverMS)
				client.experiment.PrivateGetAdBandwidthB = append(client.experiment.PrivateGetAdBandwidthB, bandwidth)
				client.experiment.PrivateGetAdDPFServerMS = append(client.experiment.PrivateGetAdDPFServerMS, serverDPFMS)
				log.Printf("[Client]: private ad query took %v seconds\n", time.Now().Sub(start).Seconds())

				start = time.Now()
				_, serverMS, bandwidth = client.QueryAd(0)
				client.experiment.GetAdClientMS = append(client.experiment.GetAdClientMS, time.Now().Sub(start).Milliseconds())
				client.experiment.GetAdServerMS = append(client.experiment.GetAdServerMS, serverMS)
				client.experiment.GetAdBandwidthB = append(client.experiment.GetAdBandwidthB, bandwidth)

				log.Printf("[Client]: non-private ad query took %v seconds\n", time.Now().Sub(start).Seconds())
			}
		}

		if i >= experimentsToDiscard {
			log.Printf("[Client]: finished trial %v of %v \n", i+1-experimentsToDiscard, args.ExperimentNumTrials)
		} else {
			log.Printf("[Client]: finished warmup trial %v of %v \n", i+1, experimentsToDiscard)

		}
	}

	// write the result of the evalaution to the specified file
	experimentJSON, _ := json.MarshalIndent(client.experiment, "", " ")
	ioutil.WriteFile(args.ExperimentSaveFile, experimentJSON, 0644)

	// prevent client from closing until user input
	if !args.AutoCloseClient {
		log.Printf("[Client]: press enter to close")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
	}

	// terminate the client's session on the server
	client.TerminateSessions()
}

func randomPrime(bits int) *big.Int {
	for {
		p, err := rand.Prime(rand.Reader, bits)
		if err != nil {
			continue
		} else {
			return p
		}
	}
}
