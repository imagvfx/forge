package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createAccessControlsTable(tx *sql.Tx) error {
	// TODO: add group_id once groups table is created
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS access_controls (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER NOT NULL,
			accessor_id INTEGER,
			mode INTEGER NOT NULL,
			FOREIGN KEY (accessor_id) REFERENCES accessors (id),
			UNIQUE (entry_id, accessor_id)
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
		ent, err := getEntry(tx, ctx, path)
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
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	if find.Accessor != nil {
		keys = append(keys, "accessors.name=?")
		vals = append(vals, *find.Accessor)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			access_controls.id,
			entries.path,
			accessors.name,
			accessors.is_group,
			access_controls.mode
		FROM access_controls
		LEFT JOIN entries ON access_controls.entry_id = entries.id
		LEFT JOIN accessors ON access_controls.accessor_id = accessors.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	acss := make([]*service.AccessControl, 0)
	for rows.Next() {
		var isGroup bool
		var mode int
		a := &service.AccessControl{}
		err := rows.Scan(
			&a.ID,
			&a.EntryPath,
			&a.Accessor,
			&isGroup,
			&mode,
		)
		if err != nil {
			return nil, err
		}
		a.AccessorType = "user"
		if isGroup {
			a.AccessorType = "group"
		}
		a.Mode = "r"
		if mode == 1 {
			a.Mode = "rw"
		}
		acss = append(acss, a)
	}
	return acss, nil
}

// userRead returns an error if the user cannot read the entry.
// It returns service.NotFound error when the context user doesn't have read permission.
func userRead(tx *sql.Tx, ctx context.Context, path string) error {
	if path == "/" {
		// Everyone should be able to access root.
		return nil
	}
	mode, err := userAccessMode(tx, ctx, path)
	if err != nil {
		return err
	}
	if mode == nil {
		// The entry should invisible to the user.
		return service.NotFound("cannot access to entry")
	}
	return nil
}

// userWrite returns an error if the user cannot write the entry.
// It returns service.NotFound error when the context user doesn't have read permission or
// returns service.Unauthorized error when the context user doesn't have write permission.
func userWrite(tx *sql.Tx, ctx context.Context, path string) error {
	mode, err := userAccessMode(tx, ctx, path)
	if err != nil {
		return err
	}
	if mode == nil {
		// The entry should invisible to the user.
		return service.NotFound("cannot access to entry")
	}
	if *mode == "r" {
		return service.Unauthorized("entry modification not allowed")
	}
	return nil
}

// userAccessMode returns the user's access control for an entry.
// It checks the parents recursively as access control inherits.
// It returns (nil, nil) when there is no access_control exists for the user.
func userAccessMode(tx *sql.Tx, ctx context.Context, path string) (*string, error) {
	if path == "" {
		return nil, fmt.Errorf("path should be specified for access check")
	}
	user := service.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	if yes {
		// admins can read any entry.
		rwMode := "rw"
		return &rwMode, nil
	}
	for {
		as, err := findAccessControls(tx, ctx, service.AccessControlFinder{EntryPath: &path})
		if err != nil {
			return nil, err
		}
		// Lower entry has precedence to higher entry.
		// In a same entry, user accessor has precedence to group accessor.
		for _, a := range as {
			if a.AccessorType == "user" && a.Accessor == user {
				return &a.Mode, nil
			}
		}
		for _, a := range as {
			if a.AccessorType == "user" {
				continue
			}
			// groups
			yes, err := isGroupMember(tx, ctx, a.Accessor, user)
			if err != nil {
				return nil, err
			}
			if yes {
				return &a.Mode, nil
			}
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	return nil, nil
}

func isAdmin(tx *sql.Tx, ctx context.Context, user string) (bool, error) {
	if user == "system" {
		// system will be implictly treated as admin.
		return true, nil
	}
	return isGroupMember(tx, ctx, "admin", user)
}

func GetAccessControl(db *sql.DB, ctx context.Context, path, name string) (*service.AccessControl, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	acl, err := getAccessControl(tx, ctx, path, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return acl, nil
}

func getAccessControl(tx *sql.Tx, ctx context.Context, path, name string) (*service.AccessControl, error) {
	as, err := findAccessControls(tx, ctx, service.AccessControlFinder{
		EntryPath: &path,
		Accessor:  &name,
	})
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, service.NotFound("access control not found")
	}
	a := as[0]
	return a, nil
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
	err := userWrite(tx, ctx, a.EntryPath)
	if err != nil {
		return err
	}
	ac, err := getAccessor(tx, ctx, a.Accessor)
	if err != nil {
		return err
	}
	acType := "user"
	if ac.IsGroup {
		acType = "group"
	}
	if a.AccessorType != acType {
		return fmt.Errorf("mismatch accessor type: got %v, want %v", a.AccessorType, acType)
	}
	mode := 0
	if a.Mode == "rw" {
		mode = 1
	}
	entryID, err := getEntryID(tx, ctx, a.EntryPath)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO access_controls (
			entry_id,
			accessor_id,
			mode
		)
		VALUES (?, ?, ?)
	`,
		entryID,
		ac.ID,
		mode,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = int(id)
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryPath: a.EntryPath,
		User:      user,
		Action:    "create",
		Category:  "access",
		Name:      a.Accessor,
		Type:      a.AccessorType,
		Value:     a.Mode,
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
	err := userWrite(tx, ctx, upd.EntryPath)
	if err != nil {
		return err
	}
	a, err := getAccessControl(tx, ctx, upd.EntryPath, upd.Accessor)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Mode != nil {
		mode := 0
		if *upd.Mode == "rw" {
			mode = 1
		}
		keys = append(keys, "mode=?")
		vals = append(vals, mode)
		a.Mode = *upd.Mode
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update property %v:%v", upd.EntryPath, upd.Accessor)
	}
	vals = append(vals, a.ID) // for where clause
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
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryPath: a.EntryPath,
		User:      user,
		Action:    "update",
		Category:  "access",
		Name:      a.Accessor,
		Type:      a.AccessorType,
		Value:     a.Mode,
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
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	a, err := getAccessControl(tx, ctx, path, name)
	if err != nil {
		return err
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
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryPath: path,
		User:      user,
		Action:    "delete",
		Category:  "access",
		Name:      a.Accessor,
		Type:      a.AccessorType,
		Value:     "",
	})
	if err != nil {
		return nil
	}
	return nil
}
