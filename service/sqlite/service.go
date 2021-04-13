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

func (s *Service) AddEntry(user string, ent *service.Entry, props []*service.Property, env []*service.Property) error {
	return AddEntry(s.db, user, ent, props, env)
}

func (s *Service) FindProperties(find service.PropertyFinder) ([]*service.Property, error) {
	return FindProperties(s.db, find)
}

func (s *Service) AddProperty(user string, ent *service.Property) error {
	return AddProperty(s.db, user, ent)
}

func (s *Service) UpdateProperty(user string, upd service.PropertyUpdater) error {
	return UpdateProperty(s.db, user, upd)
}

func (s *Service) FindEnvirons(find service.PropertyFinder) ([]*service.Property, error) {
	return FindEnvirons(s.db, find)
}

func (s *Service) AddEnviron(user string, ent *service.Property) error {
	return AddEnviron(s.db, user, ent)
}

func (s *Service) UpdateEnviron(user string, upd service.PropertyUpdater) error {
	return UpdateEnviron(s.db, user, upd)
}

func (s Service) FindAccessControls(find service.AccessControlFinder) ([]*service.AccessControl, error) {
	return FindAccessControls(s.db, find)
}

func (s Service) AddAccessControl(user string, a *service.AccessControl) error {
	return AddAccessControl(s.db, user, a)
}

func (s Service) UpdateAccessControl(user string, upd service.AccessControlUpdater) error {
	return UpdateAccessControl(s.db, user, upd)
}

func (s *Service) FindLogs(find service.LogFinder) ([]*service.Log, error) {
	return FindLogs(s.db, find)
}

func (s *Service) AddUser(u *service.User) error {
	return AddUser(s.db, u)
}

func (s *Service) GetUserByUser(user string) (*service.User, error) {
	return GetUserByUser(s.db, user)
}

func (s *Service) FindGroups(find service.GroupFinder) ([]*service.Group, error) {
	return FindGroups(s.db, find)
}

func (s *Service) AddGroup(user string, g *service.Group) error {
	return AddGroup(s.db, user, g)
}

func (s *Service) UpdateGroup(user string, upd service.GroupUpdater) error {
	return UpdateGroup(s.db, user, upd)
}
