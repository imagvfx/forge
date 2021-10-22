package forge

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
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

// Global is similar with Default in a sense that it is tied to an EntryType.
// But it won't be created for each entry. So it cannot be overrided as well.
type Global struct {
	EntryType string
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
	ID         int
	EntryPath  string
	Name       string
	Type       string
	Value      string
	ValueError error
	RawValue   string
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
		Value:    p.Value,
		RawValue: p.RawValue,
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

func (p *Property) ServiceProperty() *service.Property {
	sp := &service.Property{
		EntryPath: p.EntryPath,
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.Value,
		RawValue:  p.RawValue,
	}
	return sp
}

func AccessorTypes() []string {
	return []string{
		"user",
		"group",
	}
}

type AccessControl struct {
	ID           int
	EntryPath    string
	Accessor     string
	AccessorType string
	Mode         string
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
		Type:     p.AccessorType,
		Value:    p.Mode,
		RawValue: p.Mode,
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
	s := fmt.Sprintf("%v: %v %v %v: %v", l.When, l.User, l.Action, l.Category, l.Name)
	if l.Value != "" {
		s += fmt.Sprintf(" = %v", l.Value)
	}
	return s
}

type User struct {
	ID     int
	Name   string
	Called string
}

type UserSetting struct {
	ID                       int
	User                     string
	EntryPageSearchEntryType string
	EntryPagePropertyFilter  map[string]string
	EntryPageSortProperty    map[string]string
	QuickSearches            []service.StringKV
	PinnedPaths              []string
}

type StringKV struct {
	K string
	V string
}

type Group struct {
	ID     int
	Name   string
	Called string
}

type Member struct {
	Group  string
	Member string
}
