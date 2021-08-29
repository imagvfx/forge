package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
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
	return eval(tx, ctx, path, val)
}

func evalText(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalUser(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalTimecode(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalEntryPath(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	if val == "" {
		return "", nil
	}
	return filepath.Clean(filepath.Join(path, val)), nil
}

func evalEntryName(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	if val == "" {
		return "", nil
	}
	return filepath.Base(filepath.Clean(filepath.Join(path, val))), nil
}

func evalDate(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}

func evalInt(tx *sql.Tx, ctx context.Context, path, val string) (string, error) {
	return val, nil
}
