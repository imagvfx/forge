package sqlite

import (
	"context"
	"database/sql"

	"github.com/imagvfx/forge"
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

func (s *Service) FindEntryTypes(ctx context.Context) ([]string, error) {
	return FindEntryTypes(s.db, ctx)
}

func (s *Service) FindBaseEntryTypes(ctx context.Context) ([]string, error) {
	return FindBaseEntryTypes(s.db, ctx)
}

func (s *Service) FindOverrideEntryTypes(ctx context.Context) ([]string, error) {
	return FindOverrideEntryTypes(s.db, ctx)
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

func (s *Service) FindDefaults(ctx context.Context, find forge.DefaultFinder) ([]*forge.Default, error) {
	return FindDefaults(s.db, ctx, find)
}

func (s *Service) AddDefault(ctx context.Context, d *forge.Default) error {
	return AddDefault(s.db, ctx, d)
}

func (s *Service) UpdateDefault(ctx context.Context, upd forge.DefaultUpdater) error {
	return UpdateDefault(s.db, ctx, upd)
}

func (s *Service) DeleteDefault(ctx context.Context, entType, ctg, name string) error {
	return DeleteDefault(s.db, ctx, entType, ctg, name)
}

func (s *Service) FindGlobals(ctx context.Context, find forge.GlobalFinder) ([]*forge.Global, error) {
	return FindGlobals(s.db, ctx, find)
}

func (s *Service) GetGlobal(ctx context.Context, entType, name string) (*forge.Global, error) {
	return GetGlobal(s.db, ctx, entType, name)
}

func (s *Service) AddGlobal(ctx context.Context, d *forge.Global) error {
	return AddGlobal(s.db, ctx, d)
}

func (s *Service) UpdateGlobal(ctx context.Context, upd forge.GlobalUpdater) error {
	return UpdateGlobal(s.db, ctx, upd)
}

func (s *Service) DeleteGlobal(ctx context.Context, entType, name string) error {
	return DeleteGlobal(s.db, ctx, entType, name)
}

func (s *Service) FindEntries(ctx context.Context, find forge.EntryFinder) ([]*forge.Entry, error) {
	return FindEntries(s.db, ctx, find)
}

func (s *Service) SearchEntries(ctx context.Context, search forge.EntrySearcher) ([]*forge.Entry, error) {
	return SearchEntries(s.db, ctx, search)
}

func (s *Service) CountAllSubEntries(ctx context.Context, path string) (int, error) {
	return CountAllSubEntries(s.db, ctx, path)
}

func (s *Service) GetEntry(ctx context.Context, path string) (*forge.Entry, error) {
	return GetEntry(s.db, ctx, path)
}

func (s *Service) AddEntry(ctx context.Context, ent *forge.Entry) error {
	return AddEntry(s.db, ctx, ent)
}

func (s *Service) RenameEntry(ctx context.Context, path, newName string) error {
	return RenameEntry(s.db, ctx, path, newName)
}

func (s *Service) ArchiveEntry(ctx context.Context, path string) error {
	return ArchiveEntry(s.db, ctx, path)
}

func (s *Service) UnarchiveEntry(ctx context.Context, path string) error {
	return UnarchiveEntry(s.db, ctx, path)
}

func (s *Service) DeleteEntry(ctx context.Context, path string) error {
	return DeleteEntry(s.db, ctx, path)
}

func (s *Service) DeleteEntryRecursive(ctx context.Context, path string) error {
	return DeleteEntryRecursive(s.db, ctx, path)
}

func (s *Service) GetThumbnail(ctx context.Context, path string) (*forge.Thumbnail, error) {
	return GetThumbnail(s.db, ctx, path)
}

func (s *Service) AddThumbnail(ctx context.Context, thumb *forge.Thumbnail) error {
	return AddThumbnail(s.db, ctx, thumb)
}

func (s *Service) UpdateThumbnail(ctx context.Context, upd forge.ThumbnailUpdater) error {
	return UpdateThumbnail(s.db, ctx, upd)
}

func (s *Service) DeleteThumbnail(ctx context.Context, path string) error {
	return DeleteThumbnail(s.db, ctx, path)
}

func (s *Service) EntryProperties(ctx context.Context, path string) ([]*forge.Property, error) {
	return EntryProperties(s.db, ctx, path)
}

func (s *Service) GetProperty(ctx context.Context, path, name string) (*forge.Property, error) {
	return GetProperty(s.db, ctx, path, name)
}

func (s *Service) UpdateProperty(ctx context.Context, upd forge.PropertyUpdater) error {
	return UpdateProperty(s.db, ctx, upd)
}

func (s *Service) UpdateProperties(ctx context.Context, upds []forge.PropertyUpdater) error {
	return UpdateProperties(s.db, ctx, upds)
}

func (s *Service) EntryEnvirons(ctx context.Context, path string) ([]*forge.Property, error) {
	return EntryEnvirons(s.db, ctx, path)
}

func (s *Service) GetEnvirons(ctx context.Context, path string) ([]*forge.Property, error) {
	return GetEnvirons(s.db, ctx, path)
}

func (s *Service) GetEnviron(ctx context.Context, path, name string) (*forge.Property, error) {
	return GetEnviron(s.db, ctx, path, name)
}

func (s *Service) AddEnviron(ctx context.Context, ent *forge.Property) error {
	return AddEnviron(s.db, ctx, ent)
}

func (s *Service) UpdateEnviron(ctx context.Context, upd forge.PropertyUpdater) error {
	return UpdateEnviron(s.db, ctx, upd)
}

func (s *Service) DeleteEnviron(ctx context.Context, path, name string) error {
	return DeleteEnviron(s.db, ctx, path, name)
}

func (s Service) EntryAccessList(ctx context.Context, path string) ([]*forge.Access, error) {
	return EntryAccessList(s.db, ctx, path)
}

func (s Service) GetAccessList(ctx context.Context, path string) ([]*forge.Access, error) {
	return GetAccessList(s.db, ctx, path)
}

func (s *Service) GetAccess(ctx context.Context, path, name string) (*forge.Access, error) {
	return GetAccess(s.db, ctx, path, name)
}

func (s Service) AddAccess(ctx context.Context, a *forge.Access) error {
	return AddAccess(s.db, ctx, a)
}

func (s Service) UpdateAccess(ctx context.Context, upd forge.AccessUpdater) error {
	return UpdateAccess(s.db, ctx, upd)
}

func (s *Service) DeleteAccess(ctx context.Context, path, name string) error {
	return DeleteAccess(s.db, ctx, path, name)
}

func (s *Service) IsAdmin(ctx context.Context, user string) (bool, error) {
	return IsAdmin(s.db, ctx, user)
}

func (s *Service) FindLogs(ctx context.Context, find forge.LogFinder) ([]*forge.Log, error) {
	return FindLogs(s.db, ctx, find)
}

func (s *Service) GetLogs(ctx context.Context, path, ctg, name string) ([]*forge.Log, error) {
	return GetLogs(s.db, ctx, path, ctg, name)
}

func (s *Service) FindUsers(ctx context.Context, find forge.UserFinder) ([]*forge.User, error) {
	return FindUsers(s.db, ctx, find)
}

func (s *Service) AddUser(ctx context.Context, u *forge.User) error {
	return AddUser(s.db, ctx, u)
}

func (s *Service) UpdateUser(ctx context.Context, upd forge.UserUpdater) error {
	return UpdateUser(s.db, ctx, upd)
}

func (s *Service) GetUser(ctx context.Context, user string) (*forge.User, error) {
	return GetUser(s.db, ctx, user)
}

func (s *Service) GetUserSetting(ctx context.Context, user string) (*forge.UserSetting, error) {
	return GetUserSetting(s.db, ctx, user)
}

func (s *Service) UpdateUserSetting(ctx context.Context, upd forge.UserSettingUpdater) error {
	return UpdateUserSetting(s.db, ctx, upd)
}

func (s *Service) AddUserDataSection(ctx context.Context, user, section string) error {
	return AddUserDataSection(s.db, ctx, user, section)
}

func (s *Service) GetUserDataSection(ctx context.Context, user, section string) (*forge.UserDataSection, error) {
	return GetUserDataSection(s.db, ctx, user, section)
}

func (s *Service) DeleteUserDataSection(ctx context.Context, user, section string) error {
	return DeleteUserDataSection(s.db, ctx, user, section)
}

func (s *Service) FindUserData(ctx context.Context, find forge.UserDataFinder) ([]*forge.UserDataSection, error) {
	return FindUserData(s.db, ctx, find)
}

func (s *Service) GetUserData(ctx context.Context, user, section, key string) (string, error) {
	return GetUserData(s.db, ctx, user, section, key)
}

func (s *Service) SetUserData(ctx context.Context, user, section, key, value string) error {
	return SetUserData(s.db, ctx, user, section, key, value)
}

func (s *Service) DeleteUserData(ctx context.Context, user, section, key string) error {
	return DeleteUserData(s.db, ctx, user, section, key)
}

func (s *Service) UserRead(ctx context.Context, path string) error {
	return UserRead(s.db, ctx, path)
}

func (s *Service) UserWrite(ctx context.Context, path string) error {
	return UserWrite(s.db, ctx, path)
}

func (s *Service) FindGroups(ctx context.Context, find forge.GroupFinder) ([]*forge.Group, error) {
	return FindGroups(s.db, ctx, find)
}

func (s *Service) AddGroup(ctx context.Context, g *forge.Group) error {
	return AddGroup(s.db, ctx, g)
}

func (s *Service) UpdateGroup(ctx context.Context, upd forge.GroupUpdater) error {
	return UpdateGroup(s.db, ctx, upd)
}

func (s *Service) FindGroupMembers(ctx context.Context, find forge.MemberFinder) ([]*forge.Member, error) {
	return FindGroupMembers(s.db, ctx, find)
}

func (s *Service) AddGroupMember(ctx context.Context, m *forge.Member) error {
	return AddGroupMember(s.db, ctx, m)
}

func (s *Service) DeleteGroupMember(ctx context.Context, group, member string) error {
	return DeleteGroupMember(s.db, ctx, group, member)
}
