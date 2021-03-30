package forge

import (
	"encoding/json"
	"path/filepath"
)

type Entry struct {
	srv      *Server
	id       int
	parentID int
	path     string
}

func (e *Entry) Path() string {
	return e.path
}

func (e *Entry) Name() string {
	return filepath.Base(e.path)
}

func (e *Entry) SubEntries() ([]*Entry, error) {
	return e.srv.subEntries(e.id)
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	m := struct {
		Path       string
		SubEntries []string
	}{
		Path: e.path,
	}
	return json.Marshal(m)
}

type Property interface {
	Name() string
	Value() string
	Set(string) error
}
