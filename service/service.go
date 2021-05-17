package service

import (
	"context"
	"fmt"
	"time"
)

type Service interface {
	EntryTypes(ctx context.Context) ([]string, error)
	AddEntryType(ctx context.Context, name string) error
	RenameEntryType(ctx context.Context, name, newName string) error
	DeleteEntryType(ctx context.Context, name string) error
	SubEntryTypes(ctx context.Context, parentType string) ([]string, error)
	AddSubEntryType(ctx context.Context, parentType, subType string) error
	DeleteSubEntryType(ctx context.Context, parentType, subType string) error
	FindDefaults(ctx context.Context, find DefaultFinder) ([]*Default, error)
	AddDefault(ctx context.Context, d *Default) error
	UpdateDefault(ctx context.Context, upd DefaultUpdater) error
	DeleteDefault(ctx context.Context, entType, ctg, name string) error
	FindEntries(ctx context.Context, find EntryFinder) ([]*Entry, error)
	GetEntry(ctx context.Context, path string) (*Entry, error)
	AddEntry(ctx context.Context, ent *Entry) error
	RenameEntry(ctx context.Context, path string, newName string) error
	DeleteEntry(ctx context.Context, path string) error
	AddThumbnail(ctx context.Context, thumb *Thumbnail) error
	UpdateThumbnail(ctx context.Context, upd ThumbnailUpdater) error
	GetThumbnail(ctx context.Context, path string) (*Thumbnail, error)
	DeleteThumbnail(ctx context.Context, path string) error
	EntryProperties(ctx context.Context, path string) ([]*Property, error)
	GetProperty(ctx context.Context, path, name string) (*Property, error)
	AddProperty(ctx context.Context, p *Property) error
	UpdateProperty(ctx context.Context, upd PropertyUpdater) error
	DeleteProperty(ctx context.Context, path string, name string) error
	EntryEnvirons(ctx context.Context, path string) ([]*Property, error)
	GetEnviron(ctx context.Context, path, name string) (*Property, error)
	AddEnviron(ctx context.Context, p *Property) error
	UpdateEnviron(ctx context.Context, upd PropertyUpdater) error
	DeleteEnviron(ctx context.Context, path string, name string) error
	EntryAccessControls(ctx context.Context, path string) ([]*AccessControl, error)
	AddAccessControl(ctx context.Context, ac *AccessControl) error
	UpdateAccessControl(ctx context.Context, upd AccessControlUpdater) error
	DeleteAccessControl(ctx context.Context, path string, name string) error
	FindLogs(ctx context.Context, find LogFinder) ([]*Log, error)
	GetLogs(ctx context.Context, path, ctg, name string) ([]*Log, error)
	AddUser(ctx context.Context, u *User) error
	GetUser(ctx context.Context, user string) (*User, error)
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

type Entry struct {
	ID           int
	Path         string
	Type         string
	HasThumbnail bool
}

type EntryFinder struct {
	ParentPath *string
	Path       *string
}

type EntryTypeUpdater struct {
	ID   int
	Name *string
}

type Default struct {
	ID        int
	EntryType string
	Category  string
	Name      string
	Type      string
	Value     string
}

type DefaultFinder struct {
	EntryType *string
	Category  *string
	Name      *string
}

type DefaultUpdater struct {
	EntryType string
	Category  string
	Name      string
	Type      *string
	Value     *string
}

type Thumbnail struct {
	ID        int
	Data      []byte
	EntryPath string
}

type ThumbnailFinder struct {
	EntryPath *string
}

type ThumbnailUpdater struct {
	EntryPath string
	Data      []byte
}

type Property struct {
	ID        int
	EntryPath string
	Name      string
	Type      string
	Value     string
}

type PropertyFinder struct {
	EntryPath *string
	Name      *string
}

type PropertyUpdater struct {
	EntryPath string
	Name      string
	Value     *string
}

type AccessControl struct {
	ID        int
	EntryPath string
	// either UserID or GroupID is not nil
	Accessor     string
	AccessorType int
	Mode         int
}

type AccessControlFinder struct {
	EntryPath *string
	Accessor  *string
}

type AccessControlUpdater struct {
	EntryPath string
	Accessor  string
	Mode      *int
}

// Accessor is either a user or a group, that can be specified in entry access control list.
type Accessor struct {
	ID      int
	Name    string
	IsGroup bool
}

type Log struct {
	ID        int
	EntryPath string
	User      string
	Action    string
	Category  string
	Name      string
	Type      string
	Value     string
	When      time.Time
}

type LogFinder struct {
	EntryPath *string
	Category  *string
	Name      *string
}

type User struct {
	ID   int
	Name string
}

type UserFinder struct {
	Name *string
}

type UserUpdater struct {
	ID   int
	Name *string
}

type Group struct {
	ID   int
	Name string
}

type GroupFinder struct {
	Name *string
}

type GroupUpdater struct {
	ID   int
	Name *string
}

type Member struct {
	Group  string
	Member string
}

type MemberFinder struct {
	Group  *string
	Member *string
}
