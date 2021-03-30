package forge

import (
	"fmt"
	"path/filepath"

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
	return err
}
