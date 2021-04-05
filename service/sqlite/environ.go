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

func FindEnvirons(db *sql.DB, find service.EnvironFinder) ([]*service.Environ, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	envmap := make(map[string]*service.Environ)
	for {
		ent, err := getEntry(tx, find.EntryID)
		if err != nil {
			return nil, err
		}
		emap, err := findEnvirons(tx, find)
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
	envs := make([]*service.Environ, 0, len(envmap))
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
func findEnvirons(tx *sql.Tx, find service.EnvironFinder) (map[string]*service.Environ, error) {
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
	envmap := make(map[string]*service.Environ)
	for rows.Next() {
		e := &service.Environ{}
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

func AddEnviron(db *sql.DB, e *service.Environ) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEnviron(tx, e)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEnviron(tx *sql.Tx, e *service.Environ) error {
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
	return nil
}

func UpdateEnviron(db *sql.DB, upd service.EnvironUpdater) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateEnviron(tx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateEnviron(tx *sql.Tx, upd service.EnvironUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Value != nil {
		keys = append(keys, "val=?")
		vals = append(vals, *upd.Value)
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
	return nil
}
