package anns

import (
	"math/rand"

	"github.com/sachaservan/vec"
)

// GenerateRandomDataWithPlantedQueries generates random data
// with specified parameters and plants datapoints around queries:
//
// num: specifies the number of values to generate in total
// dim: dimension of the data being generated
// valueMin: min value (int) in each component of the vector
// valueMax: maximum value (int) in each component of the vector
// numQueries: number of queries to generate over the data
// numNN: min number of "planted" neighbors for each query
// metric: distance metric used to compute a neighbor
// maxNeighborDistance: maximum euclidean distance between each query
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

	for i := 0; i < num; i++ {
		values[i] = vec.NewRandomVec(dim, valueMin, valueMax)
	}

	queries := make([]*vec.Vec, numQueries)
	plantedIdxs := make([][]int, numQueries)

	for j := 0; j < numQueries; j++ {
		queries[j] = vec.NewRandomVec(dim, valueMin, valueMax)
		plantedIdxs[j] = make([]int, numNN)

		for k := 0; k < numNN; k++ {
			neighbor := PerturbVector(queries[j].Copy(), metric, maxNeighborDistance)
			values[num+j*numNN+k] = neighbor
			plantedIdxs[j][k] = len(values) - 1
		}
	}

	return values, queries, plantedIdxs, nil
}

func PerturbVector(v *vec.Vec, metric DistanceMetric, maxDistance float64) *vec.Vec {

	switch metric {
	case EuclideanDistance:
		r := getRandomUnitVector(v.Size())
		r.Scale(maxDistance)
		v.Add(r)
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

func getRandomUnitVector(dim int) *vec.Vec {

	coords := make([]float64, dim)
	for i := 0; i < dim; i++ {
		coords[i] = rand.NormFloat64()
	}

	return vec.NewVec(coords).Normalize()
}
