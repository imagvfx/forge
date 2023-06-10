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
	vals := make([]any, 0)
	if find.Name != nil {
		keys = append(keys, "default_properties.name=?")
		vals = append(vals, *find.Name)
	}
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	if find.DefaultID != nil {
		keys = append(keys, "properties.default_id=?")
		vals = append(vals, *find.DefaultID)
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
			evalSpecialProperty(tx, ctx, p)
		} else {
			evalProperty(tx, ctx, p)
		}
		props = append(props, p)
	}
	return props, nil
}

func propertiesByDefaultID(tx *sql.Tx, ctx context.Context, defaultID int) ([]*forge.Property, error) {
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
		WHERE properites.default_id=?`,
		defaultID,
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
			return nil, fmt.Errorf("properties from default id: %w", err)
		}
		if strings.HasPrefix(p.Name, ".") {
			evalSpecialProperty(tx, ctx, p)
		} else {
			evalProperty(tx, ctx, p)
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
		return nil, forge.NotFound("property not found: %v.%v", path, name)
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
	err = validateProperty(tx, ctx, p, nil)
	if err != nil {
		return err
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
		p.RawValue,
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
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: p.EntryPath,
		User:      user,
		Action:    "create",
		Category:  "property",
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.RawValue,
	})
	if err != nil {
		return err
	}
	return nil
}

// UpdateProperties is an efficient way of update properties of an entry, by group them in a transaction.
// If there's an error, all changes will be reverted.
func UpdateProperties(db *sql.DB, ctx context.Context, upds []forge.PropertyUpdater) error {
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
	old, err := getProperty(tx, ctx, upd.EntryPath, upd.Name)
	if err != nil {
		return err
	}
	p := &forge.Property{EntryPath: upd.EntryPath, Name: upd.Name, Type: old.Type}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if upd.Value != nil {
		p.Value = *upd.Value
		err := validateProperty(tx, ctx, p, old)
		if err != nil {
			return err
		}
		if p.RawValue != old.RawValue {
			keys = append(keys, "val=?")
			vals = append(vals, p.RawValue)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	keys = append(keys, "updated_at=?")
	vals = append(vals, time.Now().UTC())
	vals = append(vals, old.ID) // for where clause
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
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: p.EntryPath,
		User:      user,
		Action:    "update",
		Category:  "property",
		Name:      p.Name,
		Type:      p.Type,
		Value:     p.RawValue,
	})
	if err != nil {
		return nil
	}
	if p.Name == "assignee" && p.Value != "" {
		assignee := p.Value
		_, err := getAccess(tx, ctx, p.EntryPath, assignee)
		if err != nil {
			e := &forge.NotFoundError{}
			if !errors.As(err, &e) {
				return err
			}
			a := &forge.Access{
				EntryPath: p.EntryPath,
				Name:      assignee,
				Value:     "rw",
			}
			return addAccess(tx, ctx, a)
		}
		mode := "rw"
		upd := forge.AccessUpdater{
			EntryPath: p.EntryPath,
			Name:      assignee,
			Value:     &mode,
		}
		return updateAccess(tx, ctx, upd)
	}
	return nil
}

func deleteProperty(tx *sql.Tx, ctx context.Context, path, name string) error {
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	_, err = getEntry(tx, ctx, path)
	if err != nil {
		return err
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
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
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
