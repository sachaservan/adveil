package anns

import (
	"math/big"
	"sync"

	"github.com/ncw/gmp"
	"github.com/sachaservan/argsort"
	"github.com/sachaservan/vec"
)

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

// RandomizeBucketKeys replaces each key in the hash table
// to a new randomized key (using a universal hash)
// returns the new keys and universal hash used in the randomization
func (knn *LSHBasedKNN) RandomizeBucketKeys(uhash *UniversalHash) [][]*gmp.Int {

	newKeys := make([][]*gmp.Int, len(knn.Tables))

	var wg sync.WaitGroup

	for i, t := range knn.Tables {
		wg.Add(1)
		go func(i int, t *Table) {
			defer wg.Done()

			newKeys[i] = make([]*gmp.Int, len(t.Buckets))

			j := 0
			for k := range t.Buckets {
				// set new randomize key
				bigKey := gmp.NewInt(0).SetBytes([]byte(k))
				newKey := uhash.Digest(bigKey)
				newKeys[i][j] = newKey
				j++
			}

			sort := argsort.NewIntArgsort(newKeys[i])
			newKeys[i] = argsort.SortIntsByArray(newKeys[i], sort)

		}(i, t)
	}

	wg.Wait()

	return newKeys
}

// BuildWithData builds the data structure for the data
// using the parameters of the data structure
// hashBytes specifies the output hash length
// maxBucketSize is the max size of a bucket in a hash table (set -1 for no limit)
func (knn *LSHBasedKNN) BuildWithData(data []*vec.Vec) {

	knn.Data = data

	var wg sync.WaitGroup
	for i := 0; i < knn.Params.NumTables; i++ {

		knn.Tables[i] = &Table{
			Buckets: make(map[string]map[int]bool),
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for j, point := range data {
				digest := knn.Hashes[i].StringDigest(point)

				if knn.Tables[i].Buckets[digest] == nil {
					knn.Tables[i].Buckets[digest] = make(map[int]bool)
				}

				// add the value to the bucket
				if knn.Params.BucketSize == -1 || len(knn.Tables[i].Buckets[digest]) < knn.Params.BucketSize {
					knn.Tables[i].Buckets[digest][j] = true
				}
			}

		}(i)
	}

	wg.Wait()
}

// Query returns the set of points (and point ids)
// that appeared in the same buckets that query hashed to
func (knn *LSHBasedKNN) Query(query *vec.Vec) ([]*vec.Vec, []int) {

	candidates := make([]*vec.Vec, 0)
	ids := make([]int, 0)

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < knn.Params.NumTables; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			digest := knn.Hashes[i].StringDigest(query)

			// fmt.Printf("Digest is %v\n", digest)

			if bucket, ok := knn.Tables[i].Buckets[digest]; ok {

				for key := range bucket {
					mu.Lock()
					candidates = append(candidates, knn.Data[key])
					ids = append(ids, key)
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	return candidates, ids
}

// GetTableKeys returns the keys of each bucket for each hash table
func (knn *LSHBasedKNN) GetTableKeys() [][]string {

	var wg sync.WaitGroup

	keys := make([][]string, len(knn.Tables))

	for i, t := range knn.Tables {
		keys[i] = make([]string, len(t.Buckets))

		wg.Add(1)
		go func(i int, t *Table) {
			defer wg.Done()
			j := 0
			for k := range t.Buckets {
				keys[i][j] = k
				j++
			}
		}(i, t)
	}

	wg.Wait()

	return keys
}

// GetTableBuckets returns all buckets in each hash table
func (knn *LSHBasedKNN) GetTableBuckets(tableIndex int) [][]string {

	t := knn.Tables[tableIndex]

	// all buckets in table t
	buckets := make([][]string, 0)

	for _, b := range t.Buckets {

		// all values in the bucket
		bucket := make([]string, 0)
		for k := range b {
			bucket = append(bucket, string(big.NewInt(int64(k)).Bytes()))
		}

		// add the bucket to the list of buckets
		buckets = append(buckets, bucket)
	}

	return buckets
}

// GetTableMaxBucketSize returns the size of the largest bucket for each hash table
func (knn *LSHBasedKNN) GetTableMaxBucketSize() []int {

	maxBucketSizesPerTable := make([]int, len(knn.Tables))

	for i, t := range knn.Tables {
		maxBucketSize := 0
		for _, v := range t.Buckets {
			numKeysInBucket := len(v)
			if maxBucketSize < numKeysInBucket {
				maxBucketSize = numKeysInBucket
			}
		}

		maxBucketSizesPerTable[i] = maxBucketSize
	}

	return maxBucketSizesPerTable
}

// GetHashForTable returns the locality sensitive hash for the table
func (knn *LSHBasedKNN) GetHashForTable(t int) *LSH {
	return knn.Hashes[t]
}

// NumTables returns the number of tables in the KNN data structure
func (knn *LSHBasedKNN) NumTables() int {
	return knn.Params.NumTables
}
