package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

func createEnvironsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS environs (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER,
			name STRING NOT NULL,
			typ STRING NOT NULL,
			val STRING NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id),
			UNIQUE (entry_id, name)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_environs_entry_id ON environs (entry_id)`)
	return err
}

func EntryEnvirons(db *sql.DB, ctx context.Context, path string) ([]*forge.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	envs, err := entryEnvirons(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return envs, nil
}

func entryEnvirons(tx *sql.Tx, ctx context.Context, path string) ([]*forge.Property, error) {
	envmap := make(map[string]*forge.Property)
	for {
		envs, err := findEnvirons(tx, ctx, forge.PropertyFinder{EntryPath: &path})
		if err != nil {
			return nil, err
		}
		for _, e := range envs {
			if envmap[e.Name] == nil {
				envmap[e.Name] = e
			}
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	envs := make([]*forge.Property, 0, len(envmap))
	for _, e := range envmap {
		envs = append(envs, e)
	}
	return envs, nil
}

// GetEnvirons get environs defined in an entry.
// It doesn't count inherited environs unlike EntryEnvirons.
func GetEnvirons(db *sql.DB, ctx context.Context, path string) ([]*forge.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	envs, err := findEnvirons(tx, ctx, forge.PropertyFinder{EntryPath: &path})
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return envs, nil
}

// when id is empty, it will find environs of root.
// It returns a map instead of a slice, because it is better structure for aggregating the parents` environs.
func findEnvirons(tx *sql.Tx, ctx context.Context, find forge.PropertyFinder) ([]*forge.Property, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	if find.Name != nil {
		keys = append(keys, "environs.name=?")
		vals = append(vals, *find.Name)
	}
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			environs.id,
			environs.name,
			environs.typ,
			environs.val,
			environs.updated_at,
			entries.path
		FROM environs
		LEFT JOIN entries ON environs.entry_id = entries.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	envs := make([]*forge.Property, 0)
	for rows.Next() {
		e := &forge.Property{}
		err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Type,
			&e.RawValue,
			&e.UpdatedAt,
			&e.EntryPath,
		)
		if err != nil {
			return nil, err
		}
		evalProperty(tx, ctx, e)
		envs = append(envs, e)
	}
	return envs, nil
}

func GetEnviron(db *sql.DB, ctx context.Context, path, name string) (*forge.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := getEnviron(tx, ctx, path, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func getEnviron(tx *sql.Tx, ctx context.Context, path, name string) (*forge.Property, error) {
	envs, err := findEnvirons(tx, ctx, forge.PropertyFinder{EntryPath: &path, Name: &name})
	if err != nil {
		return nil, err
	}
	if len(envs) == 0 {
		return nil, forge.NotFound("environ not found")
	}
	return envs[0], nil
}

func AddEnviron(db *sql.DB, ctx context.Context, e *forge.Property) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEnviron(tx, ctx, e)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEnviron(tx *sql.Tx, ctx context.Context, e *forge.Property) error {
	err := userWrite(tx, ctx, e.EntryPath)
	if err != nil {
		return err
	}
	err = validateProperty(tx, ctx, e, nil)
	if err != nil {
		return err
	}
	entryID, err := getEntryID(tx, ctx, e.EntryPath)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO environs (
			entry_id,
			name,
			typ,
			val,
			updated_at
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		entryID,
		e.Name,
		e.Type,
		e.RawValue,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = int(id)
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: e.EntryPath,
		User:      user,
		Action:    "create",
		Category:  "environ",
		Name:      e.Name,
		Type:      e.Type,
		Value:     e.Value,
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateEnviron(db *sql.DB, ctx context.Context, upd forge.PropertyUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateEnviron(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateEnviron(tx *sql.Tx, ctx context.Context, upd forge.PropertyUpdater) error {
	err := userWrite(tx, ctx, upd.EntryPath)
	if err != nil {
		return err
	}
	old, err := getEnviron(tx, ctx, upd.EntryPath, upd.Name)
	if err != nil {
		return err
	}
	e := &forge.Property{EntryPath: upd.EntryPath, Name: upd.Name, Type: old.Type}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if upd.Value != nil {
		e.Value = *upd.Value
		err = validateProperty(tx, ctx, e, old)
		if err != nil {
			return err
		}
		if e.RawValue != old.RawValue {
			keys = append(keys, "val=?")
			vals = append(vals, e.RawValue)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	keys = append(keys, "updated_at=?")
	vals = append(vals, time.Now().UTC())
	vals = append(vals, old.ID) // for where clause
	result, err := tx.ExecContext(ctx, `
		UPDATE environs
		SET `+strings.Join(keys, ", ")+`
		WHERE id=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 property affected, got %v", n)
	}
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: e.EntryPath,
		User:      user,
		Action:    "update",
		Category:  "environ",
		Name:      e.Name,
		Type:      e.Type,
		Value:     e.RawValue,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeleteEnviron(db *sql.DB, ctx context.Context, path, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteEnviron(tx, ctx, path, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteEnviron(tx *sql.Tx, ctx context.Context, path, name string) error {
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	_, err = getEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	e, err := getEnviron(tx, ctx, path, name)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM environs
		WHERE id=?
	`,
		e.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 environ affected, got %v", n)
	}
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: path,
		User:      user,
		Action:    "delete",
		Category:  "environ",
		Name:      e.Name,
		Type:      e.Type,
	})
	if err != nil {
		return err
	}
	return nil
}
