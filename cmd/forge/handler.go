package main

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/imagvfx/forge"
)

type pathHandler struct {
	server *forge.Server
	cfg    *forge.Config
}

func handleError(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *pathHandler) Handle(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		path := r.URL.Path
		ent, err := h.server.GetEntry(path)
		if err != nil {
			return err
		}
		subEnts, err := ent.SubEntries()
		if err != nil {
			return err
		}
		props, err := ent.Properties()
		if err != nil {
			return err
		}
		envs, err := ent.Environs()
		if err != nil {
			return err
		}
		logs, err := ent.Logs()
		if err != nil {
			return err
		}
		subtyps := h.cfg.Struct[ent.Type()].SubEntryTypes
		recipe := struct {
			Entry         *forge.Entry
			SubEntries    []*forge.Entry
			Properties    []*forge.Property
			Environs      []*forge.Property
			SubEntryTypes []string
			Logs          []*forge.Log
		}{
			Entry:         ent,
			SubEntries:    subEnts,
			Properties:    props,
			Environs:      envs,
			SubEntryTypes: subtyps,
			Logs:          logs,
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
		typ := r.FormValue("type")
		err := h.server.AddEntry(path, typ)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddProperty(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		if r.Method != "POST" {
			return fmt.Errorf("need POST, got %v", r.Method)
		}
		path := r.FormValue("path")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		err := h.server.AddProperty(path, name, typ, value)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetProperty(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		if r.Method != "POST" {
			return fmt.Errorf("need POST, got %v", r.Method)
		}
		path := r.FormValue("path")
		name := r.FormValue("name")
		value := r.FormValue("value")
		err := h.server.SetProperty(path, name, value)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddEnviron(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		if r.Method != "POST" {
			return fmt.Errorf("need POST, got %v", r.Method)
		}
		path := r.FormValue("path")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		err := h.server.AddEnviron(path, name, typ, value)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetEnviron(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		if r.Method != "POST" {
			return fmt.Errorf("need POST, got %v", r.Method)
		}
		path := r.FormValue("path")
		name := r.FormValue("name")
		value := r.FormValue("value")
		err := h.server.SetEnviron(path, name, value)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}
