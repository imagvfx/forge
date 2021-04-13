package forge

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/imagvfx/forge/service"
)

type Entry struct {
	srv      *Server
	id       int
	parentID int
	path     string
	typ      string
}

func (e *Entry) Path() string {
	return e.path
}

func (e *Entry) Dir() string {
	return filepath.Dir(e.path)
}

func (e *Entry) Name() string {
	return filepath.Base(e.path)
}

func (e *Entry) Type() string {
	return e.typ
}

func (e *Entry) SubEntries() ([]*Entry, error) {
	return e.srv.subEntries(e.id)
}

func (e *Entry) Properties() ([]*Property, error) {
	return e.srv.entryProperties(e.id)
}

func (e *Entry) Environs() ([]*Property, error) {
	return e.srv.entryEnvirons(e.id)
}

func (e *Entry) AccessControls() ([]*AccessControl, error) {
	return e.srv.entryAccessControls(e.id)
}

func (e *Entry) Logs() ([]*Log, error) {
	return e.srv.entryLogs(e.id)
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	m := struct {
		Path       string
		SubEntries []string
	}{
		Path: e.path,
	}
	return json.Marshal(m)
}

// Property can be either a normal property or an environment.
type Property struct {
	srv       *Server
	id        int
	entryID   int
	entryPath string
	name      string
	typ       string
	value     string
}

func (p *Property) Entry() (*Entry, error) {
	return p.srv.getEntry(p.entryID)
}

func (p *Property) Type() string {
	return p.typ
}

func (p *Property) Name() string {
	return p.name
}

func (p *Property) RawValue() string {
	return p.value
}

func (p *Property) Value() string {
	eval := map[string]func(string) string{
		"timecode":   p.evalTimecode,
		"text":       p.evalText,
		"user":       p.evalUser,
		"entry_path": p.evalEntryPath,
		"entry_name": p.evalEntryName,
	}
	fn := eval[p.typ]
	if fn == nil {
		return ""
	}
	return fn(p.value)
}

func (p *Property) Validate() error {
	validate := map[string]func(string) error{
		"timecode":   p.validateTimecode,
		"text":       p.validateText,
		"user":       p.validateUser,
		"entry_path": p.validateEntryPath,
		"entry_name": p.validateEntryName,
	}
	fn := validate[p.typ]
	if fn == nil {
		return fmt.Errorf("unknown type of property: %v", p.typ)
	}
	return fn(p.value)
}

func (p *Property) evalText(s string) string {
	return s
}

func (p *Property) validateText(s string) error {
	// every string is valid text
	return nil
}

func (p *Property) evalUser(s string) string {
	return s
}

func (p *Property) validateUser(s string) error {
	// TODO: validate when User is implemented
	return nil
}

func (p *Property) evalTimecode(s string) string {
	return s
}

func (p *Property) validateTimecode(s string) error {
	// 00:00:00:00
	if s == "" {
		// unset
		return nil
	}
	toks := strings.Split(s, ":")
	if len(toks) != 4 {
		return fmt.Errorf("invalid timecode string: %v", s)
	}
	for _, t := range toks {
		i, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("invalid timecode string: %v", s)
		}
		if i < 0 || i > 100 {
			return fmt.Errorf("invalid timecode string: %v", s)
		}
	}
	return nil
}

func (p *Property) evalEntryPath(s string) string {
	return filepath.Clean(filepath.Join(p.entryPath, s))
}

func (p *Property) validateEntryPath(s string) error {
	if s == "" {
		// unset
		return nil
	}
	if s == "." {
		// currently only . is a valid entry path.
		// other values should resolve entry renaming issue.
		return nil
	}
	return fmt.Errorf("path except . isn't valid yet")
}

func (p *Property) evalEntryName(s string) string {
	return filepath.Base(filepath.Clean(filepath.Join(p.entryPath, s)))
}

// Entry name property accepts path of an entry and returns it's name.
// So the verification is same as validateEntryPath.
func (p *Property) validateEntryName(s string) error {
	if s == "" {
		// unset
		return nil
	}
	if s == "." {
		// currently only . is a valid entry path.
		// other values should resolve entry renaming issue.
		return nil
	}
	return fmt.Errorf("path except . isn't valid yet")
}

func (p *Property) ServiceProperty() *service.Property {
	sp := &service.Property{
		EntryID:   p.entryID,
		EntryPath: p.entryPath,
		Name:      p.name,
		Type:      p.typ,
		Value:     p.value,
	}
	return sp
}

type AccessMode int

const (
	ReadAccess = AccessMode(iota)
	ReadWriteAccess
)

func (m AccessMode) String() string {
	if m == ReadAccess {
		return "r"
	}
	return "rw"
}

type AccessorType int

const (
	UserAccessor = AccessorType(iota)
	GroupAccessor
)

func (t AccessorType) String() string {
	if t == UserAccessor {
		return "user"
	}
	return "group"
}

type AccessControl struct {
	ID           int
	EntryID      int
	EntryPath    string
	Accessor     string
	AccessorType AccessorType
	Mode         AccessMode
	Members      []*User
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

func (l *Log) String() string {
	s := fmt.Sprintf("%v: %v %v %v: %v", l.When, l.User, l.Action, l.Category, l.Name)
	if l.Value != "" {
		s += fmt.Sprintf(" = %v", l.Value)
	}
	return s
}

type User struct {
	ID   int
	User string
	Name string
}
