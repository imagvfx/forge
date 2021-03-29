package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service/sqlite"
)

func main() {
	var (
		addr   string
		dbpath string
	)
	flag.StringVar(&addr, "addr", "0.0.0.0:8080", "address to bind")
	flag.StringVar(&dbpath, "db", "forge.db", "db path to create or open")

	db, err := sqlite.Open(dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = sqlite.Init(db)
	if err != nil {
		log.Fatal(err)
	}
	svc := sqlite.NewService(db)
	server := forge.NewServer(svc)
	path := &pathHandler{
		server: server,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", path.Handle)
	http.ListenAndServe(addr, mux)
}
