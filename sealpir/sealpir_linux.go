// +build linux

package sealpir

// #cgo CFLAGS: -I"/usr/local/include"
// #cgo LDFLAGS: ./../C/libsealwrapper.a -lstdc++ -lm -lseal
// #include <stdlib.h>
// #include "./../C/wrapper.h"
import "C"
import (
	"math"
	"sync"
	"unsafe"
)

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

func (client *Client) GetFVIndex(elemIndex int64) int64 {
	return int64(C.fv_index(client.Pointer, C.ulong(elemIndex)))
}

func (client *Client) GetFVOffset(elemIndex int64) int64 {
	return int64(C.fv_offset(client.Pointer, C.ulong(elemIndex)))
}

func (client *Client) GenQuery(index int64) *Query {

	qPtr := C.gen_query(client.Pointer, C.ulong(index))
	queryC := (*QueryCStruct)(unsafe.Pointer(qPtr))

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
			answerC := (*AnswerCStruct)(unsafe.Pointer(ansPtr))

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

func (client *Client) Recover(answer *Answer, offset int64) []byte {
	// convert to answerC type
	answerC := AnswerCStruct{
		StrPtr: C.CString(answer.Str),
	}
	answerC.CiphertextSize = C.ulong(answer.CiphertextSize)
	answerC.StrLen = C.ulong(len(answer.Str))
	answerC.Count = C.ulong(answer.Count)

	res := C.recover(client.Pointer, unsafe.Pointer(&answerC))
	minSize := 8 * offset * int64(client.Params.ItemBytes)
	return C.GoBytes(unsafe.Pointer(res), C.int(minSize))
}
