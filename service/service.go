package service

type Service interface {
	FindEntries(EntryFinder) ([]*Entry, error)
	AddEntry(*Entry) error
	FindProperties(PropertyFinder) ([]*Property, error)
	AddProperty(*Property) error
	UpdateProperty(PropertyUpdater) error
}

type Entry struct {
	ID       int
	ParentID *int // nil if root entry
	Path     string
}

type EntryFinder struct {
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
	EntryID *int
	Name    *string
}

type PropertyUpdater struct {
	ID    int
	Value *string
}
