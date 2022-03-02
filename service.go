package forge

import (
	"context"
	"fmt"
)

type Service interface {
	FindEntryTypes(ctx context.Context) ([]string, error)
	FindBaseEntryTypes(ctx context.Context) ([]string, error)
	FindOverrideEntryTypes(ctx context.Context) ([]string, error)
	AddEntryType(ctx context.Context, name string) error
	RenameEntryType(ctx context.Context, name, newName string) error
	DeleteEntryType(ctx context.Context, name string) error
	FindDefaults(ctx context.Context, find DefaultFinder) ([]*Default, error)
	AddDefault(ctx context.Context, d *Default) error
	UpdateDefault(ctx context.Context, upd DefaultUpdater) error
	DeleteDefault(ctx context.Context, entType, ctg, name string) error
	FindGlobals(ctx context.Context, find GlobalFinder) ([]*Global, error)
	GetGlobal(ctx context.Context, entType, name string) (*Global, error)
	AddGlobal(ctx context.Context, d *Global) error
	UpdateGlobal(ctx context.Context, upd GlobalUpdater) error
	DeleteGlobal(ctx context.Context, entType, name string) error
	FindEntries(ctx context.Context, find EntryFinder) ([]*Entry, error)
	SearchEntries(ctx context.Context, search EntrySearcher) ([]*Entry, error)
	CountAllSubEntries(ctx context.Context, path string) (int, error)
	GetEntry(ctx context.Context, path string) (*Entry, error)
	AddEntry(ctx context.Context, ent *Entry) error
	RenameEntry(ctx context.Context, path string, newName string) error
	DeleteEntry(ctx context.Context, path string) error
	DeleteEntryRecursive(ctx context.Context, path string) error
	AddThumbnail(ctx context.Context, thumb *Thumbnail) error
	UpdateThumbnail(ctx context.Context, upd ThumbnailUpdater) error
	GetThumbnail(ctx context.Context, path string) (*Thumbnail, error)
	DeleteThumbnail(ctx context.Context, path string) error
	EntryProperties(ctx context.Context, path string) ([]*Property, error)
	GetProperty(ctx context.Context, path, name string) (*Property, error)
	UpdateProperty(ctx context.Context, upd PropertyUpdater) error
	UpdateProperties(ctx context.Context, upds []PropertyUpdater) error
	EntryEnvirons(ctx context.Context, path string) ([]*Property, error)
	GetEnviron(ctx context.Context, path, name string) (*Property, error)
	AddEnviron(ctx context.Context, p *Property) error
	UpdateEnviron(ctx context.Context, upd PropertyUpdater) error
	DeleteEnviron(ctx context.Context, path string, name string) error
	EntryAccessList(ctx context.Context, path string) ([]*Access, error)
	GetAccess(ctx context.Context, path, name string) (*Access, error)
	AddAccess(ctx context.Context, ac *Access) error
	UpdateAccess(ctx context.Context, upd AccessUpdater) error
	DeleteAccess(ctx context.Context, path string, name string) error
	FindLogs(ctx context.Context, find LogFinder) ([]*Log, error)
	GetLogs(ctx context.Context, path, ctg, name string) ([]*Log, error)
	FindUsers(ctx context.Context, find UserFinder) ([]*User, error)
	AddUser(ctx context.Context, u *User) error
	UpdateUserCalled(ctx context.Context, user, called string) error
	GetUser(ctx context.Context, user string) (*User, error)
	GetUserSetting(ctx context.Context, user string) (*UserSetting, error)
	UpdateUserSetting(ctx context.Context, upd UserSettingUpdater) error
	FindGroups(ctx context.Context, find GroupFinder) ([]*Group, error)
	AddGroup(ctx context.Context, g *Group) error
	UpdateGroup(ctx context.Context, upd GroupUpdater) error
	FindGroupMembers(ctx context.Context, find MemberFinder) ([]*Member, error)
	AddGroupMember(ctx context.Context, m *Member) error
	DeleteGroupMember(ctx context.Context, group, member string) error
}

type NotFoundError struct {
	err error
}

func (e *NotFoundError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func NotFound(s string, is ...interface{}) *NotFoundError {
	return &NotFoundError{fmt.Errorf(s, is...)}
}

type UnauthorizedError struct {
	err error
}

func (e *UnauthorizedError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func Unauthorized(s string, is ...interface{}) *UnauthorizedError {
	return &UnauthorizedError{fmt.Errorf(s, is...)}
}

type contextKey int

const (
	userNameContextKey = contextKey(iota + 1)
)

func ContextWithUserName(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, userNameContextKey, email)
}

func UserNameFromContext(ctx context.Context) string {
	email := ctx.Value(userNameContextKey)
	if email == nil {
		return ""
	}
	return email.(string)
}
