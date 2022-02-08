package forge

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Entry struct {
	ID           int
	Path         string
	Type         string
	UpdatedAt    time.Time
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

type EntryFinder struct {
	ID         *int
	ParentPath *string
	Path       *string
}

type EntrySearcher struct {
	SearchRoot string
	EntryType  string
	Keywords   []string
}

type EntryTypeUpdater struct {
	ID   int
	Name *string
}

// Status indicates a status in an entry type.
// It should have css complient color information.
type Status struct {
	Name  string
	Color string
}

// Default is property, environ or sub-entry defined for entry type,
// So it will be automatically created while creation of an entry of the entry type.
type Default struct {
	ID        int
	EntryType string
	Category  string
	Name      string
	Type      string
	Value     string
}

type DefaultFinder struct {
	EntryType *string
	Category  *string
	Name      *string
}

type DefaultUpdater struct {
	EntryType string
	Category  string
	Name      string
	Type      *string
	Value     *string
}

// Global is similar with Default in a sense that it is tied to an EntryType.
// But it won't be created for each entry. So it cannot be overrided as well.
type Global struct {
	ID        int
	EntryType string
	Name      string
	Type      string
	Value     string
}

type GlobalFinder struct {
	EntryType *string
	Name      *string
}

type GlobalUpdater struct {
	EntryType string
	Name      string
	Type      *string
	Value     *string
}

type Thumbnail struct {
	ID        int
	EntryPath string
	Data      []byte
}

type ThumbnailFinder struct {
	EntryPath *string
}

type ThumbnailUpdater struct {
	EntryPath string
	Data      []byte
}

// Property can be either a normal property or an environment.
type Property struct {
	ID         int
	EntryPath  string
	Name       string
	Type       string
	Eval       string
	Value      string
	ValueError error
	RawValue   string
	UpdatedAt  time.Time
}

func (p *Property) MarshalJSON() ([]byte, error) {
	m := struct {
		Path      string
		Name      string
		Type      string
		Eval      string
		Value     string
		RawValue  string
		UpdatedAt string
	}{
		Path:      p.EntryPath,
		Name:      p.Name,
		Type:      p.Type,
		Eval:      p.Eval,
		Value:     p.Value,
		RawValue:  p.RawValue,
		UpdatedAt: p.UpdatedAt.Format(time.RFC3339),
	}
	return json.Marshal(m)
}

func CompareProperty(t string, a, b string) int {
	// TODO: simplify the logic, so we don't need `desc` in here?
	cmp := 0
	// Entry with empty value should stand behind of non-empty value always,
	// regardless of it's order type. Meaning `desc` is not affecting here.
	if a == "" {
		cmp++
	}
	if b == "" {
		cmp--
	}
	if cmp != 0 {
		return cmp
	}
	switch t {
	case "int":
		ia, erra := strconv.Atoi(a)
		ib, errb := strconv.Atoi(b)
		// show the error value first
		if erra != nil {
			cmp--
		}
		if errb != nil {
			cmp++
		}
		if cmp != 0 {
			return cmp
		}
		if ia < ib {
			cmp = -1
		} else if ia > ib {
			cmp = 1
		}
	default:
		cmp = strings.Compare(a, b)
	}
	return cmp
}

type PropertyFinder struct {
	EntryPath *string
	Name      *string
}

type PropertyUpdater struct {
	EntryPath string
	Name      string
	Value     *string
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

func (p *Property) ServiceProperty() *Property {
	sp := &Property{
		EntryPath: p.EntryPath,
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.Value,
		RawValue:  p.RawValue,
		UpdatedAt: p.UpdatedAt,
	}
	return sp
}

func AccessorTypes() []string {
	return []string{
		"user",
		"group",
	}
}

// Accessor is either a user or a group, that can be specified in entry access control list.
type Accessor struct {
	ID      int
	IsGroup bool
	Name    string
	Called  string
}

type AccessControl struct {
	ID           int
	EntryPath    string
	Accessor     string
	AccessorType string
	Mode         string
	RawMode      int
	UpdatedAt    time.Time
}

func (p *AccessControl) MarshalJSON() ([]byte, error) {
	m := struct {
		Path      string
		Name      string
		Type      string
		Value     string
		RawValue  string
		UpdatedAt time.Time
	}{
		Path:      p.EntryPath,
		Name:      p.Accessor,
		Type:      p.AccessorType,
		Value:     p.Mode,
		RawValue:  p.Mode,
		UpdatedAt: p.UpdatedAt,
	}
	return json.Marshal(m)
}

type AccessControlFinder struct {
	EntryPath *string
	Accessor  *string
}

type AccessControlUpdater struct {
	EntryPath string
	Accessor  string
	Mode      *string
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

type LogFinder struct {
	EntryPath *string
	Category  *string
	Name      *string
}

type User struct {
	ID     int
	Name   string
	Called string
}

type UserFinder struct {
	ID     *int
	Name   *string
	Called *string
}

type UserUpdater struct {
	ID     int
	Name   *string
	Called *string
}

type UserSetting struct {
	ID                          int
	User                        string
	EntryPageSelectedCategory   string
	EntryPageShowHiddenProperty string
	EntryPageSearchEntryType    string
	EntryPagePropertyFilter     map[string]string
	EntryPageSortProperty       map[string]string
	QuickSearches               []StringKV
	PinnedPaths                 []string
}

type UserSettingFinder struct {
	User *string
}

type UserSettingUpdater struct {
	User  string
	Key   string
	Value interface{}
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

type GroupFinder struct {
	Name   *string
	Called *string
}

type GroupUpdater struct {
	Name    string
	NewName *string
	Called  *string
}

type Member struct {
	Group  string
	Member string
}

type MemberFinder struct {
	Group  string
	Member *string
}

type QuickSearchArranger struct {
	Name  string
	Index int
}

// PinnedPathArranger add/move/remove the path from UserSetting.PinnedPaths.
// The index system in Arranger is a little different than the others, so please read below.
//
// The path will be removed when Index is under 0.
// Otherwise the path will be moved to the index.
// The path will be added when it doesn't already exist in the PinnedPaths.
// It will just be appeneded to the end, when Index is equal or greater than len(PinnedPaths).
//
// TODO: It could be changed to SliceItemArranger when generics come in Go.
type PinnedPathArranger struct {
	Path  string
	Index int
}
