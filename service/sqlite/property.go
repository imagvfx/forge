package sqlite

import (
	"context"
	"database/sql"
	"fmt"
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
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_properties_entry_id ON properties (entry_id)`)
	return err
}

func EntryProperties(db *sql.DB, ctx context.Context, path string) ([]*service.Property, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	props, err := findProperties(tx, ctx, service.PropertyFinder{EntryPath: &path})
	if err != nil {
		return nil, err
	}
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
	if find.Name != nil {
		keys = append(keys, "properties.name=?")
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
			properties.name,
			properties.typ,
			properties.val,
			entries.path
		FROM properties
		LEFT JOIN entries ON properties.entry_id = entries.id
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
			&p.Name,
			&p.Type,
			&p.Value,
			&p.EntryPath,
		)
		if err != nil {
			return nil, err
		}
		props = append(props, p)
	}
	return props, nil
}

func GetProperty(db *sql.DB, ctx context.Context, path, name string) (*service.Property, error) {
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

func getProperty(tx *sql.Tx, ctx context.Context, path, name string) (*service.Property, error) {
	props, err := findProperties(tx, ctx, service.PropertyFinder{EntryPath: &path, Name: &name})
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return nil, service.NotFound("property not found")
	}
	p := props[0]
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
	err := userWrite(tx, ctx, p.EntryPath)
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
			typ,
			name,
			val
		)
		VALUES (?, ?, ?, ?)
	`,
		entryID,
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
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
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
		keys = append(keys, "val=?")
		vals = append(vals, *upd.Value)
		p.Value = *upd.Value // for logging
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update property %v:%v", upd.EntryPath, upd.Name)
	}
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
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
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
	err := userWrite(tx, ctx, path)
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
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
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
