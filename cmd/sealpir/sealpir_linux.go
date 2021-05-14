// +build linux

package sealpir

// #cgo CFLAGS: -I"/usr/local/include"
// #cgo LDFLAGS: ./../../C/libsealwrapper.a -lstdc++ -lm -lseal
// #include <stdlib.h>
// #include "../../C/wrapper.h"package sealpir

// #cgo CFLAGS: -I"/usr/local/include"
// #cgo LDFLAGS: ./../../C/libsealwrapper.a -lstdc++ -lm -lseal
// #include <stdlib.h>
// #include "../../C/wrapper.h"
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

// QueryCStruct must match struct in wrapper.h *exactly*
type QueryCStruct struct {
	StrPtr         *C.char
	StrLen         C.ulong
	ClientID       C.ulong
	CiphertextSize C.ulong
	Count          C.ulong
}

// AnswerCStruct must match struct in wrapper.h *exactly*
type AnswerCStruct struct {
	StrPtr         *C.char
	StrLen         C.ulong
	CiphertextSize C.ulong
	Count          C.ulong
}

// GaloisKeysCStruct must match struct in wrapper.h *exactly*
type GaloisKeysCStruct struct {
	StrPtr   *C.char
	StrLen   C.ulong
	ClientID C.ulong
}

func InitParams(numItems, itemBytes, polyDegree, logt, d, nParallelism int) *Params {

	if numItems <= nParallelism {
		panic("numItems must be at least 1")
	}

	if itemBytes <= 0 {
		panic("itemBytes must be at least 1")
	}

	numChunks := 1

	// won't fit in one plaintext
	// see coefficients_per_element in pir.cpp
	if int(8*itemBytes/logt) > polyDegree {
		numChunks = int(float64(8 * itemBytes / logt))
	}

	itemBytes = int(itemBytes / numChunks)
	numItems *= numChunks

	parallelDBSize := int64(math.Ceil(float64(numItems / nParallelism)))
	cParamsPtr := C.init_params(
		C.ulong(parallelDBSize),
		C.ulong(itemBytes),
		C.ulong(polyDegree),
		C.ulong(logt),
		C.ulong(d),
	)

	return &Params{
		Logt:         logt,
		Dim:          d,
		PolyDegree:   polyDegree,
		NParallelism: nParallelism,
		NumItems:     numItems,
		ItemBytes:    itemBytes,
		Pointer:      cParamsPtr,
	}
}

