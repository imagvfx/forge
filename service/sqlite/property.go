package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createPropertiesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS properties (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER,
			name STRING NOT NULL,
			typ STRING NOT NULL,
			val STRING NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id),
			UNIQUE (entry_id, name)
		)
	`)
	return err
}

func FindProperties(db *sql.DB, ctx context.Context, find service.PropertyFinder) ([]*service.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ent, err := getEntry(tx, ctx, find.EntryID)
	if err != nil {
		return nil, err
	}
	props, err := findProperties(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	for _, p := range props {
		p.EntryPath = ent.Path
	}
	sort.Slice(props, func(i, j int) bool {
		return props[i].Name < props[j].Name
	})
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return props, nil
}

// when id is empty, it will find properties of root.
func findProperties(tx *sql.Tx, ctx context.Context, find service.PropertyFinder) ([]*service.Property, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "entry_id=?")
	vals = append(vals, find.EntryID)
	if find.Name != nil {
		keys = append(keys, "name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			entry_id,
			name,
			typ,
			val
		FROM properties
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	props := make([]*service.Property, 0)
	for rows.Next() {
		p := &service.Property{}
		err := rows.Scan(
			&p.ID,
			&p.EntryID,
			&p.Name,
			&p.Type,
			&p.Value,
		)
		if err != nil {
			return nil, err
		}
		props = append(props, p)
	}
	return props, nil
}

func getProperty(tx *sql.Tx, ctx context.Context, id int) (*service.Property, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			entry_id,
			name,
			typ,
			val
		FROM properties
		WHERE id=?`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("property not found")
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

func getPropertyByPathName(tx *sql.Tx, ctx context.Context, path, name string) (*service.Property, error) {
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
		FROM properties
		WHERE entry_id=? AND name=?`,
		e.ID,
		name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("property not found")
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

func AddProperty(db *sql.DB, ctx context.Context, p *service.Property) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addProperty(tx, ctx, p)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addProperty(tx *sql.Tx, ctx context.Context, p *service.Property) error {
	ok, err := userCanWrite(tx, ctx, p.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO properties (
			entry_id,
			typ,
			name,
			val
		)
		VALUES (?, ?, ?, ?)
	`,
		p.EntryID,
		p.Type,
		p.Name,
		p.Value,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  p.EntryID,
		User:     user,
		Action:   "create",
		Category: "property",
		Name:     p.Name,
		Type:     p.Type,
		Value:    p.Value,
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateProperty(db *sql.DB, ctx context.Context, upd service.PropertyUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateProperty(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateProperty(tx *sql.Tx, ctx context.Context, upd service.PropertyUpdater) error {
	p, err := getProperty(tx, ctx, upd.ID)
	if err != nil {
		return err
	}
	ok, err := userCanWrite(tx, ctx, p.EntryID)
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
		p.Value = *upd.Value // for logging
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update property %v", upd.ID)
	}
	vals = append(vals, upd.ID) // for where clause
	result, err := tx.ExecContext(ctx, `
		UPDATE properties
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
		EntryID:  p.EntryID,
		User:     user,
		Action:   "update",
		Category: "property",
		Name:     p.Name,
		Type:     p.Type,
		Value:    p.Value,
	})
	if err != nil {
		return nil
	}
	return nil
}

func DeleteProperty(db *sql.DB, ctx context.Context, path, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteProperty(tx, ctx, path, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteProperty(tx *sql.Tx, ctx context.Context, path, name string) error {
	p, err := getPropertyByPathName(tx, ctx, path, name)
	if err != nil {
		return err
	}
	ok, err := userCanWrite(tx, ctx, p.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM properties
		WHERE id=?
	`,
		p.ID,
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
		EntryID:  p.EntryID,
		User:     user,
		Action:   "delete",
		Category: "property",
		Name:     p.Name,
		Type:     p.Type,
	})
	if err != nil {
		return nil
	}
	return nil
}
