package main

import (
	"linebackerr/db"
	"linebackerr/nflverse"
	"linebackerr/server"
	"linebackerr/sportarr"
)

func main() {
	deebee := db.Init()
	nflv := nflverse.Init(deebee)
	sportarr.Init(deebee, nflv)

	srvr := server.Init()
	if err := srvr.Start(); err != nil {
		panic(err)
	}
}
