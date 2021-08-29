package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
)

func evalProperty(tx *sql.Tx, ctx context.Context, path, typ, val string) (string, error) {
	evalFn := map[string]func(tx *sql.Tx, ctx context.Context, path, val string) (string, error){
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
	return eval(tx, ctx, path, val)
}

func evalText(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalUser(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
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

func evalTimecode(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalEntryPath(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return filepath.Clean(filepath.Join(path, val)), nil
}

func evalEntryName(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return filepath.Base(filepath.Clean(filepath.Join(path, val))), nil
}

func evalDate(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalInt(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}
