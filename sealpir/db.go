package sealpir

import (
	"math/rand"
)

type Database struct {
	Server *Server
	Bytes  []byte
}

func InitRandomDB(params *Params) ([]byte, *Database) {

	server := InitServer(params)

	data := make([]byte, params.NumItems*params.ItemBytes)
	rand.Read(data) // fill with random bytes

	db := &Database{
		Server: server,
		Bytes:  data,
	}

	server.SetupDatabase(db)

	return data, db
}
