package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

func createAccessListTable(tx *sql.Tx) error {
	// TODO: add group_id once groups table is created
	// TODO: rename the table to access_list
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS access_controls (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER NOT NULL,
			accessor_id INTEGER,
			mode INTEGER NOT NULL,
			updated_at TIMESTAMP,
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

func EntryAccessList(db *sql.DB, ctx context.Context, path string) ([]*forge.Access, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	accs, err := entryAccessList(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return accs, nil
}

func entryAccessList(tx *sql.Tx, ctx context.Context, path string) ([]*forge.Access, error) {
	acm := make(map[string]*forge.Access)
	for {
		ent, err := getEntry(tx, ctx, path)
		if err != nil {
			return nil, err
		}
		as, err := findAccessList(tx, ctx, forge.AccessFinder{EntryPath: &path})
		if err != nil {
			return nil, err
		}
		for _, a := range as {
			if acm[a.Name] != nil {
				// Already found the accessor permission on a child entry.
				continue
			}
			a.EntryPath = ent.Path
			acm[a.Name] = a
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	acs := make([]*forge.Access, 0)
	for _, a := range acm {
		acs = append(acs, a)
	}
	return acs, nil
}

func GetAccessList(db *sql.DB, ctx context.Context, path string) ([]*forge.Access, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	accs, err := findAccessList(tx, ctx, forge.AccessFinder{EntryPath: &path})
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return accs, nil
}

// when id is empty, it will find access controls of root.
func findAccessList(tx *sql.Tx, ctx context.Context, find forge.AccessFinder) ([]*forge.Access, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	if find.Name != nil {
		keys = append(keys, "accessors.name=?")
		vals = append(vals, *find.Name)
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
			access_controls.mode,
			access_controls.updated_at
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
	acss := make([]*forge.Access, 0)
	for rows.Next() {
		var isGroup bool
		a := &forge.Access{}
		err := rows.Scan(
			&a.ID,
			&a.EntryPath,
			&a.Name,
			&isGroup,
			&a.RawValue,
			&a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		a.Type = "user"
		if isGroup {
			a.Type = "group"
		}
		a.Value = "r"
		if a.RawValue == 1 {
			a.Value = "rw"
		}
		acss = append(acss, a)
	}
	return acss, nil
}

func userEnabled(tx *sql.Tx, ctx context.Context, user string) (bool, error) {
	u, err := getAccessor(tx, ctx, user)
	if err != nil {
		return false, err
	}
	if u.IsGroup {
		return false, fmt.Errorf("accessor is a group: %v", user)
	}
	enabled := !u.Disabled
	return enabled, nil
}

// UserRead simulates the context user read the path.
// It returns nil if the path exists and the user can read the path.
// It returns forge.NotFound error if the path is not exists or
// the context user doesn't have read permission.
func UserRead(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return userRead(tx, ctx, path)
}

func userRead(tx *sql.Tx, ctx context.Context, path string) error {
	user := forge.UserNameFromContext(ctx)
	enabled, err := userEnabled(tx, ctx, user)
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("user disabled: %v", user)
	}
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
		return forge.NotFound("cannot access to entry: %s", path)
	}
	return nil
}

// UserWrite simulates the context user write to the path.
// It returns nil if the path exists and the user can write to the path.
// It returns forge.NotFound error if the path is not exists or
// the context user doesn't have read permission.
// It returns forge.Unauthorized error when the user doesn't have write permission.
func UserWrite(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return userWrite(tx, ctx, path)
}

func userWrite(tx *sql.Tx, ctx context.Context, path string) error {
	user := forge.UserNameFromContext(ctx)
	enabled, err := userEnabled(tx, ctx, user)
	if err != nil {
		return err
	}
	if !enabled {
		return fmt.Errorf("user disabled: %v", user)
	}
	mode, err := userAccessMode(tx, ctx, path)
	if err != nil {
		return err
	}
	if mode == nil {
		// The entry should invisible to the user.
		return forge.NotFound("cannot access to entry: %s", path)
	}
	if *mode == "r" {
		return forge.Unauthorized("entry modification not allowed: %s", path)
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
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return nil, forge.Unauthorized("context user unspecified")
	}
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
		as, err := findAccessList(tx, ctx, forge.AccessFinder{EntryPath: &path})
		if err != nil {
			return nil, err
		}
		// Lower entry has precedence to higher entry.
		// In a same entry, user accessor has precedence to group accessor.
		for _, a := range as {
			if a.Type == "user" && a.Name == user {
				return &a.Value, nil
			}
		}
		// a user can be a member of multiple groups,
		// let's find most permissive
		value := ""
		for _, a := range as {
			if a.Type == "user" {
				continue
			}
			// groups
			yes, err := isGroupMember(tx, ctx, a.Name, user)
			if err != nil {
				return nil, err
			}
			if yes {
				if a.Value == "r" {
					value = a.Value
					continue
				}
				return &a.Value, nil
			}
		}
		if value == "r" {
			return &value, nil
		}
		if path == "/" {
			break
		}
		path = filepath.Dir(path)
	}
	return nil, nil
}

func IsAdmin(db *sql.DB, ctx context.Context, user string) (bool, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	admin, err := isAdmin(tx, ctx, user)
	if err != nil {
		return false, err
	}
	err = tx.Commit()
	if err != nil {
		return false, err
	}
	return admin, nil
}

func isAdmin(tx *sql.Tx, ctx context.Context, user string) (bool, error) {
	if user == "system" {
		// system will be implictly treated as admin.
		return true, nil
	}
	return isGroupMember(tx, ctx, "admin", user)
}

func GetAccess(db *sql.DB, ctx context.Context, path, name string) (*forge.Access, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	acl, err := getAccess(tx, ctx, path, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return acl, nil
}

func getAccess(tx *sql.Tx, ctx context.Context, path, name string) (*forge.Access, error) {
	as, err := findAccessList(tx, ctx, forge.AccessFinder{
		EntryPath: &path,
		Name:      &name,
	})
	if err != nil {
		return nil, err
	}
	if len(as) == 0 {
		return nil, forge.NotFound("access control not found")
	}
	a := as[0]
	return a, nil
}

func AddAccess(db *sql.DB, ctx context.Context, a *forge.Access) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addAccess(tx, ctx, a)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addAccess(tx *sql.Tx, ctx context.Context, a *forge.Access) error {
	err := userWrite(tx, ctx, a.EntryPath)
	if err != nil {
		return err
	}
	ac, err := getAccessor(tx, ctx, a.Name)
	if err != nil {
		return err
	}
	a.RawValue = 0
	if a.Value == "rw" {
		a.RawValue = 1
	}
	entryID, err := getEntryID(tx, ctx, a.EntryPath)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO access_controls (
			entry_id,
			accessor_id,
			mode,
			updated_at
		)
		VALUES (?, ?, ?, ?)
	`,
		entryID,
		ac.ID,
		a.RawValue,
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	a.ID = int(id)
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: a.EntryPath,
		User:      user,
		Action:    "create",
		Category:  "access",
		Name:      a.Name,
		Type:      a.Type,
		Value:     a.Value,
	})
	if err != nil {
		return err
	}
	return nil
}

func UpdateAccess(db *sql.DB, ctx context.Context, upd forge.AccessUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateAccess(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateAccess(tx *sql.Tx, ctx context.Context, upd forge.AccessUpdater) error {
	err := userWrite(tx, ctx, upd.EntryPath)
	if err != nil {
		return err
	}
	a, err := getAccess(tx, ctx, upd.EntryPath, upd.Name)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if upd.Value != nil {
		rawMode := 0
		if *upd.Value == "rw" {
			rawMode = 1
		}
		if rawMode != a.RawValue {
			keys = append(keys, "mode=?")
			vals = append(vals, rawMode)
			a.Value = *upd.Value
		}
	}
	if len(keys) == 0 {
		return nil
	}
	keys = append(keys, "updated_at=?")
	vals = append(vals, time.Now().UTC())
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
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: a.EntryPath,
		User:      user,
		Action:    "update",
		Category:  "access",
		Name:      a.Name,
		Type:      a.Type,
		Value:     a.Value,
	})
	if err != nil {
		return err
	}
	return nil
}

func DeleteAccess(db *sql.DB, ctx context.Context, path, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteAccess(tx, ctx, path, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteAccess(tx *sql.Tx, ctx context.Context, path, name string) error {
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	ent, err := getEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	_, err = getDefaultAccess(tx, ctx, ent.Type, name)
	if err == nil {
		return fmt.Errorf("cannot delete default access of %q: %v", ent.Type, name)
	}
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return err
		}
	}
	a, err := getAccess(tx, ctx, path, name)
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
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: path,
		User:      user,
		Action:    "delete",
		Category:  "access",
		Name:      a.Name,
		Type:      a.Type,
		Value:     "",
	})
	if err != nil {
		return nil
	}
	return nil
}
