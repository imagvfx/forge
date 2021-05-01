package service

import (
	"context"
	"fmt"
	"time"
)

type Service interface {
	FindEntries(ctx context.Context, find EntryFinder) ([]*Entry, error)
	GetEntry(ctx context.Context, path string) (*Entry, error)
	AddEntry(ctx context.Context, ent *Entry, props []*Property, envs []*Property) error
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
	AddUser(ctx context.Context, u *User) error
	GetUserByEmail(ctx context.Context, user string) (*User, error)
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
	userEmailContextKey = contextKey(iota + 1)
)

func ContextWithUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, userEmailContextKey, email)
}

func UserEmailFromContext(ctx context.Context) string {
	email := ctx.Value(userEmailContextKey)
	if email == nil {
		return ""
	}
	return email.(string)
}

type Entry struct {
	ID           int
	ParentID     *int // nil if root entry
	Path         string
	Type         string
	HasThumbnail bool
}

type EntryFinder struct {
	ID       *int
	ParentID *int
	Path     *string
}

type Thumbnail struct {
	ID      int
	EntryID int
	Data    []byte
}

type ThumbnailFinder struct {
	ID      *int
	EntryID *int
}

type ThumbnailUpdater struct {
	ID   int
	Data []byte
}

type Property struct {
	ID        int
	EntryID   int
	EntryPath string
	Name      string
	Type      string
	Value     string
}

type PropertyFinder struct {
	ID        *int
	EntryID   *int
	EntryPath *string
	Name      *string
}

type PropertyUpdater struct {
	ID    int
	Value *string
}

type AccessControl struct {
	ID        int
	EntryID   int
	EntryPath string
	// either UserID or GroupID is not nil
	UserID       *int
	GroupID      *int
	Accessor     string
	AccessorType int
	Mode         int
}

type AccessControlFinder struct {
	ID        *int
	EntryID   *int
	EntryPath *string
	User      *string
	Group     *string
}

type AccessControlUpdater struct {
	ID   int
	Mode *int
}

type User struct {
	ID    int
	Email string
	Name  string
}

type UserFinder struct {
	ID    *int
	Email *string
}

type UserUpdater struct {
	ID    int
	Email *string
}

type Group struct {
	ID   int
	Name string
}

type GroupFinder struct {
	ID   *int
	Name *string
}

type GroupUpdater struct {
	ID   int
	Name *string
}

type Member struct {
	ID      int
	GroupID int
	Group   string
	UserID  int
	User    string
}

type MemberFinder struct {
	ID      *int
	GroupID *int
	Group   *string
	Member  *string
}

type Log struct {
	ID       int
	EntryID  int
	User     string
	Action   string
	Category string
	Name     string
	Type     string
	Value    string
	When     time.Time
}

type LogFinder struct {
	EntryID int
}
