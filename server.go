package forge

import (
	"fmt"
	"path/filepath"

	"github.com/imagvfx/forge/property"
	"github.com/imagvfx/forge/service"
)

type Server struct {
	svc service.Service
	cfg *Config
}

func NewServer(svc service.Service, cfg *Config) *Server {
	s := &Server{
		svc: svc,
		cfg: cfg,
	}
	return s
}

func (s *Server) GetEntry(path string) (*Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("path emtpy")
	}
	es, err := s.svc.FindEntries(service.EntryFinder{
		Path: path,
	})
	if err != nil {
		return nil, err
	}
	if len(es) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	if len(es) != 1 {
		return nil, fmt.Errorf("got more than 1 entry")
	}
	e := es[0]
	parentID := -1
	if e.ParentID != nil {
		parentID = *e.ParentID
	}
	ent := &Entry{
		srv:      s,
		id:       e.ID,
		parentID: parentID,
		path:     e.Path,
		typ:      e.Type,
	}
	return ent, nil
}

func (s *Server) getEntry(id int) (*Entry, error) {
	e, err := s.svc.GetEntry(id)
	if err != nil {
		return nil, err
	}
	parentID := -1
	if e.ParentID != nil {
		parentID = *e.ParentID
	}
	ent := &Entry{
		srv:      s,
		id:       e.ID,
		parentID: parentID,
		path:     e.Path,
		typ:      e.Type,
	}
	return ent, nil
}

func (s *Server) subEntries(parent int) ([]*Entry, error) {
	es, err := s.svc.FindEntries(service.EntryFinder{
		ParentID: &parent,
	})
	if err != nil {
		return nil, err
	}
	ents := make([]*Entry, 0)
	for _, e := range es {
		parentID := -1
		if e.ParentID != nil {
			parentID = *e.ParentID
		}
		ent := &Entry{
			srv:      s,
			id:       e.ID,
			parentID: parentID,
			path:     e.Path,
			typ:      e.Type,
		}
		ents = append(ents, ent)
	}
	return ents, nil
}

func (s *Server) AddEntry(path, typ string) error {
	path = filepath.ToSlash(path)
	parent := filepath.Dir(path)
	p, err := s.GetEntry(parent)
	if err != nil {
		return fmt.Errorf("error on parent check: %v", err)
	}
	allow := false
	subtyps := s.cfg.Struct[p.Type()].SubEntryTypes
	for _, subtyp := range subtyps {
		if subtyp == typ {
			allow = true
			break
		}
	}
	if !allow {
		return fmt.Errorf("cannot create a child of type %q from %q", typ, p.Type())
	}
	e := &service.Entry{
		ParentID: &p.id,
		Path:     path,
		Type:     typ,
	}
	props := make([]*service.Property, 0)
	for _, ktv := range s.cfg.Struct[typ].Properties {
		p := &service.Property{
			Name:  ktv.Key,
			Type:  ktv.Type,
			Value: ktv.Value,
		}
		props = append(props, p)
	}
	envs := make([]*service.Environ, 0)
	for _, ktv := range s.cfg.Struct[typ].Environs {
		e := &service.Environ{
			Name:  ktv.Key,
			Value: ktv.Value,
		}
		envs = append(envs, e)
	}
	err = s.svc.AddEntry(e, props, envs)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) entryProperties(ent int) ([]*Property, error) {
	ps, err := s.svc.FindProperties(service.PropertyFinder{
		EntryID: ent,
	})
	if err != nil {
		return nil, err
	}
	props := make([]*Property, 0)
	for _, p := range ps {
		prop := &Property{
			srv:     s,
			id:      p.ID,
			entryID: p.EntryID,
			name:    p.Name,
			typ:     p.Type,
			value:   p.Value,
		}
		props = append(props, prop)
	}
	return props, nil
}

func (s *Server) getProperty(ent int, name string) (*Property, error) {
	ps, err := s.svc.FindProperties(service.PropertyFinder{
		EntryID: ent,
		Name:    &name,
	})
	if err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	if len(ps) != 1 {
		return nil, fmt.Errorf("got more than 1 property")
	}
	p := ps[0]
	prop := &Property{
		srv:     s,
		id:      p.ID,
		entryID: p.EntryID,
		name:    p.Name,
		typ:     p.Type,
		value:   p.Value,
	}
	return prop, nil
}

func (s *Server) AddProperty(path string, name, typ, value string) error {
	ent, err := s.GetEntry(path)
	if err != nil {
		return err
	}
	// When the value is in template form, the result should be valid value of the type.
	ev := Evaluator{
		Path: ent.Path(),
		Name: ent.Name(),
	}
	err = property.Validate(typ, ev.Eval(value))
	if err != nil {
		return err
	}
	err = s.svc.AddProperty(&service.Property{
		EntryID: ent.id,
		Name:    name,
		Type:    typ,
		Value:   value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetProperty(path string, name, value string) error {
	ent, err := s.GetEntry(path)
	if err != nil {
		return err
	}
	prop, err := s.getProperty(ent.id, name)
	if err != nil {
		return err
	}
	// When the value is in template form, the result should be valid value of the type.
	ev := Evaluator{
		Path: ent.Path(),
		Name: ent.Name(),
	}
	err = property.Validate(prop.Type(), ev.Eval(value))
	if err != nil {
		return err
	}
	err = s.svc.UpdateProperty(service.PropertyUpdater{
		ID:    prop.id,
		Value: &value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) entryEnvirons(ent int) ([]*Environ, error) {
	ps, err := s.svc.FindEnvirons(service.EnvironFinder{
		EntryID: ent,
	})
	if err != nil {
		return nil, err
	}
	props := make([]*Environ, 0)
	for _, p := range ps {
		prop := &Environ{
			srv:     s,
			id:      p.ID,
			entryID: p.EntryID,
			name:    p.Name,
			value:   p.Value,
		}
		props = append(props, prop)
	}
	return props, nil
}

func (s *Server) getEnviron(ent int, name string) (*Environ, error) {
	es, err := s.svc.FindEnvirons(service.EnvironFinder{
		EntryID: ent,
		Name:    &name,
	})
	if err != nil {
		return nil, err
	}
	if len(es) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	if len(es) != 1 {
		return nil, fmt.Errorf("got more than 1 environ")
	}
	e := es[0]
	env := &Environ{
		srv:     s,
		id:      e.ID,
		entryID: e.EntryID,
		name:    e.Name,
		value:   e.Value,
	}
	return env, nil
}

func (s *Server) AddEnviron(path string, name, value string) error {
	ent, err := s.GetEntry(path)
	if err != nil {
		return err
	}
	err = s.svc.AddEnviron(&service.Environ{
		EntryID: ent.id,
		Name:    name,
		Value:   value,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) SetEnviron(path string, name, value string) error {
	ent, err := s.GetEntry(path)
	if err != nil {
		return err
	}
	env, err := s.getEnviron(ent.id, name)
	if err != nil {
		return err
	}
	err = s.svc.UpdateEnviron(service.EnvironUpdater{
		ID:    env.id,
		Value: &value,
	})
	if err != nil {
		return err
	}
	return nil
}
