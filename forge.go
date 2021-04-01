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

func (e *Entry) Properties() ([]*Property, error) {
	return e.srv.entryProperties(e.id)
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

type Property struct {
	srv     *Server
	id      int
	entID   int
	name    string
	typ     string
	value   string
	inherit bool
}

func (p *Property) Entry() (*Entry, error) {
	return p.srv.getEntry(p.entID)
}

func (p *Property) Type() string {
	return p.typ
}

func (p *Property) Name() string {
	return p.name
}

func (p *Property) Value() string {
	return p.value
}

func (p *Property) Inherit() bool {
	return p.inherit
}
