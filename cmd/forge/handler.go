package main

import (
	"fmt"
	"net/http"

	"github.com/imagvfx/forge"
)

type pathHandler struct {
	server *forge.Server
}

func handleError(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *pathHandler) Handle(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		path := r.URL.Path
		fmt.Println(path)
		ent, err := h.server.GetEntry(path)
		if err != nil {
			return err
		}
		fmt.Println(ent)
		return nil
	}()
	handleError(w, err)
}
