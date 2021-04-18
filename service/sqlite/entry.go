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
			typ STRING NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES entries (id)
		)
	`)
	return err
}

func addRootEntry(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO entries
			(id, path, typ)
		VALUES
			(?, ?, ?)
	`,
		0, "/", "root",
	)
	if err != nil {
		return err
	}
	return nil
}

func FindEntries(db *sql.DB, user string, find service.EntryFinder) ([]*service.Entry, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ents, err := findEntries(tx, user, find)
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
func findEntries(tx *sql.Tx, user string, find service.EntryFinder) ([]*service.Entry, error) {
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
			path,
			typ
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
			&e.Type,
		)
		if err != nil {
			return nil, err
		}
		canRead, err := userCanRead(tx, user, e.ID)
		if err != nil {
			return nil, err
		}
		if canRead {
			ents = append(ents, e)
		}
	}
	return ents, nil
}

// getEntryParent get the entry's parent without checking user permission.
// It shouldn't be used except permission checks.
func getEntryParent(tx *sql.Tx, id int) (*int, error) {
	rows, err := tx.Query(`
		SELECT
			parent_id
		FROM entries
		WHERE id=?
	`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("entry not found: %v", id)
	}
	var parentID *int
	err = rows.Scan(
		&parentID,
	)
	return parentID, nil
}

func GetEntry(db *sql.DB, user string, id int) (*service.Entry, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ent, err := getEntry(tx, user, id)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ent, nil
}

func getEntry(tx *sql.Tx, user string, id int) (*service.Entry, error) {
	ents, err := findEntries(tx, user, service.EntryFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, fmt.Errorf("entry not found")
	}
	return ents[0], nil
}

func UserCanWriteEntry(db *sql.DB, user string, id int) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	ok, err := userCanWrite(tx, user, id)
	if err != nil {
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		return false, err
	}
	return ok, nil
}

func AddEntry(db *sql.DB, user string, e *service.Entry, props []*service.Property, envs []*service.Property) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEntry(tx, user, e)
	if err != nil {
		return err
	}
	for _, p := range props {
		p.EntryID = e.ID
		err := addProperty(tx, user, p)
		if err != nil {
			return err
		}
	}
	for _, env := range envs {
		env.EntryID = e.ID
		err := addEnviron(tx, user, env)
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

func addEntry(tx *sql.Tx, user string, e *service.Entry) error {
	if e.ParentID == nil {
		fmt.Errorf("parent id unspecified")
	}
	ok, err := userCanWrite(tx, user, *e.ParentID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	result, err := tx.Exec(`
		INSERT INTO entries (
			parent_id,
			path,
			typ
		)
		VALUES (?, ?, ?)
	`,
		e.ParentID,
		e.Path,
		e.Type,
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
		EntryID:  e.ID,
		User:     user,
		Action:   "create",
		Category: "entry",
		Name:     e.Path,
		Type:     e.Type,
	})
	if err != nil {
		return err
	}
	return nil
}
