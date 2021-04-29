package sqlite

import (
	"context"
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

func (s *Service) FindEntries(ctx context.Context, find service.EntryFinder) ([]*service.Entry, error) {
	return FindEntries(s.db, ctx, find)
}

func (s *Service) GetEntry(ctx context.Context, path string) (*service.Entry, error) {
	return GetEntry(s.db, ctx, path)
}

func (s *Service) AddEntry(ctx context.Context, ent *service.Entry, props []*service.Property, env []*service.Property) error {
	return AddEntry(s.db, ctx, ent, props, env)
}

func (s *Service) RenameEntry(ctx context.Context, path, newName string) error {
	return RenameEntry(s.db, ctx, path, newName)
}

func (s *Service) DeleteEntry(ctx context.Context, path string) error {
	return DeleteEntry(s.db, ctx, path)
}

func (s *Service) GetThumbnail(ctx context.Context, path string) (*service.Thumbnail, error) {
	return GetThumbnail(s.db, ctx, path)
}

func (s *Service) AddThumbnail(ctx context.Context, thumb *service.Thumbnail) error {
	return AddThumbnail(s.db, ctx, thumb)
}

func (s *Service) UpdateThumbnail(ctx context.Context, upd service.ThumbnailUpdater) error {
	return UpdateThumbnail(s.db, ctx, upd)
}

func (s *Service) DeleteThumbnail(ctx context.Context, path string) error {
	return DeleteThumbnail(s.db, ctx, path)
}

func (s *Service) FindProperties(ctx context.Context, find service.PropertyFinder) ([]*service.Property, error) {
	return FindProperties(s.db, ctx, find)
}

func (s *Service) AddProperty(ctx context.Context, ent *service.Property) error {
	return AddProperty(s.db, ctx, ent)
}

func (s *Service) UpdateProperty(ctx context.Context, upd service.PropertyUpdater) error {
	return UpdateProperty(s.db, ctx, upd)
}

func (s *Service) DeleteProperty(ctx context.Context, path, name string) error {
	return DeleteProperty(s.db, ctx, path, name)
}

func (s *Service) FindEnvirons(ctx context.Context, find service.PropertyFinder) ([]*service.Property, error) {
	return FindEnvirons(s.db, ctx, find)
}

func (s *Service) AddEnviron(ctx context.Context, ent *service.Property) error {
	return AddEnviron(s.db, ctx, ent)
}

func (s *Service) UpdateEnviron(ctx context.Context, upd service.PropertyUpdater) error {
	return UpdateEnviron(s.db, ctx, upd)
}

func (s *Service) DeleteEnviron(ctx context.Context, path, name string) error {
	return DeleteEnviron(s.db, ctx, path, name)
}

func (s Service) FindAccessControls(ctx context.Context, find service.AccessControlFinder) ([]*service.AccessControl, error) {
	return FindAccessControls(s.db, ctx, find)
}

func (s Service) AddAccessControl(ctx context.Context, a *service.AccessControl) error {
	return AddAccessControl(s.db, ctx, a)
}

func (s Service) UpdateAccessControl(ctx context.Context, upd service.AccessControlUpdater) error {
	return UpdateAccessControl(s.db, ctx, upd)
}

func (s *Service) DeleteAccessControl(ctx context.Context, path, name string) error {
	return DeleteAccessControl(s.db, ctx, path, name)
}

func (s *Service) FindLogs(ctx context.Context, find service.LogFinder) ([]*service.Log, error) {
	return FindLogs(s.db, ctx, find)
}

func (s *Service) AddUser(ctx context.Context, u *service.User) error {
	return AddUser(s.db, ctx, u)
}

func (s *Service) GetUserByEmail(ctx context.Context, user string) (*service.User, error) {
	return GetUserByEmail(s.db, ctx, user)
}

func (s *Service) FindGroups(ctx context.Context, find service.GroupFinder) ([]*service.Group, error) {
	return FindGroups(s.db, ctx, find)
}

func (s *Service) AddGroup(ctx context.Context, g *service.Group) error {
	return AddGroup(s.db, ctx, g)
}

func (s *Service) UpdateGroup(ctx context.Context, upd service.GroupUpdater) error {
	return UpdateGroup(s.db, ctx, upd)
}

func (s *Service) FindGroupMembers(ctx context.Context, find service.MemberFinder) ([]*service.Member, error) {
	return FindGroupMembers(s.db, ctx, find)
}

func (s *Service) AddGroupMember(ctx context.Context, m *service.Member) error {
	return AddGroupMember(s.db, ctx, m)
}

func (s *Service) DeleteGroupMember(ctx context.Context, id int) error {
	return DeleteGroupMember(s.db, ctx, id)
}
