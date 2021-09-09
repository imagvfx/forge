package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strconv"
)

// evalProperty evaluates a value in db to a value for user.
// It also performs basic validation in case the value in db is outdated.
// Invalid value can come from manual modification of db, or formatting change to property type.
// Eg. It once saved string path of an entry for entry_path, but now saves the id.
func evalProperty(tx *sql.Tx, ctx context.Context, entry, typ, val string) (string, error) {
	evalFn := map[string]func(tx *sql.Tx, ctx context.Context, entry, val string) (string, error){
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
		return "", fmt.Errorf("unknown type of property: %v", typ)
	}
	if val == "" {
		// empty string is always accepted
		return "", nil
	}
	return eval(tx, ctx, entry, val)
}

func evalText(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	return val, nil
}

func evalUser(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	id, err := strconv.Atoi(val)
	if err != nil {
		return "", err
	}
	u, err := getUserByID(tx, ctx, id)
	if err != nil {
		return "", err
	}
	return u.Name, nil
}

func evalTimecode(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	// 00:00:00:00
	if len(val) != 11 {
		return "", fmt.Errorf("invalid value for timecode: %v", val)
	}
	if val[2] != ':' && val[5] != ':' && val[8] != ':' {
		return "", fmt.Errorf("invalid value for timecode: %v", val)
	}
	for i := 0; i < 12; i += 3 {
		_, err := strconv.Atoi(val[i : i+2])
		if err != nil {
			return "", fmt.Errorf("invalid value for timecode: %v", val)
		}
	}
	return val, nil
}

func evalEntryPath(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	id, err := strconv.Atoi(val)
	if err != nil {
		return "", err
	}
	ent, err := getEntryByID(tx, ctx, id)
	if err != nil {
		return "", err
	}
	return ent.Path, nil
}

func evalEntryName(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	id, err := strconv.Atoi(val)
	if err != nil {
		return "", err
	}
	ent, err := getEntryByID(tx, ctx, id)
	if err != nil {
		return "", err
	}
	return path.Base(ent.Path), nil
}

func evalDate(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	// 2006/01/02
	if len(val) != 10 {
		return "", fmt.Errorf("invalid value for date: %v", val)
	}
	if val[4] != '/' && val[6] != '/' {
		return "", fmt.Errorf("invalid value for date: %v", val)
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
			return "", fmt.Errorf("invalid value for date: %v", val)
		}
	}
	return val, nil
}

func evalInt(tx *sql.Tx, ctx context.Context, entry, val string) (string, error) {
	_, err := strconv.Atoi(val)
	if err != nil {
		return "", fmt.Errorf("invalid value for int: %v", val)
	}
	return val, nil
}
