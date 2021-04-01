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
			inherit BOOL NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id),
			UNIQUE (entry_id, name)
		)
	`)
	return err
}

func FindProperties(db *sql.DB, find service.PropertyFinder) ([]*service.Property, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	propmap := make(map[string]*service.Property)
	onlyInherit := false
	for {
		ent, err := getEntry(tx, find.EntryID)
		if err != nil {
			return nil, err
		}
		pmap, err := findProperties(tx, find, onlyInherit)
		if err != nil {
			return nil, err
		}
		for name, p := range pmap {
			if propmap[name] == nil {
				propmap[name] = p
			}
		}
		if ent.ParentID == nil {
			break
		}
		find.EntryID = *ent.ParentID
		onlyInherit = true
	}
	props := make([]*service.Property, 0, len(propmap))
	for _, p := range propmap {
		props = append(props, p)
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
// It returns a map instead of a slice, because it is better structure for aggregating the parents` properties.
func findProperties(tx *sql.Tx, find service.PropertyFinder, onlyInherit bool) (map[string]*service.Property, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "entry_id=?")
	vals = append(vals, find.EntryID)
	if find.Name != nil {
		keys = append(keys, "name=?")
		vals = append(vals, *find.Name)
	}
	if onlyInherit {
		keys = append(keys, "inherit=?")
		vals = append(vals, true)
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
			val,
			inherit
		FROM properties
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	propmap := make(map[string]*service.Property)
	for rows.Next() {
		p := &service.Property{}
		err := rows.Scan(
			&p.ID,
			&p.EntryID,
			&p.Name,
			&p.Type,
			&p.Value,
			&p.Inherit,
		)
		if err != nil {
			return nil, err
		}
		propmap[p.Name] = p
	}
	return propmap, nil
}

func AddProperty(db *sql.DB, p *service.Property) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addProperty(tx, p)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addProperty(tx *sql.Tx, p *service.Property) error {
	result, err := tx.Exec(`
		INSERT INTO properties (
			entry_id,
			typ,
			name,
			val,
			inherit
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		p.EntryID,
		p.Type,
		p.Name,
		p.Value,
		p.Inherit,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = int(id)
	return nil
}

func UpdateProperty(db *sql.DB, upd service.PropertyUpdater) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateProperty(tx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateProperty(tx *sql.Tx, upd service.PropertyUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Value != nil {
		keys = append(keys, "val=?")
		vals = append(vals, *upd.Value)
	}
	if upd.Inherit != nil {
		keys = append(keys, "inherit=?")
		vals = append(vals, *upd.Inherit)
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
	return nil
}
