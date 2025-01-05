package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/imagvfx/forge"
)

// evalProperty evaluates a value in db to a value for user.
// It also performs basic validation in case the value in db is outdated.
// Invalid value can come from manual modification of db, or formatting change to property type.
// Eg. It once saved string path of an entry for entry_path, but now saves the id.
func evalProperty(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	handled := evalSpecialProperty(tx, ctx, p)
	if handled {
		return
	}
	evalFn := map[string]func(tx *sql.Tx, ctx context.Context, p *forge.Property){
		"timecode":   evalTimecode,
		"text":       evalText,
		"user":       evalUser,
		"entry_path": evalEntryPath,
		"entry_name": evalEntryName,
		"entry_link": evalEntryLink,
		"date":       evalDate,
		"int":        evalInt,
		"tag":        evalTag,
		"search":     evalSearch,
	}
	eval := evalFn[p.Type]
	if eval == nil {
		p.ValueError = fmt.Errorf("unknown type of property: %v", p.Type)
		return
	}
	if p.RawValue == "" {
		// empty string is always accepted
		return
	}
	eval(tx, ctx, p)
}

func evalText(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	val := p.RawValue
	p.Eval = val
	p.Value = val
}

func evalUser(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	id, err := strconv.Atoi(p.RawValue)
	if err != nil {
		p.ValueError = err
		return
	}
	u, err := getUserByID(tx, ctx, id)
	if err != nil {
		p.ValueError = err
		return
	}
	called := u.Called
	if called == "" {
		called = u.Name
	}
	p.Eval = called
	p.Value = u.Name
}

func evalTimecode(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	// 00:00:00:00
	val := p.RawValue
	if len(val) != 11 {
		p.ValueError = fmt.Errorf("invalid value for timecode: %v", val)
		return
	}
	if val[2] != ':' && val[5] != ':' && val[8] != ':' {
		p.ValueError = fmt.Errorf("invalid value for timecode: %v", val)
		return
	}
	for i := 0; i < 12; i += 3 {
		_, err := strconv.Atoi(val[i : i+2])
		if err != nil {
			p.ValueError = fmt.Errorf("invalid value for timecode: %v", val)
			return
		}
	}
	p.Eval = val
	p.Value = val
}

func evalEntryPath(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	id, err := strconv.Atoi(p.RawValue)
	if err != nil {
		p.ValueError = err
		return
	}
	if id == 0 {
		// 0 is a special id that indicates the entry itself.
		p.Eval = p.EntryPath
		p.Value = "."
		return
	}
	ent, err := getEntryByID(tx, ctx, id)
	if err != nil {
		p.ValueError = err
		return
	}
	pth, err := filepath.Rel(p.EntryPath, ent.Path)
	if err != nil {
		p.ValueError = err
		return
	}
	p.Eval = ent.Path
	p.Value = pth
}

func evalEntryName(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	evalEntryPath(tx, ctx, p)
	p.Eval = filepath.Base(p.Eval)
	p.Value = filepath.Base(p.Value)
}

func evalEntryLink(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	eval := ""
	raw := strings.TrimSpace(p.RawValue)
	for _, pth := range strings.Split(raw, "\n") {
		if eval != "" {
			eval += "\n"
		}
		pth = strings.TrimSpace(pth)
		eval += pth
	}
	p.Eval = eval
	p.Value = eval
}

func evalDate(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	// 2006/01/02
	val := p.RawValue
	if len(val) != 10 {
		p.ValueError = fmt.Errorf("invalid value for date: %v", val)
		return
	}
	if val[4] != '/' && val[6] != '/' {
		p.ValueError = fmt.Errorf("invalid value for date: %v", val)
		return
	}
	blocks := [][2]int{
		{0, 4},  // 2006
		{5, 7},  // 01
		{8, 10}, // 02
	}
	for _, b := range blocks {
		start := b[0]
		end := b[1]
		_, err := strconv.Atoi(val[start:end])
		if err != nil {
			p.ValueError = fmt.Errorf("invalid value for date: %v", val)
			return
		}
	}
	p.Eval = val
	p.Value = val
}

func evalInt(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	val := p.RawValue
	_, err := strconv.Atoi(val)
	if err != nil {
		p.ValueError = fmt.Errorf("invalid value for int: %v", val)
		return
	}
	p.Eval = val
	p.Value = val
}

func evalTag(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	p.Eval = strings.TrimSpace(p.RawValue)
	p.Value = strings.TrimSpace(p.RawValue)
}

func evalSearch(tx *sql.Tx, ctx context.Context, p *forge.Property) {
	p.Eval = p.RawValue
	p.Value = p.RawValue
}

// evalSpecialProperty evaluates special properties that defined in forge.
// It will return true when given property was special property.
func evalSpecialProperty(tx *sql.Tx, ctx context.Context, p *forge.Property) bool {
	switch p.Name {
	case ".predefined_sub_entries":
		subNameType := make(map[string]string)
		for _, nt := range strings.Split(p.RawValue, ",") {
			nt = strings.TrimSpace(nt)
			toks := strings.Split(nt, ":")
			if len(toks) != 2 {
				p.ValueError = fmt.Errorf(".predefined_sub_entries value should consists of 'subent:type' tokens: %v", nt)
				return true
			}
			sub := strings.TrimSpace(toks[0])
			typeID := strings.TrimSpace(toks[1])
			id, err := strconv.Atoi(typeID)
			if err != nil {
				p.ValueError = fmt.Errorf("invalid entry type id for '%v' in .predefined_sub_entries: %v", p.Name, typeID)
				return true
			}
			// Internally it saves with entry type id. Get the type name.
			typ, err := getEntryTypeByID(tx, ctx, id)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					p.ValueError = err
					return true
				}
				p.ValueError = fmt.Errorf("not found the entry type defined for '%v' in .predefined_sub_entries: %v", p.Name, typ)
				return true
			}
			subNameType[sub] = typ
		}
		subNames := make([]string, 0, len(subNameType))
		for sub := range subNameType {
			subNames = append(subNames, sub)
		}
		sort.Slice(subNames, func(i, j int) bool { return subNames[i] < subNames[j] })
		val := ""
		for i, sub := range subNames {
			if i != 0 {
				val += ", "
			}
			val += sub + ":" + subNameType[sub]
		}
		p.Eval = val
		p.Value = val
		return true
	case ".sub_entry_types":
		p.Eval = p.RawValue
		p.Value = p.RawValue
		return true
	}
	return false
}
