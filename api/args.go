package api

import (
	"github.com/sachaservan/adveil/anns"
	"github.com/sachaservan/adveil/sealpir"
)

// Error is provided as a response to API queries
type Error struct {
	Msg string
}

// AdQueryArgs arguments to a bucket hash PIR query
type AdQueryArgs struct {
	Query *sealpir.Query // private PIR query
	Index int64          // non-private query
}

// AdQueryResponse response to a bucket hash PIR query
type AdQueryResponse struct {
	Error              Error
	Answer             []*sealpir.Answer // private PIR query
	Item               []byte            // non-private query
	StatsTotalTimeInMS int64
}

// BucketQueryArgs arguments to a bucket PIR query
type BucketQueryArgs struct {
	Queries map[int]*sealpir.Query // one query per hash table
}

// BucketQueryResponse response to a bucket PIR query
type BucketQueryResponse struct {
	Error                    Error
	Answers                  map[int][]*sealpir.Answer
	StatsNaiveBandwidthBytes int64 // bandwidth of performing naive (send entire database over) PIR
	StatsTotalTimeInMS       int64
}

// SetKeysArgs for setting SealPIR galois keys
type SetKeysArgs struct {
	TableDBGaloisKeys *sealpir.GaloisKeys
}

// SetKeysResponse if error occurs
type SetKeysResponse struct {
	Error Error
}

// InitSessionArgs initializes a new experiment session
type InitSessionArgs struct{}

// InitSessionResponse response to a client following session creation
type InitSessionResponse struct {
	SessionParameters
	Error Error

	TableNumBuckets    map[int]int               // number of hash buckets in each table
	TableHashFunctions map[int]*anns.LSH         // LSH functions used to query tables
	TablePIRParams     *sealpir.SerializedParams // SealPIR params for each hash table

	StatsTotalTimeInMS int64
}

// TerminateSessionArgs used by client to kill the server (useful for experiments)
type TerminateSessionArgs struct{}

// TerminateSessionResponse  response to clients terminate session call
type TerminateSessionResponse struct{}

// WaitForExperimentArgs is used by the client to wait until the experiment starts
// before making API calls
type WaitForExperimentArgs struct{}

// WaitForExperimentResponse is used to signal to the client that server is ready
type WaitForExperimentResponse struct{}

// SessionParameters contains all the metadata information
// needed for a client to issue PIR queries
type SessionParameters struct {
	SessionID     int64
	NumFeatures   int // number of features in each feature vector
	NumCategories int // number of ads in total
	NumTables     int // number of hash tables
	NumProbes     int // number of probes per hash table
	NumTableDBs   int // number of databases representing hash tables to query
}
