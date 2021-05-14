package anns

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/sachaservan/vec"
)

// Data parameters
const NumberOfValues int = 1000
const NumberOfDims int = 100
const DataValueRangeMin float64 = -50
const DataValueRangeMax float64 = 50

// Test parameters
const NumberOfTables int = 10
const NumQueries int = 100
const NumNNPerQuery int = 1
const MaxDistanceToNN float64 = 2

// Hilbert-based KNN parameters
const HilbertCurveBits int = 24

// LSH parameters
const Metric = EuclideanDistance
const NumberOfProjections = 10      // aka k (larger leads to fewer false-negatives)
const ProjectionWidth float64 = 200 // aka r (larger leads to more false-positives)
const ApproximationFactor = 2       // aka c
const HashBytes = 2                 // hash function output byte

// builds a KNN datastructure over randomly generated data
// all parameters are specified at the top of the file
func genTestLSHBased() (*LSHBasedKNN, []*vec.Vec, []*vec.Vec, [][]int) {

	params := LSHParams{}
	params.NumFeatures = NumberOfDims
	params.NumProjections = NumberOfProjections
	params.ProjectionWidth = ProjectionWidth
	params.NumTables = NumberOfTables
	params.ApproximationFactor = ApproximationFactor
	params.HashBytes = HashBytes
	params.Metric = Metric

	values, queries, neighborIdxs, err := GenerateRandomDataWithPlantedQueries(
		NumberOfValues,
		NumberOfDims,
		DataValueRangeMin,
		DataValueRangeMax,
		NumQueries,
		NumNNPerQuery,
		Metric,
		MaxDistanceToNN)

	if err != nil {
		panic(err)
	}

	knn, _ := NewLSHBased(&params)
	knn.BuildWithData(values)

	return knn, values, queries, neighborIdxs
}

func setup() {
	rand.Seed(time.Now().Unix())
}

func TestBuild(t *testing.T) {
	rand.Seed(time.Now().Unix())

	genTestLSHBased()
}

// executes queries over the KNN data structure with brute-force search over
// the returned candidates and majority candidate variant
//
// run with 'go test -v -run TestQueryLSH' to see log outputs.
func TestQueryLSH(t *testing.T) {
	setup()

	t.Logf("Collecting query stats...\n")

	var distanceFunction DistanceFunction
	switch Metric {
	case EuclideanDistance:
		distanceFunction = vec.EuclideanDistance
		break
	case HammingDistance:
		distanceFunction = vec.HammingDistance
		break
	}

	lsh, values, queries, neighborIdxs := genTestLSHBased()

	maxDistance := 0.0
	avgDistance := 0.0
	for i, q := range queries {
		for j := 0; j < NumNNPerQuery; j++ {
			dist := distanceFunction(q, values[neighborIdxs[i*NumNNPerQuery][j]])
			maxDistance = math.Max(maxDistance, dist)
			avgDistance += dist
		}
	}

	avgDistance /= float64(len(queries) * NumNNPerQuery)

	if maxDistance > MaxDistanceToNN {
		t.Fatalf("Max distance in planted queries is too large! %v > %v", maxDistance, MaxDistanceToNN)
	}

	t.Logf("[Stats] Avg distance between queries and planted data %v\n", avgDistance)
	t.Logf("[Stats] Max distance between queries and planted data %v\n", maxDistance)
	t.Logf("[Stats] Number of buckets in table %v\n", len(lsh.Tables[0].Buckets))
	t.Log()

	recallBFSCandidate := 0.0 // number of queries where the NN was found
	recallFreqCandidate := 0.0
	allCandidatesCount := 0.0

	for n, query := range queries {

		// query the LSH data structure and get back a set of candidate points
		candidates, ids := lsh.Query(query)

		// expect at least a few candidates if params were set up correctly
		if len(candidates) == 0 {
			continue
		}

		allCandidates := make(map[int]bool)
		for i := range candidates {
			allCandidates[ids[i]] = true
		}

		// find the best candidate in the set
		nearestFoundBest := BruteForceSearchTopKNN(query, candidates, ids, distanceFunction, NumNNPerQuery)

		// return the most frequent candidates
		nearestFoundSorted, err := GetSortedCandidates(candidates, ids)

		for i := 0; i < NumNNPerQuery; i++ {

			actualNeighborIdx := neighborIdxs[n][i]
			actualDistance := distanceFunction(query, values[actualNeighborIdx])

			if len(nearestFoundBest) > i {
				distanceBest := distanceFunction(query, nearestFoundBest[i])
				if distanceBest <= ApproximationFactor*actualDistance {
					recallBFSCandidate++
				}
			}

			if err == nil && len(nearestFoundSorted) > i {
				distanceSorted := distanceFunction(query, nearestFoundSorted[i])
				if distanceSorted <= ApproximationFactor*actualDistance {
					recallFreqCandidate++
				}
			}
		}

		allCandidatesCount += float64(len(allCandidates))
	}

	recallBFSCandidate /= float64(len(queries) * NumNNPerQuery)
	recallFreqCandidate /= float64(len(queries) * NumNNPerQuery)

	precisionBestCandidate := float64(len(queries)*NumNNPerQuery) * recallBFSCandidate / allCandidatesCount
	precisionMajCandidate := float64(len(queries)*NumNNPerQuery) * recallFreqCandidate / allCandidatesCount

	t.Logf("[Best Candidate] Recall  %v\n", recallBFSCandidate)
	t.Logf("[Maj Candidate]  Recall  %v\n", recallFreqCandidate)
	t.Logf("[Best Candidate] Precision %v\n", precisionBestCandidate)
	t.Logf("[Maj Candidate]  Precision %v\n", precisionMajCandidate)

}
