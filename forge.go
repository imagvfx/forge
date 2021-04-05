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
	typ      string
}

func (e *Entry) Path() string {
	return e.path
}

func (e *Entry) Dir() string {
	return filepath.Dir(e.path)
}

func (e *Entry) Name() string {
	return filepath.Base(e.path)
}

func (e *Entry) Type() string {
	return e.typ
}

func (e *Entry) SubEntries() ([]*Entry, error) {
	return e.srv.subEntries(e.id)
}

func (e *Entry) Properties() ([]*Property, error) {
	return e.srv.entryProperties(e.id)
}

func (e *Entry) Environs() ([]*Property, error) {
	return e.srv.entryEnvirons(e.id)
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

// Property can be either a normal property or an environment.
type Property struct {
	srv       *Server
	id        int
	entryID   int
	entryPath string
	name      string
	typ       string
	value     string
}

func (p *Property) Entry() (*Entry, error) {
	return p.srv.getEntry(p.entryID)
}

func (p *Property) Type() string {
	return p.typ
}

func (p *Property) Name() string {
	return p.name
}

func (p *Property) RawValue() string {
	return p.value
}

func (p *Property) Value() string {
	switch p.typ {
	case "entry_path":
		return filepath.Clean(filepath.Join(p.entryPath, p.value))
	case "entry_name":
		return filepath.Base(filepath.Clean(filepath.Join(p.entryPath, p.value)))
	}
	return p.value
}
