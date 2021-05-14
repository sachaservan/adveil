package main

import (
	"adveil/anns"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/alexflint/go-arg"
	"github.com/gonum/stat"
	"github.com/sachaservan/vec"
)

type Result struct {
	AvgDistanceContextual float64 `json:"avg_dist_contextual"`
	AvgDistanceTargeted   float64 `json:"avg_dist_targeted"`
	StdDistanceContextual float64 `json:"std_dist_contextual"`
	StdDistanceTargeted   float64 `json:"std_dist_targeted"`

	AvgRecallContextual float64 `json:"avg_recall_contextual"`
	AvgRecallTargeted   float64 `json:"avg_recall_targeted"`
	StdRecallContextual float64 `json:"std_recall_contextual"`
	StdRecallTargeted   float64 `json:"std_recall_targeted"`
}

// Experiment contains all the parameters used in conducting an recall
// experiment comaparing the percentage of NN returned by LSH
// Experiment struct can be saved to a json file for further analysis
type Experiment struct {
	ComparisonResults      *Result
	NumValues              int     `json:"num_values"`
	NumFeatures            int     `json:"num_features"`
	DataValueRangeMin      float64 `json:"data_min"`
	DataValueRangeMax      float64 `json:"data_max"`
	NumQueries             int     `json:"num_queries"`
	NumNNPerQuery          int     `json:"num_nn_per_query"`
	MaxDistanceToNN        float64 `json:"max_distance_to_nn"`
	MaxDistanceFromContext float64 `json:"max_distance_from_context"`
	NumTables              int     `json:"num_tables"`
}

func compare(
	simsearch *anns.LSHBasedKNN,
	distanceFunc anns.DistanceFunction,
	values []*vec.Vec,
	users []*vec.Vec,
	contextualAds [][]*vec.Vec,
	targetedAdIdxs [][]int,
	numNN int,
	distanceMetric anns.DistanceMetric,
	maxDistanceToNN float64) *Result {

	params := simsearch.Params

	// result contains the mean and standard deviation
	// of all the comparisons performed
	result := &Result{}

	recallContextual := make([]float64, 0)
	recallTargeted := make([]float64, 0)
	avgDistanceContextual := make([]float64, 0)
	avgDistanceTargeted := make([]float64, 0)

	for n, user := range users {

		testAvgRecallContextual := 0.0
		testAvgRecallTargeted := 0.0
		testAvgDistanceToContextualCandidate := 0.0
		testAvgDistanceToTargetedCandidate := 0.0

		contextual := contextualAds[n]
		candidates, ids := simsearch.Query(user)

		nearestFoundTargetedAds := anns.BruteForceSearchTopKNN(user, candidates, ids, distanceFunc, numNN)

		for i := 0; i < numNN; i++ {
			// get the planted targeted and contextual ads
			distanceContextual := distanceFunc(user, contextual[i])

			testAvgDistanceToContextualCandidate += distanceContextual
			if distanceContextual <= params.ApproximationFactor*maxDistanceToNN {
				testAvgRecallContextual++
			}

			if i >= len(nearestFoundTargetedAds) {
				break
			}

			distanceTargetedFound := distanceFunc(user, nearestFoundTargetedAds[i])

			testAvgDistanceToTargetedCandidate += distanceTargetedFound
			if distanceTargetedFound <= params.ApproximationFactor*maxDistanceToNN {
				testAvgRecallTargeted++
			}
		}

		testAvgRecallContextual /= float64(numNN)
		testAvgRecallTargeted /= float64(numNN)

		testAvgDistanceToContextualCandidate /= float64(numNN)
		testAvgDistanceToTargetedCandidate /= float64(numNN)

		recallContextual = append(recallContextual, testAvgRecallContextual)
		recallTargeted = append(recallTargeted, testAvgRecallTargeted)
		avgDistanceContextual = append(avgDistanceContextual, testAvgDistanceToContextualCandidate)
		avgDistanceTargeted = append(avgDistanceTargeted, testAvgDistanceToTargetedCandidate)

	}

	result.AvgRecallContextual, result.StdRecallContextual = stat.MeanStdDev(recallContextual, nil)
	result.AvgRecallTargeted, result.StdRecallTargeted = stat.MeanStdDev(recallTargeted, nil)
	result.AvgDistanceContextual, result.StdDistanceContextual = stat.MeanStdDev(avgDistanceContextual, nil)
	result.AvgDistanceTargeted, result.StdDistanceTargeted = stat.MeanStdDev(avgDistanceTargeted, nil)

	return result
}

