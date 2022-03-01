package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

func createPropertiesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS properties (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER,
			default_id INTEGER NOT NULL,
			val STRING NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id),
			FOREIGN KEY (default_id) REFERENCES default_properties (id),
			UNIQUE (entry_id, default_id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_properties_entry_id ON properties (entry_id)`)
	return err
}

func EntryProperties(db *sql.DB, ctx context.Context, path string) ([]*forge.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	props, err := entryProperties(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return props, nil
}

func entryProperties(tx *sql.Tx, ctx context.Context, path string) ([]*forge.Property, error) {
	return findProperties(tx, ctx, forge.PropertyFinder{EntryPath: &path})
}

// when id is empty, it will find properties of root.
func findProperties(tx *sql.Tx, ctx context.Context, find forge.PropertyFinder) ([]*forge.Property, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.Name != nil {
		keys = append(keys, "default_properties.name=?")
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
			properties.id,
			default_properties.name,
			default_properties.type,
			properties.val,
			properties.updated_at,
			entries.path
		FROM properties
		LEFT JOIN entries ON properties.entry_id = entries.id
		LEFT JOIN default_properties ON properties.default_id = default_properties.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	props := make([]*forge.Property, 0)
	for rows.Next() {
		p := &forge.Property{}
		err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Type,
			&p.RawValue,
			&p.UpdatedAt,
			&p.EntryPath,
		)
		if err != nil {
			return nil, fmt.Errorf("find properties: %w", err)
		}
		if strings.HasPrefix(p.Name, ".") {
			p.Eval, p.Value, p.ValueError = evalSpecialProperty(tx, ctx, p.Name, p.RawValue)
		} else {
			p.Eval, p.Value, p.ValueError = evalProperty(tx, ctx, p.EntryPath, p.Type, p.RawValue)
		}
		props = append(props, p)
	}
	return props, nil
}

func GetProperty(db *sql.DB, ctx context.Context, path, name string) (*forge.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	p, err := getProperty(tx, ctx, path, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func getProperty(tx *sql.Tx, ctx context.Context, path, name string) (*forge.Property, error) {
	props, err := findProperties(tx, ctx, forge.PropertyFinder{EntryPath: &path, Name: &name})
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return nil, forge.NotFound("property not found")
	}
	p := props[0]
	return p, nil
}

func addProperty(tx *sql.Tx, ctx context.Context, p *forge.Property) error {
	err := userWrite(tx, ctx, p.EntryPath)
	if err != nil {
		return err
	}
	ent, err := getEntry(tx, ctx, p.EntryPath)
	if err != nil {
		return err
	}
	d, err := getDefaultProperty(tx, ctx, ent.Type, p.Name)
	if err != nil {
		return err
	}
	if strings.HasPrefix(p.Name, ".") {
		p.Value, err = validateSpecialProperty(tx, ctx, p.Name, p.Value)
		if err != nil {
			return err
		}
	} else {
		p.Value, err = validateProperty(tx, ctx, p.EntryPath, p.Type, p.Value)
		if err != nil {
			return err
		}
	}
	entryID, err := getEntryID(tx, ctx, p.EntryPath)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO properties (
			entry_id,
			default_id,
			val,
			updated_at
		)
		VALUES (?, ?, ?, ?)
	`,
		entryID,
		d.ID,
		p.Value,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)
	user := forge.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: p.EntryPath,
		User:      user,
		Action:    "create",
		Category:  "property",
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.Value,
	})
	if err != nil {
		return err
	}
	return nil
}

// BulkUpdateProperties is an efficient way of update properties of an entry, by group them in a transaction.
// If there's an error, all changes will be reverted.
func BulkUpdateProperties(db *sql.DB, ctx context.Context, upds []forge.PropertyUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if len(upds) == 0 {
		return nil
	}
	// Before update the properties, check the updates are all related with an entry.
	// If it isn't, refuse the updates.
	path := upds[0].EntryPath
	for _, upd := range upds[1:] {
		if upd.EntryPath != path {
			return fmt.Errorf("entry path should be same among property updaters in a bulk update")
		}
	}
	for _, upd := range upds {
		err := updateProperty(tx, ctx, upd)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func UpdateProperty(db *sql.DB, ctx context.Context, upd forge.PropertyUpdater) error {
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

func updateProperty(tx *sql.Tx, ctx context.Context, upd forge.PropertyUpdater) error {
	err := userWrite(tx, ctx, upd.EntryPath)
	if err != nil {
		return err
	}
	p, err := getProperty(tx, ctx, upd.EntryPath, upd.Name)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Value != nil {
		if strings.HasPrefix(p.Name, ".") {
			*upd.Value, err = validateSpecialProperty(tx, ctx, p.Name, *upd.Value)
			if err != nil {
				return err
			}
		} else {
			*upd.Value, err = validateProperty(tx, ctx, upd.EntryPath, p.Type, *upd.Value)
			if err != nil {
				return err
			}
		}
		if p.RawValue != *upd.Value {
			keys = append(keys, "val=?")
			vals = append(vals, *upd.Value)
			p.Value = *upd.Value // for logging
		}
	}
	if len(keys) == 0 {
		return nil
	}
	keys = append(keys, "updated_at=?")
	vals = append(vals, time.Now().UTC())
	vals = append(vals, p.ID) // for where clause
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
	user := forge.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: p.EntryPath,
		User:      user,
		Action:    "update",
		Category:  "property",
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.Value,
	})
	if err != nil {
		return nil
	}
	return nil
}

func deleteProperty(tx *sql.Tx, ctx context.Context, path, name string) error {
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	ent, err := getEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	_, err = getDefaultProperty(tx, ctx, ent.Type, name)
	if err == nil {
		return fmt.Errorf("cannot delete default property of %q: %v", ent.Type, name)
	}
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return err
		}
	}
	p, err := getProperty(tx, ctx, path, name)
	if err != nil {
		return err
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
	user := forge.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: p.EntryPath,
		User:      user,
		Action:    "delete",
		Category:  "property",
		Name:      p.Name,
		Type:      p.Type,
	})
	if err != nil {
		return nil
	}
	return nil
}
