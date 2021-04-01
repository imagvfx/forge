package sqlite

import (
	"database/sql"

	"github.com/imagvfx/forge/service"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	s := &Service{
		db: db,
	}
	return s
}

func (s *Service) FindEntries(find service.EntryFinder) ([]*service.Entry, error) {
	return FindEntries(s.db, find)
}

func (s *Service) GetEntry(id int) (*service.Entry, error) {
	return GetEntry(s.db, id)
}

func (s *Service) AddEntry(ent *service.Entry) error {
	return AddEntry(s.db, ent)
}

func (s *Service) FindProperties(find service.PropertyFinder) ([]*service.Property, error) {
	return FindProperties(s.db, find)
}

func (s *Service) AddProperty(ent *service.Property) error {
	return AddProperty(s.db, ent)
}

func (s *Service) UpdateProperty(upd service.PropertyUpdater) error {
	return UpdateProperty(s.db, upd)
}
