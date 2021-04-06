package service

import "time"

type Service interface {
	FindEntries(EntryFinder) ([]*Entry, error)
	GetEntry(int) (*Entry, error)
	AddEntry(*Entry, []*Property, []*Property) error
	FindProperties(PropertyFinder) ([]*Property, error)
	AddProperty(*Property) error
	UpdateProperty(PropertyUpdater) error
	FindEnvirons(PropertyFinder) ([]*Property, error)
	AddEnviron(*Property) error
	UpdateEnviron(PropertyUpdater) error
	FindLogs(LogFinder) ([]*Log, error)
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

type Log struct {
	ID       int
	EntryID  int
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
