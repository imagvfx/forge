package forge

import (
	"fmt"
	"path/filepath"

	"github.com/imagvfx/forge/property"
	"github.com/imagvfx/forge/service"
)

type Server struct {
	svc service.Service
}

func NewServer(svc service.Service) *Server {
	s := &Server{
		svc: svc,
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
		}
		ents = append(ents, ent)
	}
	return ents, nil
}

func (s *Server) AddEntry(path string) error {
	path = filepath.ToSlash(path)
	parent := filepath.Dir(path)
	p, err := s.GetEntry(parent)
	if err != nil {
		return fmt.Errorf("error on parent check: %v", err)
	}
	e := &service.Entry{
		ParentID: &p.id,
		Path:     path,
	}
	err = s.svc.AddEntry(e)
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
			inherit: p.Inherit,
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
		inherit: p.Inherit,
	}
	return prop, nil
}

func (s *Server) AddProperty(path string, name, typ, value string) error {
	err := property.Validate(typ, value)
	if err != nil {
		return err
	}
	ent, err := s.GetEntry(path)
	if err != nil {
		return err
	}
	err = s.svc.AddProperty(&service.Property{
		EntryID: ent.id,
		Name:    name,
		Type:    typ,
		Value:   value,
		Inherit: false,
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
	err = property.Validate(prop.typ, value)
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
