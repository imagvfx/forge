package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge"
)

// see createAccessorTable for table creation.

// addEveryoneGroup adds 'everyone' group to accessors table.
// everyone group includes every user as the name says.
// One unique thing for everyone group is that it won't hold members.
// Membership checking for any user to everyone will return true.
// The group isn't allowed to be renamed or deleted.
func addEveryoneGroup(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO accessors
			(is_group, name, called, disabled)
		VALUES
			(?, ?, ?, ?)
	`,
		true, "everyone", "Everyone", false,
	)
	if err != nil {
		return err
	}
	return nil
}

// addAdminGroup adds 'admin' group to accessors table.
// admin group is created while initializing db, so it's the first created accessor.
// Members of admin group are able to see/modify any entry.
// The group isn't allowed to be renamed or deleted.
func addAdminGroup(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO accessors
			(is_group, name, called, disabled)
		VALUES
			(?, ?, ?, ?)
	`,
		true, "admin", "Admin", false,
	)
	if err != nil {
		return err
	}
	return nil
}

func FindGroups(db *sql.DB, ctx context.Context, find forge.GroupFinder) ([]*forge.Group, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	groups, err := findGroups(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func findGroups(tx *sql.Tx, ctx context.Context, find forge.GroupFinder) ([]*forge.Group, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	keys = append(keys, "is_group=?")
	vals = append(vals, true)
	if find.Name != nil {
		keys = append(keys, "name=?")
		vals = append(vals, *find.Name)
	}
	if find.Called != nil {
		keys = append(keys, "called=?")
		vals = append(vals, *find.Called)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name,
			called
		FROM accessors
		`+where+`
		ORDER BY id ASC
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := make([]*forge.Group, 0)
	for rows.Next() {
		u := &forge.Group{}
		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Called,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, u)
	}
	return groups, nil
}

func GetGroup(db *sql.DB, ctx context.Context, name string) (*forge.Group, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := getGroup(tx, ctx, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func getGroup(tx *sql.Tx, ctx context.Context, name string) (*forge.Group, error) {
	groups, err := findGroups(tx, ctx, forge.GroupFinder{Name: &name})
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, forge.NotFound("group not found: %v", name)
	}
	return groups[0], nil
}

func AddGroup(db *sql.DB, ctx context.Context, g *forge.Group) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return forge.Unauthorized("user doesn't have permission to add group: %v", user)
	}
	if strings.Split(g.Name, "@")[0] == "everyone" {
		return fmt.Errorf("'everyone[@host]' group will be created automatically and cannot be created by a user")
	}
	err = addGroup(tx, ctx, g)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addGroup(tx *sql.Tx, ctx context.Context, g *forge.Group) error {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO accessors (
			is_group,
			name,
			called,
			disabled
		)
		VALUES (?, ?, ?, ?)
	`,
		true,
		g.Name,
		g.Called,
		false,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	g.ID = int(id)
	return nil
}

func UpdateGroup(db *sql.DB, ctx context.Context, upd forge.GroupUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return forge.Unauthorized("user doesn't have permission to update group: %v", user)
	}
	err = updateGroup(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateGroup(tx *sql.Tx, ctx context.Context, upd forge.GroupUpdater) error {
	g, err := getGroup(tx, ctx, upd.Name)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if upd.NewName != nil {
		if upd.Name == "admin" || strings.Split(upd.Name, "@")[0] == "everyone" {
			return fmt.Errorf("rename 'admin' or 'everyone[@host]' group is not supported: %v", upd.Name)
		}
		if g.Name != *upd.NewName {
			keys = append(keys, "name=?")
			vals = append(vals, *upd.NewName)
		}
	}
	if upd.Called != nil {
		if g.Called != *upd.Called {
			keys = append(keys, "called=?")
			vals = append(vals, *upd.Called)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	vals = append(vals, upd.Name) // for where clause
	result, err := tx.ExecContext(ctx, `
		UPDATE accessors
		SET `+strings.Join(keys, ", ")+`
		WHERE is_group=1 AND name=?
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
