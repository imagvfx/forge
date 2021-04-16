package sqlite

import (
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

func FindProperties(db *sql.DB, user string, find service.PropertyFinder) ([]*service.Property, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ent, err := getEntry(tx, user, find.EntryID)
	if err != nil {
		return nil, err
	}
	props, err := findProperties(tx, user, find)
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
func findProperties(tx *sql.Tx, user string, find service.PropertyFinder) ([]*service.Property, error) {
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
	rows, err := tx.Query(`
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

func getProperty(tx *sql.Tx, user string, id int) (*service.Property, error) {
	rows, err := tx.Query(`
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

func AddProperty(db *sql.DB, user string, p *service.Property) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addProperty(tx, user, p)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addProperty(tx *sql.Tx, user string, p *service.Property) error {
	result, err := tx.Exec(`
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
	err = addLog(tx, &service.Log{
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

func UpdateProperty(db *sql.DB, user string, upd service.PropertyUpdater) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateProperty(tx, user, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateProperty(tx *sql.Tx, user string, upd service.PropertyUpdater) error {
	p, err := getProperty(tx, user, upd.ID)
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
		return fmt.Errorf("need at least one field to update property %v", upd.ID)
	}
	vals = append(vals, upd.ID) // for where clause
	result, err := tx.Exec(`
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
	err = addLog(tx, &service.Log{
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
