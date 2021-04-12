package sqlite

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createAccessControlsTable(tx *sql.Tx) error {
	// TODO: add group_id once groups table is created
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS access_controls (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER NOT NULL,
			user_id INTEGER,
			typ INTEGER NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id),
			FOREIGN KEY (user_id) REFERENCES users (id),
			UNIQUE (entry_id, user_id)
		)
	`)
	return err
}

func FindAccessControls(db *sql.DB, find service.AccessControlFinder) ([]*service.AccessControl, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	acss := make([]*service.AccessControl, 0)
	for {
		ent, err := getEntry(tx, find.EntryID)
		if err != nil {
			return nil, err
		}
		as, err := findAccessControls(tx, find)
		if err != nil {
			return nil, err
		}
		acss = append(acss, as...)
		if ent.ParentID == nil {
			break
		}
		find.EntryID = *ent.ParentID
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return acss, nil
}

// when id is empty, it will find access controls of root.
func findAccessControls(tx *sql.Tx, find service.AccessControlFinder) ([]*service.AccessControl, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "entry_id=?")
	vals = append(vals, find.EntryID)
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.Query(`
		SELECT
			id,
			entry_id,
			user_id,
			typ
		FROM access_controls
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	acss := make([]*service.AccessControl, 0)
	for rows.Next() {
		a := &service.AccessControl{}
		err := rows.Scan(
			&a.ID,
			&a.EntryID,
			&a.UserID,
			&a.Type,
		)
		if err != nil {
			return nil, err
		}
		err = attachAccessorInfo(tx, a)
		if err != nil {
			return nil, err
		}
		acss = append(acss, a)
	}
	return acss, nil
}

func getAccessControl(tx *sql.Tx, id int) (*service.AccessControl, error) {
	rows, err := tx.Query(`
		SELECT
			id,
			entry_id,
			user_id,
			typ
		FROM access_controls
		WHERE id=?`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("access not found")
	}
	a := &service.AccessControl{}
	err = rows.Scan(
		&a.ID,
		&a.EntryID,
		&a.UserID,
		&a.Type,
	)
	if err != nil {
		return nil, err
	}
	err = attachAccessorInfo(tx, a)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func attachAccessorInfo(tx *sql.Tx, a *service.AccessControl) error {
	if a.UserID != nil && a.GroupID != nil {
		return fmt.Errorf("both user_id and group_id is defined")
	}
	if a.UserID == nil && a.GroupID == nil {
		return fmt.Errorf("both user_id and group_id is nil")
	}
	if a.UserID != nil {
		u, err := getUser(tx, *a.UserID)
		if err != nil {
			return err
		}
		a.Accessor = u.Name
		a.AccessorType = 0 // user
		a.Members = []*service.User{u}
	} else {
		// TODO: process for group access
	}
	return nil
}

func AddAccessControl(db *sql.DB, user string, a *service.AccessControl) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addAccessControl(tx, user, a)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addAccessControl(tx *sql.Tx, user string, a *service.AccessControl) error {
	result, err := tx.Exec(`
		INSERT INTO access_controls (
			entry_id,
			user_id,
			typ
		)
		VALUES (?, ?, ?)
	`,
		a.EntryID,
		a.UserID,
		a.Type,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = int(id)
	err = attachAccessorInfo(tx, a)
	if err != nil {
		return err
	}
	err = addLog(tx, &service.Log{
		EntryID:  a.EntryID,
		User:     user,
		Action:   "create",
		Category: "access",
		Name:     a.Accessor,
		Type:     strconv.Itoa(a.AccessorType),
		Value:    strconv.Itoa(a.Type),
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateAccessControl(db *sql.DB, user string, upd service.AccessControlUpdater) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateAccessControl(tx, user, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateAccessControl(tx *sql.Tx, user string, upd service.AccessControlUpdater) error {
	a, err := getAccessControl(tx, upd.ID)
	if err != nil {
		return err
	}
	err = attachAccessorInfo(tx, a)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Type != nil {
		keys = append(keys, "typ=?")
		vals = append(vals, *upd.Type)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update property %v", upd.ID)
	}
	vals = append(vals, upd.ID) // for where clause
	result, err := tx.Exec(`
		UPDATE access_controls
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
		EntryID:  a.EntryID,
		User:     user,
		Action:   "update",
		Category: "access",
		Name:     a.Accessor,
		Type:     strconv.Itoa(a.AccessorType),
		Value:    strconv.Itoa(a.Type),
	})
	if err != nil {
		return err
	}
	return nil
}
