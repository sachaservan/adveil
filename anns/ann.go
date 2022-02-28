package anns

import (
	"github.com/sachaservan/vec"
)

// KNN data structure provides a way to construct it and query
type KNN interface {
	BuildWithData(data []*vec.Vec, maxBucketSize int) ([]*vec.Vec, error)
	Query(query *vec.Vec, k int) ([]*vec.Vec, error)
}

// DistanceMetric specifies the distance LSH should be sensitive to
type DistanceMetric int

const (
	// HammingDistance specifies a hamming weight distance metric
	HammingDistance DistanceMetric = iota
	// EuclideanDistance specifies a euclidean (l2) distance metric
	EuclideanDistance
)

// Table stores a set of hash buckets
type Table struct {
	Buckets map[string]map[int]bool // hash table for all buckets per LSH table
}

// LSHBasedKNN is a data structure that uses GaussianHash to
// hash a set of points into buckets for nearest neighbor search
type LSHBasedKNN struct {
	Params *LSHParams     // parameters used in constructing the data structure
	Data   []*vec.Vec     // copy of the original data vectors
	Tables map[int]*Table // array of hash tables storing the data
	Hashes map[int]*LSH   // hash function for each of the numTables tables
}

// DistanceFunction returns the distance between p and q according to a distance metric
type DistanceFunction func(p, q *vec.Vec) float64

// LSHParams encapsulates the parameters used in constructing the LSH-based data structure
type LSHParams struct {
	NumFeatures         int            `json:"num_features"`         // number of features in each data point
	NumTables           int            `json:"num_tables"`           // number of hash tables to construct
	NumProbes           int            `json:"num_probes"`           // number of probes per hash table
	NumProjections      int            `json:"num_projections"`      // number of hash functions to compose
	ApproximationFactor float64        `json:"approximation_factor"` // lsh approx factor
	ProjectionWidth     float64        `json:"projection_width"`     // width of each hash value (only applies to certain distance metrics)
	HashBytes           int            `json:"hash_bytes"`           // hash function output bytes
	Metric              DistanceMetric `json:"distance_metric"`
	BucketSize          int            `json:"bucket_size"` // max bucket size in each hash table
}

// NewLSHBased generates a new KNN datastructure based on LSH
// using the specified paramters
// see lsh_nn.go for details
func NewLSHBased(params *LSHParams) (*LSHBasedKNN, error) {

	knn := &LSHBasedKNN{}
	knn.Params = params

	// initialize a new set of hashes
	knn.Hashes = make(map[int]*LSH)
	knn.Tables = make(map[int]*Table)
	for i := 0; i < knn.Params.NumTables; i++ {
		switch knn.Params.Metric {
		case EuclideanDistance:
			knn.Hashes[i] = NewEuclideanLSH(knn.Params.NumFeatures, knn.Params.ProjectionWidth, knn.Params.NumProjections)
		}
	}

	return knn, nil
}
