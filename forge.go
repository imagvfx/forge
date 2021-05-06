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
	ID           int
	Path         string
	Type         string
	HasThumbnail bool
}

func (e *Entry) Name() string {
	return filepath.Base(e.Path)
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	m := struct {
		Path       string
		SubEntries []string
	}{
		Path: e.Path,
	}
	return json.Marshal(m)
}

type Thumbnail struct {
	ID        int
	EntryID   int
	EntryPath string
	Data      []byte
}

// Property can be either a normal property or an environment.
type Property struct {
	ID        int
	EntryPath string
	Name      string
	Type      string
	Value     string
}

func (p *Property) Eval() string {
	eval := map[string]func(string) string{
		"timecode":   p.evalTimecode,
		"text":       p.evalText,
		"user":       p.evalUser,
		"entry_path": p.evalEntryPath,
		"entry_name": p.evalEntryName,
		"date":       p.evalDate,
		"int":        p.evalInt,
	}
	fn := eval[p.Type]
	if fn == nil {
		return ""
	}
	return fn(p.Value)
}

func (p *Property) Validate() error {
	validate := map[string]func(string) error{
		"timecode":   p.validateTimecode,
		"text":       p.validateText,
		"user":       p.validateUser,
		"entry_path": p.validateEntryPath,
		"entry_name": p.validateEntryName,
		"date":       p.validateDate,
		"int":        p.validateInt,
	}
	fn := validate[p.Type]
	if fn == nil {
		return fmt.Errorf("unknown type of property: %v", p.Type)
	}
	return fn(p.Value)
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
	return filepath.Clean(filepath.Join(p.EntryPath, s))
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
	return filepath.Base(filepath.Clean(filepath.Join(p.EntryPath, s)))
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

func (p *Property) evalDate(s string) string {
	return s
}

func (p *Property) validateDate(s string) error {
	if s == "" {
		// unset
		return nil
	}
	_, err := time.Parse("2006/01/02", s)
	if err != nil {
		return fmt.Errorf("invalid date string: need yyyy/mm/dd (ex: 2006/01/02)")
	}
	return nil
}

func (p *Property) evalInt(s string) string {
	return s
}

func (p *Property) validateInt(s string) error {
	if s == "" {
		// unset
		return nil
	}
	_, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("cannot convert to int")
	}
	return nil
}

func (p *Property) ServiceProperty() *service.Property {
	sp := &service.Property{
		EntryPath: p.EntryPath,
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.Value,
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
	switch l.Category {
	case "access":
		return l.AccessControlString()
	default:
		s := fmt.Sprintf("%v: %v %v %v: %v", l.When, l.User, l.Action, l.Category, l.Name)
		if l.Value != "" {
			s += fmt.Sprintf(" = %v", l.Value)
		}
		return s
	}
}

func (l *Log) AccessControlString() string {
	v, _ := strconv.Atoi(l.Value)
	if l.Action != "delete" {
		mode := AccessMode(v)
		return fmt.Sprintf("%v: %v %v %v: %v = %v", l.When, l.User, l.Action, l.Category, l.Name, mode)
	} else {
		return fmt.Sprintf("%v: %v %v %v: %v", l.When, l.User, l.Action, l.Category, l.Name)
	}
}

type User struct {
	ID   int
	Name string
}

type Group struct {
	ID   int
	Name string
}

type Member struct {
	Group  string
	Member string
}
