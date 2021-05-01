package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createEnvironsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS environs (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER,
			name STRING NOT NULL,
			typ STRING NOT NULL,
			val STRING NOT NULL,
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

func EntryEnvirons(db *sql.DB, ctx context.Context, path string) ([]*service.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	envmap := make(map[string]*service.Property)
	for {
		emap, err := findEnvirons(tx, ctx, service.PropertyFinder{EntryPath: &path})
		if err != nil {
			return nil, err
		}
		for name, e := range emap {
			if envmap[name] == nil {
				envmap[name] = e
			}
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	envs := make([]*service.Property, 0, len(envmap))
	for _, e := range envmap {
		envs = append(envs, e)
	}
	sort.Slice(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return envs, nil
}

// when id is empty, it will find environs of root.
// It returns a map instead of a slice, because it is better structure for aggregating the parents` environs.
func findEnvirons(tx *sql.Tx, ctx context.Context, find service.PropertyFinder) (map[string]*service.Property, error) {

	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "environs.id=?")
		vals = append(vals, *find.ID)
	}
	if find.EntryID != nil {
		keys = append(keys, "environs.entry_id=?")
		vals = append(vals, *find.EntryID)
	}
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
			environs.entry_id,
			environs.name,
			environs.typ,
			environs.val,
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
	envmap := make(map[string]*service.Property)
	for rows.Next() {
		e := &service.Property{}
		err := rows.Scan(
			&e.ID,
			&e.EntryID,
			&e.Name,
			&e.Type,
			&e.Value,
			&e.EntryPath,
		)
		if err != nil {
			return nil, err
		}
		envmap[e.Name] = e
	}
	return envmap, nil
}

func GetEnviron(db *sql.DB, ctx context.Context, path, name string) (*service.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := getEnvironByPathName(tx, ctx, path, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func getEnviron(tx *sql.Tx, ctx context.Context, id int) (*service.Property, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			entry_id,
			name,
			typ,
			val
		FROM environs
		WHERE id=?`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("environ not found")
	}
	e := &service.Property{}
	err = rows.Scan(
		&e.ID,
		&e.EntryID,
		&e.Name,
		&e.Type,
		&e.Value,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func getEnvironByPathName(tx *sql.Tx, ctx context.Context, path, name string) (*service.Property, error) {
	e, err := getEntryByPath(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			entry_id,
			name,
			typ,
			val
		FROM environs
		WHERE entry_id=? AND name=?`,
		e.ID,
		name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("environ not found")
	}
	p := &service.Property{}
	err = rows.Scan(
		&p.ID,
		&p.EntryID,
		&p.Name,
		&p.Type,
		&p.Value,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func AddEnviron(db *sql.DB, ctx context.Context, e *service.Property) error {
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

func addEnviron(tx *sql.Tx, ctx context.Context, e *service.Property) error {
	ok, err := userCanWrite(tx, ctx, e.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO environs (
			entry_id,
			name,
			typ,
			val
		)
		VALUES (?, ?, ?, ?)
	`,
		e.EntryID,
		e.Name,
		e.Type,
		e.Value,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = int(id)
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  e.EntryID,
		User:     user,
		Action:   "create",
		Category: "environ",
		Name:     e.Name,
		Type:     e.Type,
		Value:    e.Value,
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateEnviron(db *sql.DB, ctx context.Context, upd service.PropertyUpdater) error {
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

func updateEnviron(tx *sql.Tx, ctx context.Context, upd service.PropertyUpdater) error {
	e, err := getEnviron(tx, ctx, upd.ID)
	if err != nil {
		return err
	}
	ok, err := userCanWrite(tx, ctx, e.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Value != nil {
		keys = append(keys, "val=?")
		vals = append(vals, *upd.Value)
		e.Value = *upd.Value // for logging
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update property %v", upd.ID)
	}
	vals = append(vals, upd.ID) // for where clause
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
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  e.EntryID,
		User:     user,
		Action:   "update",
		Category: "environ",
		Name:     e.Name,
		Type:     e.Type,
		Value:    e.Value,
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
	e, err := getEnvironByPathName(tx, ctx, path, name)
	if err != nil {
		return err
	}
	ok, err := userCanWrite(tx, ctx, e.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
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
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  e.EntryID,
		User:     user,
		Action:   "delete",
		Category: "environ",
		Name:     e.Name,
		Type:     e.Type,
	})
	if err != nil {
		return nil
	}
	return nil
}
