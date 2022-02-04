package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
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
func evalProperty(tx *sql.Tx, ctx context.Context, entry, typ, val string) (string, string, error) {
	evalFn := map[string]func(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error){
		"timecode":   evalTimecode,
		"text":       evalText,
		"user":       evalUser,
		"entry_path": evalEntryPath,
		"entry_name": evalEntryName,
		"date":       evalDate,
		"int":        evalInt,
	}
	eval := evalFn[typ]
	if eval == nil {
		return "", "", fmt.Errorf("unknown type of property: %v", typ)
	}
	if val == "" {
		// empty string is always accepted
		return "", "", nil
	}
	return eval(tx, ctx, entry, val)
}

func evalText(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	return val, val, nil
}

func evalUser(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	id, err := strconv.Atoi(val)
	if err != nil {
		return "", "", err
	}
	u, err := getUserByID(tx, ctx, id)
	if err != nil {
		return "", "", err
	}
	return u.Called, u.Name, nil
}

func evalTimecode(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	// 00:00:00:00
	if len(val) != 11 {
		return "", "", fmt.Errorf("invalid value for timecode: %v", val)
	}
	if val[2] != ':' && val[5] != ':' && val[8] != ':' {
		return "", "", fmt.Errorf("invalid value for timecode: %v", val)
	}
	for i := 0; i < 12; i += 3 {
		_, err := strconv.Atoi(val[i : i+2])
		if err != nil {
			return "", "", fmt.Errorf("invalid value for timecode: %v", val)
		}
	}
	return val, val, nil
}

func evalEntryPath(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	id, err := strconv.Atoi(val)
	if err != nil {
		return "", "", err
	}
	ent, err := getEntryByID(tx, ctx, id)
	if err != nil {
		return "", "", err
	}
	pth, err := filepath.Rel(entry, ent.Path)
	if err != nil {
		return "", "", err
	}
	return ent.Path, pth, nil
}

func evalEntryName(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	id, err := strconv.Atoi(val)
	if err != nil {
		return "", "", err
	}
	ent, err := getEntryByID(tx, ctx, id)
	if err != nil {
		return "", "", err
	}
	pth, err := filepath.Rel(entry, ent.Path)
	if err != nil {
		return "", "", err
	}
	return path.Base(ent.Path), pth, nil
}

func evalDate(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	// 2006/01/02
	if len(val) != 10 {
		return "", "", fmt.Errorf("invalid value for date: %v", val)
	}
	if val[4] != '/' && val[6] != '/' {
		return "", "", fmt.Errorf("invalid value for date: %v", val)
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
			return "", "", fmt.Errorf("invalid value for date: %v", val)
		}
	}
	return val, val, nil
}

func evalInt(tx *sql.Tx, ctx context.Context, entry, val string) (string, string, error) {
	_, err := strconv.Atoi(val)
	if err != nil {
		return "", "", fmt.Errorf("invalid value for int: %v", val)
	}
	return val, val, nil
}

// evalSpecialProperty evaluates special properties that starts with dot (.).
// For normal properties, it will return the input value unmodified.
//
// TODO: Match the arugments to other eval functions?
func evalSpecialProperty(tx *sql.Tx, ctx context.Context, name, val string) (string, string, error) {
	// TODO: .sub_entry_types
	switch name {
	case ".predefined_sub_entries":
		subNameType := make(map[string]string)
		for _, nt := range strings.Split(val, ",") {
			nt = strings.TrimSpace(nt)
			toks := strings.Split(nt, ":")
			if len(toks) != 2 {
				return "", "", fmt.Errorf(".predefined_sub_entries value should consists of 'subent:type' tokens: %v", nt)
			}
			sub := strings.TrimSpace(toks[0])
			typeID := strings.TrimSpace(toks[1])
			id, err := strconv.Atoi(typeID)
			if err != nil {
				return "", "", fmt.Errorf("invalid entry type id for '%v' in .predefined_sub_entries: %v", name, typeID)
			}
			// Internally it saves with entry type id. Get the type name.
			typ, err := getEntryTypeByID(tx, ctx, id)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return "", "", err
				}
				return "", "", fmt.Errorf("not found the entry type defined for '%v' in .predefined_sub_entries: %v", name, typ)
			}
			subNameType[sub] = typ
		}
		subNames := make([]string, 0, len(subNameType))
		for sub := range subNameType {
			subNames = append(subNames, sub)
		}
		sort.Slice(subNames, func(i, j int) bool { return subNames[i] < subNames[j] })
		val = ""
		for i, sub := range subNames {
			if i != 0 {
				val += ", "
			}
			val += sub + ":" + subNameType[sub]
		}
		return val, val, nil
	}
	return val, val, nil
}
