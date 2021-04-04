package forge

import (
	"bytes"
	"encoding/json"
	"html/template"
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

func (e *Entry) Environs() ([]*Environ, error) {
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

type Property struct {
	srv     *Server
	id      int
	entryID int
	name    string
	typ     string
	value   string
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
	ent, err := p.Entry()
	if err != nil {
		return err.Error()
	}
	ev := Evaluator{
		Path: ent.Path(),
		Name: ent.Name(),
	}
	return ev.Eval(p.value)
}

type Environ struct {
	srv     *Server
	id      int
	entryID int
	name    string
	value   string
}

func (p *Environ) Entry() (*Entry, error) {
	return p.srv.getEntry(p.entryID)
}

func (p *Environ) Name() string {
	return p.name
}

func (p *Environ) RawValue() string {
	return p.value
}

func (p *Environ) Value() string {
	ent, err := p.Entry()
	if err != nil {
		return err.Error()
	}
	ev := Evaluator{
		Path: ent.Path(),
		Name: ent.Name(),
	}
	return ev.Eval(p.value)
}

type Evaluator struct {
	Path string
	Name string
}

func (e Evaluator) Eval(val string) string {
	t, err := template.New("").Parse(val)
	if err != nil {
		return err.Error()
	}
	recipe := struct {
		Path string
		Name string
	}{
		Path: e.Path,
		Name: e.Name,
	}
	w := &bytes.Buffer{}
	err = t.Execute(w, recipe)
	if err != nil {
		return err.Error()
	}
	return w.String()
}
