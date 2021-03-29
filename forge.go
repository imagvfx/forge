package forge

import "encoding/json"

type Entry struct {
	srv      *Server
	id       int
	parentID int
	path     string
	name     string
}

func (e *Entry) Path() string {
	return e.path
}

func (e *Entry) Name() string {
	return e.name
}

func (e *Entry) SubEntries() ([]*Entry, error) {
	return e.srv.subEntries(e.id)
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	m := struct {
		Name       string
		SubEntries []string
	}{
		Name: e.name,
	}
	return json.Marshal(m)
}

type Property interface {
	Name() string
	Value() string
	Set(string) error
}
