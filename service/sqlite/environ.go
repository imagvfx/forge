package sqlite

import (
	"database/sql"
	"fmt"
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
	return err
}

func FindEnvirons(db *sql.DB, user string, find service.PropertyFinder) ([]*service.Property, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	envmap := make(map[string]*service.Property)
	for {
		ent, err := getEntry(tx, user, find.EntryID)
		if err != nil {
			return nil, err
		}
		emap, err := findEnvirons(tx, user, find)
		if err != nil {
			return nil, err
		}
		for name, e := range emap {
			if envmap[name] == nil {
				e.EntryPath = ent.Path
				envmap[name] = e
			}
		}
		if ent.ParentID == nil {
			break
		}
		find.EntryID = *ent.ParentID
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
func findEnvirons(tx *sql.Tx, user string, find service.PropertyFinder) (map[string]*service.Property, error) {
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
		FROM environs
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
		)
		if err != nil {
			return nil, err
		}
		envmap[e.Name] = e
	}
	return envmap, nil
}

func getEnviron(tx *sql.Tx, user string, id int) (*service.Property, error) {
	rows, err := tx.Query(`
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

func AddEnviron(db *sql.DB, user string, e *service.Property) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEnviron(tx, user, e)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEnviron(tx *sql.Tx, user string, e *service.Property) error {
	result, err := tx.Exec(`
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
	err = addLog(tx, &service.Log{
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

func UpdateEnviron(db *sql.DB, user string, upd service.PropertyUpdater) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateEnviron(tx, user, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateEnviron(tx *sql.Tx, user string, upd service.PropertyUpdater) error {
	e, err := getEnviron(tx, user, upd.ID)
	if err != nil {
		return err
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
	result, err := tx.Exec(`
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
	err = addLog(tx, &service.Log{
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
