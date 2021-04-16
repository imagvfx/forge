package service

import (
	"time"
)

type Service interface {
	FindEntries(string, EntryFinder) ([]*Entry, error)
	GetEntry(string, int) (*Entry, error)
	AddEntry(string, *Entry, []*Property, []*Property) error
	FindProperties(string, PropertyFinder) ([]*Property, error)
	AddProperty(string, *Property) error
	UpdateProperty(string, PropertyUpdater) error
	FindEnvirons(string, PropertyFinder) ([]*Property, error)
	AddEnviron(string, *Property) error
	UpdateEnviron(string, PropertyUpdater) error
	FindAccessControls(string, AccessControlFinder) ([]*AccessControl, error)
	AddAccessControl(string, *AccessControl) error
	UpdateAccessControl(string, AccessControlUpdater) error
	FindLogs(string, LogFinder) ([]*Log, error)
	AddUser(*User) error
	GetUserByUser(string) (*User, error)
	FindGroups(GroupFinder) ([]*Group, error)
	AddGroup(string, *Group) error
	UpdateGroup(string, GroupUpdater) error
	FindGroupMembers(MemberFinder) ([]*Member, error)
	AddGroupMember(string, *Member) error
	DeleteGroupMember(string, int) error
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
	Path     string
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
	ID   int
	User string
	Name string
}

type UserFinder struct {
	ID   *int
	User *string
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