func InitClient(params *Params, clientId int) *Client {

	return &Client{
		Params:  params,
		Pointer: C.init_client_wrapper(params.Pointer, C.ulong(clientId)),
	}
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

func (client *Client) GenGaloisKeys() *GaloisKeys {

	// HACK: convert SerializedGaloisKeys struct from wrapper.h into a GaloisKeys struct
	// see: https://stackoverflow.com/questions/28551043/golang-cast-memory-to-struct
	keyPtr := C.gen_galois_keys(client.Pointer)
	size := unsafe.Sizeof(GaloisKeysCStruct{})
	structMem := (*(*[1<<31 - 1]byte)(keyPtr))[:size]
	keyC := (*(*GaloisKeysCStruct)(unsafe.Pointer(&structMem[0])))

	key := GaloisKeys{
		Str: C.GoStringN(keyC.StrPtr, C.int(keyC.StrLen)),
	}
	key.ClientID = uint64(keyC.ClientID)

	return &key
}

func (server *Server) SetGaloisKeys(keys *GaloisKeys) {

	galKeysC := GaloisKeysCStruct{
		StrPtr: C.CString(keys.Str),
	}
	galKeysC.StrLen = C.ulong(len(keys.Str))
	galKeysC.ClientID = C.ulong(keys.ClientID)

	keysPtr := unsafe.Pointer(&galKeysC)

	for i := 0; i < server.Params.NParallelism; i++ {
		C.set_galois_keys(server.DBs[i], keysPtr)
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

func (client *Client) GetFVIndex(elemIndex int64) int64 {
	return int64(C.fv_index(client.Pointer, C.ulong(elemIndex)))
}

func (client *Client) GetFVOffset(elemIndex int64) int64 {
	return int64(C.fv_offset(client.Pointer, C.ulong(elemIndex)))
}

func (client *Client) GenQuery(index int64) *Query {

	// HACK: convert SerializedQuery struct from wrapper.h into a Query struct
	// see: https://stackoverflow.com/questions/28551043/golang-cast-memory-to-struct
	qPtr := C.gen_query(client.Pointer, C.ulong(index))
	size := unsafe.Sizeof(QueryCStruct{})
	structMem := (*(*[1<<31 - 1]byte)(qPtr))[:size]
	queryC := (*(*QueryCStruct)(unsafe.Pointer(&structMem[0])))

	query := Query{
		Str: C.GoStringN(queryC.StrPtr, C.int(queryC.StrLen)),
	}

	C.free_query(qPtr)

	query.CiphertextSize = uint64(queryC.CiphertextSize)
	query.Count = uint64(queryC.Count)
	query.ClientID = uint64(queryC.ClientID)

	return &query
}

func (server *Server) GenAnswer(query *Query) []*Answer {

	// convert to queryC type
	queryC := QueryCStruct{
		StrPtr: C.CString(query.Str),
	}
	queryC.CiphertextSize = C.ulong(query.CiphertextSize)
	queryC.StrLen = C.ulong(len(query.Str))
	queryC.Count = C.ulong(query.Count)
	queryC.ClientID = C.ulong(query.ClientID)

	qPtr := unsafe.Pointer(&queryC)

	answers := make([]*Answer, server.Params.NParallelism)

	var wg sync.WaitGroup
	for i := 0; i < server.Params.NParallelism; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ansPtr := C.gen_answer(server.DBs[i], qPtr)

			// convert SerializedAnswer struct from wrapper.h into a Answer struct
			// see: https://stackoverflow.com/questions/28551043/golang-cast-memory-to-struct
			size := unsafe.Sizeof(AnswerCStruct{})
			structMem := (*(*[1<<31 - 1]byte)(ansPtr))[:size]
			answerC := (*(*AnswerCStruct)(unsafe.Pointer(&structMem[0])))

			answer := Answer{
				Str: C.GoStringN(answerC.StrPtr, C.int(answerC.StrLen)),
			}

			C.free_answer(ansPtr)

			answer.CiphertextSize = uint64(answerC.CiphertextSize)
			answer.Count = uint64(answerC.Count)

			answers[i] = &answer
		}(i)
	}

	wg.Wait()

	return answers
}

func (server *Server) ExpandedQuery(query *Query) *ExpandedQuery {

	// convert to queryC type
	queryC := QueryCStruct{
		StrPtr: C.CString(query.Str),
	}
	queryC.CiphertextSize = C.ulong(query.CiphertextSize)
	queryC.StrLen = C.ulong(len(query.Str))
	queryC.Count = C.ulong(query.Count)
	queryC.ClientID = C.ulong(query.ClientID)

	qPtr := unsafe.Pointer(&queryC)

	return &ExpandedQuery{
		Pointer: C.gen_expanded_query(server.DBs[0], qPtr),
	}
}

func (server *Server) GenAnswerWithExpandedQuery(expandedQuery *ExpandedQuery) []*Answer {

	answers := make([]*Answer, server.Params.NParallelism)

	var wg sync.WaitGroup
	for i := 0; i < server.Params.NParallelism; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			ansPtr := C.gen_answer_with_expanded_query(server.DBs[i], expandedQuery.Pointer)

			// convert SerializedAnswer struct from wrapper.h into a Answer struct
			// see: https://stackoverflow.com/questions/28551043/golang-cast-memory-to-struct
			size := unsafe.Sizeof(AnswerCStruct{})
			structMem := (*(*[1<<31 - 1]byte)(ansPtr))[:size]
			answerC := (*(*AnswerCStruct)(unsafe.Pointer(&structMem[0])))

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

func (client *Client) Recover(answer *Answer, offset int64) []byte {

	// convert to answerC type
	answerC := AnswerCStruct{
		StrPtr: C.CString(answer.Str),
	}
	answerC.CiphertextSize = C.ulong(answer.CiphertextSize)
	answerC.StrLen = C.ulong(len(answer.Str))
	answerC.Count = C.ulong(answer.Count)

	res := C.recover(client.Pointer, unsafe.Pointer(&answerC))
	minSize := 8 * (offset + 1) * int64(client.Params.ItemBytes)
	return C.GoBytes(unsafe.Pointer(res), C.int(minSize))
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

func DeserializeParamsList(serList []*SerializedParams) []*Params {
	list := make([]*Params, len(serList))

	for i := 0; i < len(serList); i++ {
		list[i] = DeserializeParams(serList[i])
	}

	return list
}
