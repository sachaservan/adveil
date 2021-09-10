package main

import (
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
	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/client"
)

// command-line arguments to run the server
var args struct {
	ServerAddr          string
	ServerPort          string
	SecurityBits        int    `default:"1024"` // e.g., 1024 RSA security; 128 for secret-sharing security
	ExperimentNumTrials int    `default:"1"`    // number of times to run this experiment configuration
	ExperimentSaveFile  string `default:"output.json"`
	AutoCloseClient     bool   `default:"true"` // close client when done
}

func main() {

	gob.Register(&anns.GaussianHash{})

	arg.MustParse(&args)

	cli := &client.Client{}
	cli.ServerAddr = args.ServerAddr
	cli.ServerPort = args.ServerPort
	cli.Experiment = &client.RuntimeExperiment{}

	// init experiment
	cli.Experiment.GetBucketServerMS = make([]int64, 0)
	cli.Experiment.GetBucketClientMS = make([]int64, 0)

	log.Printf("[Client]: waiting for server to initialize \n")

	// wait for the server(s) to finish initializing
	cli.WaitForExperimentStart()

	log.Printf("[Client]: starting experiment \n")

	log.Printf("[Client]: initializing session \n")

	cli.InitSession()

	log.Printf("[Client]: sending PIR keys to the server \n")

	cli.SendPIRKeys()

	log.Printf("[Client]: session initialized (SID = %v)\n", cli.SessionParams.SessionID)

	experimentsToDiscard := 2 // discard first couple experiments which are always slower due to server warmup
	for i := 0; i < args.ExperimentNumTrials+experimentsToDiscard; i++ {

		start := time.Now()
		_, serverMS, bandwidthUp, bandwidthDown, bandwidthNaive := cli.QueryBuckets()

		if i >= experimentsToDiscard {
			cli.Experiment.GetBucketClientMS = append(cli.Experiment.GetBucketClientMS, time.Now().Sub(start).Milliseconds())
			cli.Experiment.GetBucketServerMS = append(cli.Experiment.GetBucketServerMS, serverMS)
			cli.Experiment.GetBucketBandwidthNaiveB = append(cli.Experiment.GetBucketBandwidthNaiveB, bandwidthNaive)
			cli.Experiment.GetBucketBandwidthUpB = append(cli.Experiment.GetBucketBandwidthUpB, bandwidthUp)
			cli.Experiment.GetBucketBandwidthDownB = append(cli.Experiment.GetBucketBandwidthDownB, bandwidthDown)

			log.Printf("[Client]: bucket query took %v seconds\n", time.Now().Sub(start).Seconds())
		}

		if i >= experimentsToDiscard {
			log.Printf("[Client]: finished trial %v of %v \n", i+1-experimentsToDiscard, args.ExperimentNumTrials)
		} else {
			log.Printf("[Client]: finished warmup trial %v of %v \n", i+1, experimentsToDiscard)

		}
	}

	// write the result of the evalaution to the specified file
	experimentJSON, _ := json.MarshalIndent(cli.Experiment, "", " ")
	ioutil.WriteFile(args.ExperimentSaveFile, experimentJSON, 0644)

	// prevent client from closing until user input
	if !args.AutoCloseClient {
		log.Printf("[Client]: press enter to close")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
	}

	// terminate the client's session on the server
	cli.TerminateSessions()
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
