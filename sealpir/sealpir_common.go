package sealpir

// #cgo CFLAGS: -I"/usr/local/include"
// #cgo LDFLAGS: ./../C/libsealwrapper.a -lstdc++ -lm -lseal
// #include <stdlib.h>
// #include "../C/wrapper.h"
import "C"
import (
	"math"
	"sync"
	"unsafe"
)

// SealPIR parameters: see C/main.cpp
const DefaultSealPolyDegree = 2048
const DefaultSealLogt = 12
const DefaultSealRecursionDim = 2

type Params struct {
	Pointer      unsafe.Pointer
	NumItems     int
	ItemBytes    int
	NParallelism int
	Logt         int
	Dim          int
	PolyDegree   int
	ItemChunks   int
}

type SerializedParams struct {
	NumItems     int
	ItemBytes    int
	NParallelism int
	Logt         int
	Dim          int
	PolyDegree   int
}

type ExpandedQuery struct {
	Pointer unsafe.Pointer
}

type Client struct {
	Params  *Params
	Pointer unsafe.Pointer
}

type Server struct {
	DBs    []unsafe.Pointer // one database per parallelism
	Params *Params
}

type GaloisKeys struct {
	Str      string
	ClientID uint64
}

type Query struct {
	Str            string
	ClientID       uint64
	CiphertextSize uint64
	Count          uint64
}

type Answer struct {
	Str            string
	ClientID       uint64
	CiphertextSize uint64
	Count          uint64
}

func InitServer(params *Params) *Server {

	parallelism := params.NParallelism
	dbs := make([]unsafe.Pointer, parallelism)

	for i := 0; i < parallelism; i++ {
		dbs[i] = C.init_server_wrapper(params.Pointer)
	}
	return &Server{
		DBs:    dbs,
		Params: params,
	}
}

func (server *Server) SetupDatabase(db *Database) {

	allData := db.Bytes
	bytes := server.Params.ItemBytes
	partsize := bytes * int(math.Ceil(float64(len(allData)/bytes/server.Params.NParallelism)))

	padding := make([]byte, len(allData)%partsize)
	allData = append(allData, padding...)

	// split the database into many sub-databases
	for i := 0; i < server.Params.NParallelism; i++ {
		chunkBytes := allData[i*partsize : i*partsize+partsize]
		C.setup_database(server.DBs[i], C.CString(string(chunkBytes)))
	}

}

func (client *Client) GenGaloisKeys() *GaloisKeys {
	keyPtr := C.gen_galois_keys(client.Pointer)
	keyC := (*GaloisKeysCStruct)(unsafe.Pointer(keyPtr))

	key := GaloisKeys{
		Str: C.GoStringN(keyC.StrPtr, C.int(keyC.StrLen)),
	}
	key.ClientID = uint64(keyC.ClientID)

	return &key
}

func (server *Server) GenAnswerWithExpandedQuery(expandedQuery *ExpandedQuery) []*Answer {

	answers := make([]*Answer, server.Params.NParallelism)

	var wg sync.WaitGroup
	for i := 0; i < server.Params.NParallelism; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ansPtr := C.gen_answer_with_expanded_query(server.DBs[i], expandedQuery.Pointer)
			answerC := (*AnswerCStruct)(unsafe.Pointer(ansPtr))

			answer := Answer{
				Str: C.GoStringN(answerC.StrPtr, C.int(answerC.StrLen)),
			}

			answer.CiphertextSize = uint64(answerC.CiphertextSize)
			answer.Count = uint64(answerC.Count)

			answers[i] = &answer
		}(i)
	}

	wg.Wait()

	return answers
}

// SerializeParams returns a serialized version of params
func SerializeParams(params *Params) *SerializedParams {
	ser := &SerializedParams{}
	ser.ItemBytes = params.ItemBytes
	ser.NParallelism = params.NParallelism
	ser.NumItems = params.NumItems
	ser.Dim = params.Dim
	ser.PolyDegree = params.PolyDegree
	ser.Logt = params.Logt

	return ser
}

func DeserializeParams(ser *SerializedParams) *Params {
	return InitParams(
		ser.NumItems,
		ser.ItemBytes,
		ser.PolyDegree,
		ser.Logt,
		ser.Dim,
		ser.NParallelism,
	)
}

func SerializeParamsList(paramsList []*Params) []*SerializedParams {

	serList := make([]*SerializedParams, len(paramsList))

	for i := 0; i < len(paramsList); i++ {
		serList[i] = SerializeParams(paramsList[i])
	}

	return serList
}

func SerializeParamsMap(paramsMap map[int]*Params) map[int]*SerializedParams {

	serMap := make(map[int]*SerializedParams)

	for k, v := range paramsMap {
		serMap[k] = SerializeParams(v)
	}

	return serMap
}

func DeserializeParamsList(serList []*SerializedParams) []*Params {
	list := make([]*Params, len(serList))

	for i := 0; i < len(serList); i++ {
		list[i] = DeserializeParams(serList[i])
	}

	return list
}

func DeserializeParamsMap(serMap map[int]*SerializedParams) map[int]*Params {
	m := make(map[int]*Params)

	for k, v := range serMap {
		m[k] = DeserializeParams(v)
	}

	return m
}

func (params *Params) Free() {
	C.free_params(params.Pointer)
}

func (client *Client) Free() {
	C.free_client_wrapper(client.Pointer)
}

func (server *Server) Free() {
	for i := 0; i < server.Params.NParallelism; i++ {
		C.free_server_wrapper(server.DBs[i])
	}
}
