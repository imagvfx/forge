package main

import (
	"fmt"
	"net/http"
	"path/filepath"

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
		if path[:3] != "/e/" {
			return fmt.Errorf("assumtion of the path failed")
		}
		path = path[2:]
		ent, err := h.server.GetEntry(path)
		if err != nil {
			return err
		}
		subEnts, err := ent.SubEntries()
		if err != nil {
			return err
		}
		recipe := struct {
			Entry      *forge.Entry
			SubEntries []*forge.Entry
		}{
			Entry:      ent,
			SubEntries: subEnts,
		}
		err = Tmpl.ExecuteTemplate(w, "path.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

type apiHandler struct {
	server *forge.Server
}

func (h *apiHandler) HandleAddEntry(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		if r.Method != "POST" {
			return fmt.Errorf("need POST, got %v", r.Method)
		}
		// parent, if suggested, will be used as prefix of the path.
		parent := r.FormValue("parent")
		path := r.FormValue("path")
		path = filepath.Join(parent, path)
		err := h.server.AddEntry(path)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}
