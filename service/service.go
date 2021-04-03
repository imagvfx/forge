package service

type Service interface {
	FindEntries(EntryFinder) ([]*Entry, error)
	GetEntry(int) (*Entry, error)
	AddEntry(*Entry) error
	FindProperties(PropertyFinder) ([]*Property, error)
	AddProperty(*Property) error
	UpdateProperty(PropertyUpdater) error
	FindEnvirons(EnvironFinder) ([]*Environ, error)
	AddEnviron(*Environ) error
	UpdateEnviron(EnvironUpdater) error
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
	ID      int
	EntryID int
	Name    string
	Type    string
	Value   string
}

type PropertyFinder struct {
	EntryID int
	Name    *string
}

type PropertyUpdater struct {
	ID    int
	Value *string
}

type Environ struct {
	ID      int
	EntryID int
	Name    string
	Value   string
}

type EnvironFinder struct {
	EntryID int
	Name    *string
}

type EnvironUpdater struct {
	ID    int
	Value *string
}
