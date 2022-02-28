package server

import (
	"crypto/rand"
	"log"
	"math/big"

	"github.com/sachaservan/adveil/api"
	"github.com/sachaservan/adveil/sealpir"
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
func (serv *Server) InitSession(args api.InitSessionArgs, reply *api.InitSessionResponse) error {

	log.Printf("[Server]: received request to InitSession")

	// generate new session ID
	sessionID := newUUID()

	// make a new session for the client
	clientSession := &ClientSession{
		sessionID: sessionID,
	}

	serv.Sessions[sessionID] = clientSession

	reply.SessionID = sessionID
	reply.NumFeatures = serv.KnnParams.NumFeatures
	reply.NumCategories = serv.NumCategories
	reply.NumTables = serv.KnnParams.NumTables
	reply.NumProbes = serv.KnnParams.NumProbes
	reply.NumTableDBs = len(serv.TableDBs)
	reply.TablePIRParams = sealpir.SerializeParams(serv.TableParams)
	reply.TableHashFunctions = serv.Knn.Hashes

	return nil
}

func (serv *Server) SetPIRKeys(args api.SetKeysArgs, reply *api.SetKeysResponse) error {

	log.Printf("[Server]: received request to SetPIRKeys")

	for i := 0; i < serv.KnnParams.NumTables; i++ {
		serv.TableDBs[i].Server.SetGaloisKeys(args.TableDBGaloisKeys)
	}

	return nil
}

// TerminateSession kills the server
func (serv *Server) TerminateSession(args *api.TerminateSessionArgs, reply *api.TerminateSessionResponse) error {
	serv.Killed = true

	// serv.AdDb.Server.Free()
	// serv.TableDBs[0].Server.Free()

	// TODO: if using different DBs for each table then free each table
	// for _, db := range server.TableDBs {
	// 	db.Server.Free()
	// }

	return nil
}

func newUUID() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}
