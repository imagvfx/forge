package sqlite

import (
	"database/sql"
	"fmt"
	"sort"
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
			group_id INTEGER,
			mode INTEGER NOT NULL,
			FOREIGN KEY (entry_id) REFERENCES entries (id),
			FOREIGN KEY (user_id) REFERENCES users (id),
			UNIQUE (entry_id, user_id, group_id)
		)
	`)
	return err
}

func FindAccessControls(db *sql.DB, user string, find service.AccessControlFinder) ([]*service.AccessControl, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	acm := make(map[string]*service.AccessControl)
	for {
		ent, err := getEntry(tx, user, find.EntryID)
		if err != nil {
			return nil, err
		}
		as, err := findAccessControls(tx, user, find)
		if err != nil {
			return nil, err
		}
		for _, a := range as {
			if acm[a.Accessor] != nil {
				// NOTE: user and group shouldn't share same name.
				continue
			}
			a.EntryPath = ent.Path
			acm[a.Accessor] = a
		}
		if ent.ParentID == nil {
			break
		}
		find.EntryID = *ent.ParentID
	}
	acs := make([]*service.AccessControl, 0)
	for _, a := range acm {
		acs = append(acs, a)
	}
	sort.Slice(acs, func(i, j int) bool {
		return acs[i].Accessor < acs[j].Accessor
	})
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return acs, nil
}

// when id is empty, it will find access controls of root.
func findAccessControls(tx *sql.Tx, user string, find service.AccessControlFinder) ([]*service.AccessControl, error) {
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
			group_id,
			mode
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
			&a.GroupID,
			&a.Mode,
		)
		if err != nil {
			return nil, err
		}
		err = attachAccessorInfo(tx, user, a)
		if err != nil {
			return nil, err
		}
		acss = append(acss, a)
	}
	return acss, nil
}

func userCanRead(tx *sql.Tx, user string, entID int) (bool, error) {
	_, err := userAccessControl(tx, user, entID)
	if err != nil {
		return false, err
	}
	return true, nil
}

// userAccessControl returns the user's access control for an entry.
// It checks the parents recursively as access control inherits.
func userAccessControl(tx *sql.Tx, user string, entID int) (*service.AccessControl, error) {
	u, err := getUserByUser(tx, user)
	if err != nil {
		return nil, err
	}
	adminGroupID := 1
	admins, err := findGroupMembers(tx, service.MemberFinder{GroupID: &adminGroupID})
	for _, admin := range admins {
		if admin.UserID == u.ID {
			// admins can read any entry.
			a, err := getAccessControl(tx, user, adminGroupID)
			if err != nil {
				return nil, err
			}
			return a, nil
		}
	}
	for {
		as, err := findAccessControls(tx, user, service.AccessControlFinder{EntryID: entID})
		if err != nil {
			return nil, err
		}
		// Lower entry has precedence to higher entry.
		// In a same entry, user accessor has precedence to group accessor.
		for _, a := range as {
			if a.UserID == nil {
				continue
			}
			if *a.UserID == u.ID {
				return a, nil
			}
		}
		for _, a := range as {
			if a.GroupID == nil {
				continue
			}
			members, err := findGroupMembers(tx, service.MemberFinder{GroupID: a.GroupID})
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				if m.UserID == u.ID {
					return a, nil
				}
			}
		}
		parentID, err := getEntryParent(tx, entID)
		if err != nil {
			return nil, err
		}
		if parentID == nil {
			break
		}
		entID = *parentID
	}
	return nil, nil
}

func getAccessControl(tx *sql.Tx, user string, id int) (*service.AccessControl, error) {
	rows, err := tx.Query(`
		SELECT
			access_controls.id,
			access_controls.entry_id,
			entries.path,
			access_controls.user_id,
			access_controls.group_id,
			access_controls.mode
		FROM access_controls
		LEFT JOIN entries
		ON access_controls.entry_id = entries.id
		WHERE access_controls.id=?`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("access not found: %v", id)
	}
	a := &service.AccessControl{}
	err = rows.Scan(
		&a.ID,
		&a.EntryID,
		&a.EntryPath,
		&a.UserID,
		&a.GroupID,
		&a.Mode,
	)
	if err != nil {
		return nil, err
	}
	err = attachAccessorInfo(tx, user, a)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func attachAccessorInfo(tx *sql.Tx, user string, a *service.AccessControl) error {
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
		a.Accessor = u.User
		a.AccessorType = 0 // user
	} else {
		g, err := getGroup(tx, *a.GroupID)
		if err != nil {
			return err
		}
		a.Accessor = g.Name
		a.AccessorType = 1 // group
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
			group_id,
			mode
		)
		VALUES (?, ?, ?, ?)
	`,
		a.EntryID,
		a.UserID,
		a.GroupID,
		a.Mode,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = int(id)
	err = attachAccessorInfo(tx, user, a)
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
		Value:    strconv.Itoa(a.Mode),
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
	a, err := getAccessControl(tx, user, upd.ID)
	if err != nil {
		return err
	}
	err = attachAccessorInfo(tx, user, a)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Mode != nil {
		keys = append(keys, "mode=?")
		vals = append(vals, *upd.Mode)
		a.Mode = *upd.Mode
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
		Value:    strconv.Itoa(a.Mode),
	})
	if err != nil {
		return err
	}
	return nil
}
