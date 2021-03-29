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

func (s *Service) AddEntry(ent *service.Entry) error {
	return AddEntry(s.db, ent)
}
