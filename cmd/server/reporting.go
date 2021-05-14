package main

import (
	"adveil/crypto"
	"adveil/elgamal"
	"adveil/token"
	"log"
	"math/rand"
	"sync"
	"time"
)

type EncryptedReport struct {
	C     *elgamal.Ciphertext
	Token *token.SignedToken
}

// ShuffleTokensArgs consists of the vector of tokens that need to be shuffled
type ShuffleReportsArgs struct {
	Reports    []*EncryptedReport
	DLEQProofs []*crypto.Proof // proofs that each token was redeemed correctly
}

// ShuffleTokensResponse reutrns the shuffled set of tokens
type ShuffleReportsResponse struct {
	Reports []*EncryptedReport
}

// requests the other server to decrypt and shuffle all tokens
func (server *Server) runMetricsExperiment(reports []*EncryptedReport) *MetricsExperiment {

	var wg sync.WaitGroup
	batchSize := len(reports) / server.NumProcs

	// init experiment
	experiment := &MetricsExperiment{}
	experiment.NumReports = len(reports)
	experiment.ShuffleProcessingMS = make([]int64, 1)
	experiment.DecryptionProcessingMS = make([]int64, 1)
	experiment.TokenProcessingMS = make([]int64, 1)

	start := time.Now() // total time

	startRedeem := time.Now()

	tokenPk := &token.PublicKey{
		Pk: server.RPk.Pk,
	}

	tokenSk := &token.SecretKey{
		Sk: server.RSk.Sk,
	}

	// redeem all tokens
	proofs := make([]*crypto.Proof, len(reports))
	for i := 0; i <= server.NumProcs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < batchSize; j++ {
				index := i*batchSize + j

				if index >= len(reports) {
					break
				}
				report := reports[index]
				_, proofs[index] = tokenSk.RedeemAndProve(tokenPk, report.Token)
			}
		}(i)
	}

	wg.Wait()

	experiment.TokenProcessingMS[0] = time.Now().Sub(startRedeem).Milliseconds()

	args := ShuffleReportsArgs{}
	res := ShuffleReportsResponse{}
	args.Reports = reports
	args.DLEQProofs = proofs

	startShuffle := time.Now()

	// keep trying until success
	for !server.call("Server.ShuffleReports", &args, &res) {
		time.Sleep(10 * time.Millisecond)
	}

	experiment.ShuffleProcessingMS[0] = time.Now().Sub(startShuffle).Milliseconds()

	newReports := make([]*crypto.Point, len(reports))

	startDecrypt := time.Now()

	// decrypt all reports
	for i := 0; i <= server.NumProcs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < batchSize; j++ {
				index := i*batchSize + j
				if index >= len(reports) {
					break
				}

				report := res.Reports[index]
				finDec := server.RSk.Decrypt(report.C)
				newReports[index] = finDec
			}
		}(i)
	}

	wg.Wait()

	experiment.DecryptionProcessingMS[0] = time.Now().Sub(startDecrypt).Milliseconds()

	log.Printf("[Server]: finished shuffling %v reports in %v ms", len(reports), time.Now().Sub(start).Milliseconds())

	return experiment
}

// ShuffleReports shuffles the encrypted reports and returns them
func (server *Server) ShuffleReports(args *ShuffleReportsArgs, reply *ShuffleReportsResponse) error {

	log.Printf("[Server]: received ShuffleReports")

	batchSize := len(args.Reports) / server.NumProcs
	newReports := make([]*EncryptedReport, len(args.Reports))
	newIndices := rand.Perm(len(args.Reports))

	h2cObj, _ := crypto.GetDefaultCurveHash()

	var wg sync.WaitGroup
	for i := 0; i <= server.NumProcs; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < batchSize; j++ {
				index := i*batchSize + j
				if index >= len(args.Reports) {
					break
				}

				// partial decrypt and permute report location
				report := args.Reports[index]
				partDec := server.RSk.PartialDecrypt(report.C)
				report.C = partDec
				newReports[newIndices[index]] = report

				// verify token redemption proof
				if !args.DLEQProofs[index].Verify(h2cObj) {
					// TODO: this doesn't currently work!
					// log.Printf("[Server]: received DLEQ proof does not verify")
				}
			}
		}(i)
	}

	wg.Wait()

	reply.Reports = newReports

	return nil
}
