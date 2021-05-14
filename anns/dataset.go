package anns

import (
	"math"
	"math/rand"

	"github.com/sachaservan/vec"
)

const trainDatasetSuffix = "_train.csv"
const testDatasetSuffix = "_test.csv"
const neighborsDatasetSuffix = "_neighbors.csv"

// GenerateRandomDataWithPlantedQueries generates random data
// with specified parameters and plants datapoints around queries.
//
// num: specifies the number of values to generate in total
// dim: dinension of the data vectors generated
// valueMin: min value (int) in each component of the vector
// valueMax: maximum value (int) in each component of the vector
// numQueries: number of queries to generate over the data
// numNN: min number of "planted" neighbors for each query
// maxNeighborDistance: maximum euclidean distance between each query
// minDistanceThreshold: minimum distance to all non-neighbor points from every query
// and planted neighbors
//
// returns (data, queries, planted)
func GenerateRandomDataWithPlantedQueries(
	num int,
	dim int,
	valueMin float64,
	valueMax float64,
	numQueries int,
	numNN int,
	metric DistanceMetric,
	maxNeighborDistance float64) ([]*vec.Vec, []*vec.Vec, [][]int, error) {

	// generate random valued vectors
	values := make([]*vec.Vec, num+numQueries*numNN)

	for i := range values {
		values[i] = vec.NewRandomVec(dim, valueMin, valueMax)
	}

	queries := make([]*vec.Vec, numQueries)
	plantedIdxs := make([][]int, numQueries*numNN)

	for j := 0; j < numQueries; j++ {
		queries[j] = vec.NewRandomVec(dim, valueMin, valueMax)
		plantedIdxs[j*numNN] = make([]int, numNN)

		for k := 0; k < numNN; k++ {
			neighbor := PerturbVector(queries[j].Copy(), metric, maxNeighborDistance)
			values = append(values, neighbor)
			plantedIdxs[j*numNN][k] = len(values) - 1
		}
	}

	return values, queries, plantedIdxs, nil
}

func PerturbVector(v *vec.Vec, metric DistanceMetric, maxDistance float64) *vec.Vec {

	switch metric {
	case EuclideanDistance:
		r := math.Sqrt(maxDistance * maxDistance / float64(v.Size()))

		// fixed-point approximation hack; scale by 1000.0
		fp := int(r * 1000.0)

		for coord := 0; coord < v.Size(); coord++ {
			shift := float64(rand.Intn(fp)) / 1000.0

			if rand.Intn(2) == 0 {
				shift *= -1
			}

			v.AddToCoord(shift, coord)
		}

		break

	case HammingDistance:
		for i := 0; i < int(maxDistance); i++ {
			coord := rand.Intn(v.Size())
			val := v.Coord(coord)
			v.SetValueToCoord(1-val, coord) // flip the bit
		}
		break
	}

	return v
}