// 1. Generate a synthetic dataset with planted queries as described in Datar, Mayur, et al.
//    "Locality-sensitive hashing scheme based on p-stable distributions."
// 2. Each query and neighbor is interpreted as the "website" and contextual "ad" associated with the website
// 	  this ensures that each website is paired with a contextual ad that is close to the website's features
// 3. Generate a second set of queries that represent "users" where the
//    distance from the "user" to a "website" is maxDistanceFromContext
// 4. For each "user" query generate a planted neighbor that represents the "targeted" ad that is
//    within distance maxNeighborDistance
func generateFeatureVectors(
	num int,
	dim int,
	valueMin float64,
	valueMax float64,
	numQueries int,
	numNN int,
	maxNeighborDistance float64,
	maxDistanceFromContext float64) ([]*vec.Vec, []*vec.Vec, [][]*vec.Vec, [][]int) {

	// generate random valued vectors
	values := make([]*vec.Vec, num+numQueries*numNN)

	for i := range values {
		values[i] = vec.NewRandomVec(dim, valueMin, valueMax)
	}

	websites := make([]*vec.Vec, numQueries)
	users := make([]*vec.Vec, numQueries)
	contextualAds := make([][]*vec.Vec, numQueries*numNN)
	targetedAdIdxs := make([][]int, numQueries*numNN)

	for j := 0; j < numQueries; j++ {
		websites[j] = vec.NewRandomVec(dim, valueMin, valueMax)
		users[j] = anns.PerturbVector(websites[j].Copy(), anns.EuclideanDistance, maxDistanceFromContext)
		targetedAdIdxs[j*numNN] = make([]int, numNN)
		contextualAds[j*numNN] = make([]*vec.Vec, numNN)

		for k := 0; k < numNN; k++ {
			ctx := anns.PerturbVector(websites[j].Copy(), anns.EuclideanDistance, maxNeighborDistance)
			contextualAds[j*numNN][k] = ctx

			targeted := anns.PerturbVector(users[j].Copy(), anns.EuclideanDistance, maxNeighborDistance)

			values = append(values, targeted)
			targetedAdIdxs[j*numNN][k] = len(values) - 1

		}
	}

	return values, users, contextualAds, targetedAdIdxs
}

func main() {
	var args struct {
		SaveFileName           string  `default:"results.json"`
		DatasetSize            int     `default:"10000"`
		NumFeatures            int     `default:"100"`
		DataMax                float64 `default:"50"`
		DataMin                float64 `default:"-50"`
		NumQueries             int     `default:"200"`
		NumProjections         int     `default:"10"`
		ApproximationFactor    float64 `default:"2"`
		NumNN                  int     `default:"1"`
		MaxDistanceToNN        float64 `default:"100"`
		ProjectionWidth        float64 `default:"100"`
		MaxDistanceFromContext []float64
		NumTables              []int
	}

	arg.MustParse(&args)

	distanceMetric := anns.EuclideanDistance
	distanceFunc := vec.EuclideanDistance

	allExperiments := make([]*Experiment, 0)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, numTables := range args.NumTables {

		for _, maxDistanceFromContext := range args.MaxDistanceFromContext {

			wg.Add(1)

			go func(numTables int, maxDistanceFromContext float64) {
				defer wg.Done()

				params := &anns.LSHParams{
					NumFeatures:         args.NumFeatures,
					NumTables:           numTables,
					NumProjections:      args.NumProjections,
					ProjectionWidth:     args.ProjectionWidth,
					ApproximationFactor: args.ApproximationFactor,
					Metric:              distanceMetric,
					BucketSize:          -1,
				}

				fmt.Printf("[Info]: generating data %v\n", args.SaveFileName)
				fmt.Printf("[Info]: experiment maxDistanceFromContext %v\n", maxDistanceFromContext)

				values, users, contextualAds, targetedAdIdxs := generateFeatureVectors(
					args.DatasetSize,
					args.NumFeatures,
					args.DataMin,           // min value
					args.DataMax,           // max value
					args.NumQueries,        // num queries
					args.NumNN,             // num NN per query
					args.MaxDistanceToNN,   // max (Euclidean) distance to a neighbor
					maxDistanceFromContext, // max distance from user to context
				)

				numNN := args.NumNN

				fmt.Printf("[Info]: results will be saved to %v\n", args.SaveFileName)

				simsearch, _ := anns.NewLSHBased(params)
				simsearch.BuildWithData(values)
				fmt.Printf("[Info]: max bucket size %v\n", simsearch.GetTableMaxBucketSize()[0])

				result := compare(
					simsearch,
					distanceFunc,
					values,
					users,
					contextualAds,
					targetedAdIdxs,
					numNN,
					distanceMetric,
					args.MaxDistanceToNN,
				)

				experiment := &Experiment{
					ComparisonResults:      result,
					NumValues:              args.DatasetSize,
					NumFeatures:            args.NumFeatures,
					DataValueRangeMin:      args.DataMin,
					DataValueRangeMax:      args.DataMax,
					NumQueries:             args.NumQueries,
					NumTables:              numTables,
					MaxDistanceToNN:        args.MaxDistanceToNN,
					MaxDistanceFromContext: maxDistanceFromContext,
					NumNNPerQuery:          numNN,
				}

				mu.Lock()
				allExperiments = append(allExperiments, experiment)
				mu.Unlock()

				fmt.Printf("[Info]: (L=%v, K=%v) Recall (contextual):   %v\n", numTables, numNN, result.AvgRecallContextual)
				fmt.Printf("[Info]: (L=%v, K=%v) Recall (targeted):     %v\n", numTables, numNN, result.AvgRecallTargeted)
				fmt.Printf("[Info]: (L=%v, K=%v) Distance (contextual): %v\n", numTables, numNN, result.AvgDistanceContextual)
				fmt.Printf("[Info]: (L=%v, K=%v) Distance (targeted):   %v\n", numTables, numNN, result.AvgDistanceTargeted)

			}(numTables, maxDistanceFromContext)
		}
	}

	wg.Wait()

	file, err := json.MarshalIndent(allExperiments, "", " ")

	if err != nil {
		fmt.Printf("[Error]: %v when converting to json\n", err)
	}

	err = ioutil.WriteFile(args.SaveFileName, file, 0644)
	if err != nil {
		fmt.Printf("[Error]: %v when saving to file\n", err)
	}
}
