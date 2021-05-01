package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
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
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_access_controls_entry_id ON access_controls (entry_id)`)
	return err
}

func EntryAccessControls(db *sql.DB, ctx context.Context, path string) ([]*service.AccessControl, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	acm := make(map[string]*service.AccessControl)
	for {
		ent, err := getEntryByPath(tx, ctx, path)
		if err != nil {
			return nil, err
		}
		as, err := findAccessControls(tx, ctx, service.AccessControlFinder{EntryPath: &path})
		if err != nil {
			return nil, err
		}
		for _, a := range as {
			if acm[a.Accessor] != nil {
				// Already found the accessor permission on a child entry.
				continue
			}
			a.EntryPath = ent.Path
			acm[a.Accessor] = a
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
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
func findAccessControls(tx *sql.Tx, ctx context.Context, find service.AccessControlFinder) ([]*service.AccessControl, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "access_controls.id=?")
		vals = append(vals, *find.ID)
	}
	if find.EntryID != nil {
		keys = append(keys, "access_controls.entry_id=?")
		vals = append(vals, *find.EntryID)
	}
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	if find.User != nil {
		keys = append(keys, "users.email=?")
		vals = append(vals, *find.User)
	}
	if find.Group != nil {
		keys = append(keys, "groups.name=?")
		vals = append(vals, *find.Group)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			access_controls.id,
			access_controls.entry_id,
			entries.path,
			access_controls.user_id,
			access_controls.group_id,
			access_controls.mode,
			users.email,
			groups.name
		FROM access_controls
		LEFT JOIN entries ON access_controls.entry_id = entries.id
		LEFT JOIN users ON access_controls.user_id = users.id
		LEFT JOIN groups ON access_controls.group_id = groups.id
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
		var user *string
		var group *string
		err := rows.Scan(
			&a.ID,
			&a.EntryID,
			&a.EntryPath,
			&a.UserID,
			&a.GroupID,
			&a.Mode,
			&user,
			&group,
		)
		if err != nil {
			return nil, err
		}
		if user != nil {
			a.Accessor = *user
			a.AccessorType = 0
		} else {
			a.Accessor = *group
			a.AccessorType = 1
		}
		acss = append(acss, a)
	}
	return acss, nil
}

func userCanRead(tx *sql.Tx, ctx context.Context, entID int) (bool, error) {
	mode, err := userAccessMode(tx, ctx, entID)
	if err != nil {
		return false, err
	}
	if mode == nil {
		return false, nil
	}
	return true, nil
}

func userCanWrite(tx *sql.Tx, ctx context.Context, entID int) (bool, error) {
	mode, err := userAccessMode(tx, ctx, entID)
	if err != nil {
		return false, err
	}
	if mode == nil {
		return false, nil
	}
	if *mode == 0 {
		// read mode
		return false, nil
	}
	return true, nil
}

// userAccessMode returns the user's access control for an entry.
// It checks the parents recursively as access control inherits.
func userAccessMode(tx *sql.Tx, ctx context.Context, entID int) (*int, error) {
	user := service.UserEmailFromContext(ctx)
	u, err := getUserByEmail(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	adminGroupID := 1
	admins, err := findGroupMembers(tx, ctx, service.MemberFinder{GroupID: &adminGroupID})
	for _, admin := range admins {
		if admin.UserID == u.ID {
			// admins can read any entry.
			rwMode := 1
			return &rwMode, nil
		}
	}
	for {
		as, err := findAccessControls(tx, ctx, service.AccessControlFinder{EntryID: &entID})
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
				return &a.Mode, nil
			}
		}
		for _, a := range as {
			if a.GroupID == nil {
				continue
			}
			members, err := findGroupMembers(tx, ctx, service.MemberFinder{GroupID: a.GroupID})
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				if m.UserID == u.ID {
					return &a.Mode, nil
				}
			}
		}
		parentID, err := getEntryParent(tx, ctx, entID)
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

// getAccessControl finds and returns AccessControl by it's id.
// The reason it doesn't use findAccessConrtol is the function is entry based.
// I might refactor it.
func getAccessControl(tx *sql.Tx, ctx context.Context, id int) (*service.AccessControl, error) {
	as, err := findAccessControls(tx, ctx, service.AccessControlFinder{
		ID: &id,
	})
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, fmt.Errorf("access control not found")
	}
	a := as[0]
	return a, nil
}

func getAccessControlByPathName(tx *sql.Tx, ctx context.Context, path, name string) (*service.AccessControl, error) {
	var user *string
	var group *string
	if strings.Contains(name, "@") {
		user = &name
	} else {
		group = &name
	}
	as, err := findAccessControls(tx, ctx, service.AccessControlFinder{
		EntryPath: &path,
		User:      user,
		Group:     group,
	})
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, fmt.Errorf("access control not found")
	}
	a := as[0]
	return a, nil
}

func attachAccessorInfo(tx *sql.Tx, ctx context.Context, a *service.AccessControl) error {
	if a.UserID != nil && a.GroupID != nil {
		return fmt.Errorf("both user_id and group_id is defined")
	}
	if a.UserID == nil && a.GroupID == nil {
		return fmt.Errorf("both user_id and group_id is nil")
	}
	if a.UserID != nil {
		u, err := getUser(tx, ctx, *a.UserID)
		if err != nil {
			return err
		}
		a.Accessor = u.Email
		a.AccessorType = 0 // user
	} else {
		g, err := getGroup(tx, ctx, *a.GroupID)
		if err != nil {
			return err
		}
		a.Accessor = g.Name
		a.AccessorType = 1 // group
	}
	return nil
}

func AddAccessControl(db *sql.DB, ctx context.Context, a *service.AccessControl) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addAccessControl(tx, ctx, a)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addAccessControl(tx *sql.Tx, ctx context.Context, a *service.AccessControl) error {
	ok, err := userCanWrite(tx, ctx, a.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	result, err := tx.ExecContext(ctx, `
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
	err = attachAccessorInfo(tx, ctx, a)
	if err != nil {
		return err
	}
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
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

func UpdateAccessControl(db *sql.DB, ctx context.Context, upd service.AccessControlUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateAccessControl(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateAccessControl(tx *sql.Tx, ctx context.Context, upd service.AccessControlUpdater) error {
	a, err := getAccessControl(tx, ctx, upd.ID)
	if err != nil {
		return err
	}
	ok, err := userCanWrite(tx, ctx, a.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
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
	result, err := tx.ExecContext(ctx, `
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
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
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

func DeleteAccessControl(db *sql.DB, ctx context.Context, path, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteAccessControl(tx, ctx, path, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteAccessControl(tx *sql.Tx, ctx context.Context, path, name string) error {
	a, err := getAccessControlByPathName(tx, ctx, path, name)
	if err != nil {
		return err
	}
	ok, err := userCanWrite(tx, ctx, a.EntryID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("user cannot modify entry")
	}
	result, err := tx.Exec(`
		DELETE FROM access_controls
		WHERE id=?
	`,
		a.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 access_control affected, got %v", n)
	}
	user := service.UserEmailFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryID:  a.EntryID,
		User:     user,
		Action:   "delete",
		Category: "access",
		Name:     a.Accessor,
		Type:     strconv.Itoa(a.AccessorType),
		Value:    "",
	})
	if err != nil {
		return nil
	}
	return nil
}
