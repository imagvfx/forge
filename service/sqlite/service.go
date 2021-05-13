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

func (s *Service) EntryTypes(ctx context.Context) ([]string, error) {
	return EntryTypes(s.db, ctx)
}

func (s *Service) AddEntryType(ctx context.Context, name string) error {
	return AddEntryType(s.db, ctx, name)
}

func (s *Service) RenameEntryType(ctx context.Context, name, newName string) error {
	return RenameEntryType(s.db, ctx, name, newName)
}

func (s *Service) DeleteEntryType(ctx context.Context, name string) error {
	return DeleteEntryType(s.db, ctx, name)
}

func (s *Service) FindEntryDefaults(ctx context.Context, find service.EntryDefaultFinder) ([]*service.EntryDefault, error) {
	return FindEntryDefaults(s.db, ctx, find)
}

func (s *Service) AddEntryDefault(ctx context.Context, d *service.EntryDefault) error {
	return AddEntryDefault(s.db, ctx, d)
}

func (s *Service) UpdateEntryDefault(ctx context.Context, upd service.EntryDefaultUpdater) error {
	return UpdateEntryDefault(s.db, ctx, upd)
}

func (s *Service) DeleteEntryDefault(ctx context.Context, entType, ctg, name string) error {
	return DeleteEntryDefault(s.db, ctx, entType, ctg, name)
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

func (s *Service) EntryProperties(ctx context.Context, path string) ([]*service.Property, error) {
	return EntryProperties(s.db, ctx, path)
}

func (s *Service) GetProperty(ctx context.Context, path, name string) (*service.Property, error) {
	return GetProperty(s.db, ctx, path, name)
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

func (s *Service) EntryEnvirons(ctx context.Context, path string) ([]*service.Property, error) {
	return EntryEnvirons(s.db, ctx, path)
}

func (s *Service) GetEnviron(ctx context.Context, path, name string) (*service.Property, error) {
	return GetEnviron(s.db, ctx, path, name)
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

func (s Service) EntryAccessControls(ctx context.Context, path string) ([]*service.AccessControl, error) {
	return EntryAccessControls(s.db, ctx, path)
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

func (s *Service) GetLogs(ctx context.Context, path, ctg, name string) ([]*service.Log, error) {
	return GetLogs(s.db, ctx, path, ctg, name)
}

func (s *Service) AddUser(ctx context.Context, u *service.User) error {
	return AddUser(s.db, ctx, u)
}

func (s *Service) GetUser(ctx context.Context, user string) (*service.User, error) {
	return GetUser(s.db, ctx, user)
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

func (s *Service) DeleteGroupMember(ctx context.Context, group, member string) error {
	return DeleteGroupMember(s.db, ctx, group, member)
}
