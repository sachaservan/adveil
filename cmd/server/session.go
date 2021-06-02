package main

import (
	"crypto/rand"
	"log"
	"math/big"

	"github.com/sachaservan/adveil/cmd/api"
	"github.com/sachaservan/adveil/cmd/sealpir"
)

// ClientSession stores session info for a client request
type ClientSession struct {
	sessionID int64
}

// InitSessionFromServerArgs used to initialize a server to a given states
type InitSessionFromServerArgs struct {
	Session *ClientSession
}

// InitSessionFromServerResponse acks a session install
type InitSessionFromServerResponse struct {
	Error api.Error
}

// InitSession initializes a new KNN query session for the client
func (server *Server) InitSession(args api.InitSessionArgs, reply *api.InitSessionResponse) error {

	log.Printf("[Server]: received request to InitSession")

	// generate new session ID
	sessionID := newUUID()

	// make a new session for the client
	clientSession := &ClientSession{
		sessionID: sessionID,
	}

	server.Sessions[sessionID] = clientSession

	reply.SessionID = sessionID
	reply.NumFeatures = server.KnnParams.NumFeatures
	reply.AdPIRParams = sealpir.SerializeParams(server.AdDb.Server.Params)
	reply.AdSizeKB = server.AdSize
	reply.NumAds = server.NumAds

	if server.ANNS {
		reply.NumTables = server.Knn.NumTables()
		reply.TablePIRParams = sealpir.SerializeParamsMap(server.TableParams)
		reply.TableHashFunctions = server.Knn.Hashes
		reply.IDtoVecPIRParams = sealpir.SerializeParamsMap(server.IDtoVecParams)
	}

	return nil
}

func (server *Server) SetPIRKeys(args api.SetKeysArgs, reply *api.SetKeysResponse) error {

	log.Printf("[Server]: received request to SetPIRKeys")

	server.AdDb.Server.SetGaloisKeys(args.AdDBGaloisKeys)

	server.IDtoVecDB[0].Server.SetGaloisKeys(args.IDtoVecKeys[0])

	for i := 0; i < server.KnnParams.NumTables; i++ {
		server.TableDBs[i].Server.SetGaloisKeys(args.TableDBGaloisKeys[i])
	}

	return nil
}

// TerminateSession kills the server
func (server *Server) TerminateSession(args *api.TerminateSessionArgs, reply *api.TerminateSessionResponse) error {
	server.Killed = true

	server.AdDb.Server.Free()

	for _, db := range server.TableDBs {
		db.Server.Free()
	}

	return nil
}

func newUUID() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}
