package sealpir

import (
	"crypto/rand"
	"crypto/sha256"
	"math"
	"math/big"
	"testing"
)

func getTestParams() *Params {
	// test parameters
	numItems := 1 << 12
	itemBytes := 288
	polyDegree := 2048
	logt := 12
	d := 2

	return InitParams(numItems, itemBytes, polyDegree, logt, d, 1)
}

func getTestParamsParallel(nParallel int) *Params {
	// test parameters
	numItems := int(math.Ceil(1 << 12 / float64(nParallel)))
	itemBytes := 203
	polyDegree := 2048
	logt := 12
	d := 2

	return InitParams(numItems, itemBytes, polyDegree, logt, d, nParallel)
}

func TestDBBitItem(t *testing.T) {
	// test parameters
	numItems := int(math.Ceil(1<<13) / 30)
	itemBytes := 10000 // BIG item
	polyDegree := 2048
	logt := 12
	d := 2

	params := InitParams(numItems, itemBytes, polyDegree, logt, d, 30)
	InitRandomDB(params)

}

func TestInitParams(t *testing.T) {
	getTestParams()
}

func TestInitClient(t *testing.T) {
	params := getTestParams()
	client := InitClient(params, 0)

	params.Free()
	client.Free()
}

func TestInitServer(t *testing.T) {
	params := getTestParams()
	server := InitServer(params)

	params.Free()
	server.Free()
}

func TestFull(t *testing.T) {

	// replicates SealPIR/main.cpp / SealPIR/main.c

	params := getTestParams()
	client := InitClient(params, 0)

	keys := client.GenGaloisKeys()

	_, db := InitRandomDB(params)

	server := db.Server
	server.SetGaloisKeys(keys)

	elemIndexBig, _ := rand.Int(rand.Reader, big.NewInt(int64(params.NumItems)))
	elemIndex := elemIndexBig.Int64() % int64(params.NumItems)

	index := client.GetFVIndex(elemIndex)
	offset := client.GetFVOffset(elemIndex)

	query := client.GenQuery(index)

	for trial := 0; trial < 100; trial++ {
		answers := server.GenAnswer(query)
		res := client.Recover(answers[0], offset)

		bytes := int64(params.ItemBytes)

		// check that we retrieved the correct element
		for i := int64(0); i < int64(params.ItemBytes); i++ {
			if res[(offset*bytes)+i] != db.Bytes[(elemIndex*bytes)+i] {
				t.Fatalf("Main: elems %d, db %d\n",
					res[(offset*bytes)+i],
					db.Bytes[(elemIndex*bytes)+i])
			}
		}
	}

	client.Free()
	server.Free()
	params.Free()
}

func TestFullParallel(t *testing.T) {

	// replicates SealPIR/main.cpp and SealPIR/main.c

	params := getTestParamsParallel(4)
	client := InitClient(params, 0)

	keys := client.GenGaloisKeys()

	_, db := InitRandomDB(params)

	server := db.Server
	server.SetGaloisKeys(keys)

	elemIndexBig, _ := rand.Int(rand.Reader, big.NewInt(int64(params.NumItems)))
	elemIndex := elemIndexBig.Int64() % int64(params.NumItems)
	numItemsPerParallelDB := int64(math.Ceil(float64(params.NumItems / params.NParallelism)))
	queryDb := int(elemIndex / numItemsPerParallelDB)
	queryIndex := elemIndex % numItemsPerParallelDB

	index := client.GetFVIndex(queryIndex)
	offset := client.GetFVOffset(queryIndex)

	query := client.GenQuery(index)

	for i := 0; i < 100; i++ {

		answers := server.GenAnswer(query)

		res := client.Recover(answers[queryDb], offset)

		bytes := int64(params.ItemBytes)

		// check that we retrieved the correct element
		for i := int64(0); i < int64(params.ItemBytes); i++ {
			if res[(offset*bytes)+i] != db.Bytes[(elemIndex*bytes)+i] {
				t.Fatalf("Main: elems %d, db %d\n",
					res[(offset*bytes)+i],
					db.Bytes[(elemIndex*bytes)+i])
			}
		}
	}

	client.Free()
	server.Free()
	params.Free()
}

func BenchmarkHash(b *testing.B) {
	var prev [32]byte
	for i := 0; i < b.N; i++ {
		prev = sha256.Sum256(prev[:])
	}
}

func BenchmarkParallelQuery(b *testing.B) {
	// test parameters
	numItems := 1 << 12
	nParallel := 1
	itemBytes := 256
	polyDegree := 2048
	logt := 12
	d := 2

	params := InitParams(numItems, itemBytes, polyDegree, logt, d, nParallel)

	_, db := InitRandomDB(params)

	client := InitClient(params, 0)

	keys := client.GenGaloisKeys()
	db.Server.SetGaloisKeys(keys)

	elemIndexBig, _ := rand.Int(rand.Reader, big.NewInt(int64(params.NumItems)))
	elemIndex := elemIndexBig.Int64() % int64(params.NumItems)
	index := client.GetFVIndex(elemIndex)
	query := client.GenQuery(index)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		db.Server.GenAnswer(query)
	}
}
