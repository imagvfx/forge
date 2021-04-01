package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			path STRING NOT NULL UNIQUE,
			FOREIGN KEY (parent_id) REFERENCES entries (id)
		)
	`)
	return err
}

func addRootEntry(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO entries
			(id, path)
		VALUES
			(?, ?)
	`,
		0, "/",
	)
	if err != nil {
		return err
	}
	return nil
}

func FindEntries(db *sql.DB, find service.EntryFinder) ([]*service.Entry, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ents, err := findEntries(tx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ents, nil
}

// when id is empty, it will find entries of root.
func findEntries(tx *sql.Tx, find service.EntryFinder) ([]*service.Entry, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
	if find.Path != "" {
		keys = append(keys, "path=?")
		vals = append(vals, find.Path)
	}
	if find.ParentID != nil {
		keys = append(keys, "parent_id=?")
		vals = append(vals, find.ParentID)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.Query(`
		SELECT
			id,
			parent_id,
			path
		FROM entries
		`+where+`
		ORDER BY id ASC
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ents := make([]*service.Entry, 0)
	for rows.Next() {
		e := &service.Entry{}
		err := rows.Scan(
			&e.ID,
			&e.ParentID,
			&e.Path,
		)
		if err != nil {
			return nil, err
		}
		ents = append(ents, e)
	}
	return ents, nil
}

func GetEntry(db *sql.DB, id int) (*service.Entry, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ent, err := getEntry(tx, id)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ent, nil
}

func getEntry(tx *sql.Tx, id int) (*service.Entry, error) {
	ents, err := findEntries(tx, service.EntryFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	return ents[0], nil
}

func AddEntry(db *sql.DB, e *service.Entry) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEntry(tx, e)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEntry(tx *sql.Tx, e *service.Entry) error {
	result, err := tx.Exec(`
		INSERT INTO entries (
			parent_id,
			path
		)
		VALUES (?, ?)
	`,
		e.ParentID,
		e.Path,
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
