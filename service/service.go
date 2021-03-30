package service

type Service interface {
	FindEntries(EntryFinder) ([]*Entry, error)
	AddEntry(*Entry) error
}

type service interface {
	DelEntry(int) error
	Properties(int) ([]Property, error)
	Property(int, string) (Property, error)
	AddProperty(int, Property) error
	SetProperty(int, string) error
	DelProperty(int, int) error
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
	ID    int
	Entry int
	Type  string
	Name  string
	Value string
}
