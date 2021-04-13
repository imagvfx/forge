package service

import (
	"time"
)

type Service interface {
	FindEntries(EntryFinder) ([]*Entry, error)
	GetEntry(int) (*Entry, error)
	AddEntry(string, *Entry, []*Property, []*Property) error
	FindProperties(PropertyFinder) ([]*Property, error)
	AddProperty(string, *Property) error
	UpdateProperty(string, PropertyUpdater) error
	FindEnvirons(PropertyFinder) ([]*Property, error)
	AddEnviron(string, *Property) error
	UpdateEnviron(string, PropertyUpdater) error
	FindAccessControls(AccessControlFinder) ([]*AccessControl, error)
	AddAccessControl(string, *AccessControl) error
	UpdateAccessControl(string, AccessControlUpdater) error
	FindLogs(LogFinder) ([]*Log, error)
	AddUser(*User) error
	GetUserByUser(string) (*User, error)
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
	Members      []*User
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
