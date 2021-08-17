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

// Default is property, environ or sub-entry defined for entry type,
// So it will be automatically created while creation of an entry of the entry type.
type Default struct {
	EntryType string
	Category  string
	Name      string
	Type      string
	Value     string
}

type Thumbnail struct {
	ID        int
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

func (p *Property) MarshalJSON() ([]byte, error) {
	m := struct {
		Path     string
		Name     string
		Type     string
		Value    string
		RawValue string
	}{
		Path:     p.EntryPath,
		Name:     p.Name,
		Type:     p.Type,
		Value:    p.Eval(),
		RawValue: p.Value,
	}
	return json.Marshal(m)
}

func LessProperty(t, a, b string) bool {
	switch t {
	case "int":
		ia, erra := strconv.Atoi(a)
		ib, errb := strconv.Atoi(b)
		// show the error value first
		if erra != nil {
			return true
		}
		if errb != nil {
			return false
		}
		return ia < ib
	}
	return a < b
}

func PropertyTypes() []string {
	return []string{
		"text",
		// sort by name except text
		"date",
		"entry_path",
		"entry_name",
		"int",
		"timecode",
		"user",
	}
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
	validate := map[string]func(string) (string, error){
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
	corrected, err := fn(p.Value)
	if err != nil {
		return err
	}
	p.Value = corrected
	return nil
}

func (p *Property) evalText(s string) string {
	return s
}

func (p *Property) validateText(s string) (string, error) {
	// every string is valid text
	return s, nil
}

func (p *Property) evalUser(s string) string {
	return s
}

func (p *Property) validateUser(s string) (string, error) {
	// TODO: validate when User is implemented
	return s, nil
}

func (p *Property) evalTimecode(s string) string {
	return s
}

func (p *Property) validateTimecode(s string) (string, error) {
	// 00:00:00:00
	if s == "" {
		// unset
		return s, nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true,
	}
	ss := ""
	for _, r := range s {
		ch := string(r)
		if isDigit[ch] {
			ss += ch
		}
	}
	if len(ss) != 8 {
		return "", fmt.Errorf("invalid timecode string: %v", s)
	}
	s = strings.Join(
		[]string{
			ss[0:2], ss[2:4], ss[4:6], ss[6:8],
		},
		":",
	)
	return s, nil
}

func (p *Property) evalEntryPath(s string) string {
	if s == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(p.EntryPath, s))
}

func (p *Property) validateEntryPath(s string) (string, error) {
	if s == "" {
		// unset
		return s, nil
	}
	if s == "." {
		// currently only . is a valid entry path.
		// other values should resolve entry renaming issue.
		return s, nil
	}
	return "", fmt.Errorf("path except . isn't valid yet")
}

func (p *Property) evalEntryName(s string) string {
	if s == "" {
		return ""
	}
	return filepath.Base(filepath.Clean(filepath.Join(p.EntryPath, s)))
}

// Entry name property accepts path of an entry and returns it's name.
// So the verification is same as validateEntryPath.
func (p *Property) validateEntryName(s string) (string, error) {
	if s == "" {
		// unset
		return s, nil
	}
	if s == "." {
		// currently only . is a valid entry path.
		// other values should resolve entry renaming issue.
		return s, nil
	}
	return "", fmt.Errorf("path except . isn't valid yet")
}

func (p *Property) evalDate(s string) string {
	return s
}

func (p *Property) validateDate(s string) (string, error) {
	if s == "" {
		// unset
		return s, nil
	}
	// Need 8 digits in what ever form.
	isDigit := map[string]bool{
		"0": true, "1": true, "2": true, "3": true, "4": true,
		"5": true, "6": true, "7": true, "8": true, "9": true,
	}
	ss := ""
	for _, r := range s {
		ch := string(r)
		if isDigit[ch] {
			ss += ch
		}
	}
	if len(ss) != 8 {
		return "", fmt.Errorf("invalid date string: want yyyy/mm/dd, got %v", s)
	}
	s = strings.Join(
		[]string{
			ss[0:4], ss[4:6], ss[6:8],
		},
		"/",
	)
	_, err := time.Parse("2006/01/02", s)
	if err != nil {
		return "", fmt.Errorf("invalid date string: %v", err)
	}
	return s, nil
}

func (p *Property) evalInt(s string) string {
	return s
}

func (p *Property) validateInt(s string) (string, error) {
	if s == "" {
		// unset
		return s, nil
	}
	_, err := strconv.Atoi(s)
	if err != nil {
		return "", fmt.Errorf("cannot convert to int")
	}
	return s, nil
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

func AccessorTypes() []string {
	return []string{
		"user",
		"group",
	}
}

func (t AccessorType) String() string {
	if t == UserAccessor {
		return "user"
	}
	return "group"
}

type AccessControl struct {
	ID           int
	EntryPath    string
	Accessor     string
	AccessorType AccessorType
	Mode         AccessMode
}

func (p *AccessControl) MarshalJSON() ([]byte, error) {
	m := struct {
		Path     string
		Name     string
		Type     string
		Value    string
		RawValue string
	}{
		Path:     p.EntryPath,
		Name:     p.Accessor,
		Type:     p.AccessorType.String(),
		Value:    p.Mode.String(),
		RawValue: p.Mode.String(),
	}
	return json.Marshal(m)
}

type Log struct {
	ID        int
	EntryPath string
	User      string
	Action    string
	Category  string
	Name      string
	Type      string
	Value     string
	When      time.Time
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

type UserSetting struct {
	ID                       int
	User                     string
	EntryPageSearchEntryType string
	EntryPagePropertyFilter  map[string]string
	EntryPageSortProperty    map[string]string
	EntryPageQuickSearch     map[string]string
	PinnedPaths              []string
}

type Group struct {
	ID   int
	Name string
}

type Member struct {
	Group  string
	Member string
}
