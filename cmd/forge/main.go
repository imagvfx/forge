package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service/sqlite"
	"github.com/kybin/bml"
)

var Tmpl *template.Template

func main() {
	var (
		addr   string
		dbpath string
	)
	flag.StringVar(&addr, "addr", "0.0.0.0:8080", "address to bind")
	flag.StringVar(&dbpath, "db", "forge.db", "db path to create or open")

	cfg, err := forge.LoadConfig("config/")
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range cfg.Struct {
		fmt.Println(s)
	}

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
	api := &apiHandler{
		server: server,
	}
	Tmpl = template.New("")
	Tmpl = template.Must(bml.ToHTMLTemplate(Tmpl, "tmpl/*"))
	mux := http.NewServeMux()
	mux.HandleFunc("/", path.Handle)
	mux.HandleFunc("/api/add-entry", api.HandleAddEntry)
	mux.HandleFunc("/api/add-property", api.HandleAddProperty)
	mux.HandleFunc("/api/set-property", api.HandleSetProperty)
	mux.HandleFunc("/api/add-environ", api.HandleAddEnviron)
	mux.HandleFunc("/api/set-environ", api.HandleSetEnviron)
	http.ListenAndServe(addr, mux)
}
