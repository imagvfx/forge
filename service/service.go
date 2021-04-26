package service

import (
	"context"
	"time"
)

type Service interface {
	FindEntries(user string, find EntryFinder) ([]*Entry, error)
	GetEntry(user string, id int) (*Entry, error)
	UserCanWriteEntry(user string, id int) (bool, error)
	AddEntry(user string, ent *Entry, props []*Property, envs []*Property) error
	RenameEntry(user string, path string, newName string) error
	DeleteEntry(user string, path string) error
	FindProperties(user string, find PropertyFinder) ([]*Property, error)
	AddProperty(user string, p *Property) error
	UpdateProperty(user string, upd PropertyUpdater) error
	DeleteProperty(user string, path string, name string) error
	FindEnvirons(user string, find PropertyFinder) ([]*Property, error)
	AddEnviron(user string, p *Property) error
	UpdateEnviron(user string, upd PropertyUpdater) error
	DeleteEnviron(user string, path string, name string) error
	FindAccessControls(user string, find AccessControlFinder) ([]*AccessControl, error)
	AddAccessControl(user string, ac *AccessControl) error
	UpdateAccessControl(user string, upd AccessControlUpdater) error
	DeleteAccessControl(user string, path string, name string) error
	FindLogs(user string, find LogFinder) ([]*Log, error)
	AddUser(u *User) error
	GetUserByEmail(user string) (*User, error)
	FindGroups(find GroupFinder) ([]*Group, error)
	AddGroup(user string, g *Group) error
	UpdateGroup(user string, upd GroupUpdater) error
	FindGroupMembers(find MemberFinder) ([]*Member, error)
	AddGroupMember(user string, m *Member) error
	DeleteGroupMember(user string, id int) error
}

type contextKey int

const (
	userEmailContextKey = contextKey(iota + 1)
)

func ContextWithUserEmail(ctx context.Context, email string) {
	context.WithValue(ctx, userEmailContextKey, email)
}

func UserEmailFromContext(ctx context.Context) string {
	email := ctx.Value(userEmailContextKey)
	if email == nil {
		return ""
	}
	return email.(string)
}

type NotFoundError struct {
	Err string
}

func (e NotFoundError) Error() string {
	return e.Err
}

type Entry struct {
	ID       int
	ParentID *int // nil if root entry
	Path     string
	Type     string
}

type EntryFinder struct {
	ID       *int
	ParentID *int
	Path     *string
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
	EntryID int
	Name    *string
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
	EntryID int
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
