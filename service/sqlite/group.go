package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
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
			(is_group, name)
		VALUES
			(?, ?)
	`,
		true, "everyone",
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
			(is_group, name)
		VALUES
			(?, ?)
	`,
		true, "admin",
	)
	if err != nil {
		return err
	}
	return nil
}

func FindGroups(db *sql.DB, ctx context.Context, find service.GroupFinder) ([]*service.Group, error) {
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

func findGroups(tx *sql.Tx, ctx context.Context, find service.GroupFinder) ([]*service.Group, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "is_group=?")
	vals = append(vals, true)
	if find.Name != nil {
		keys = append(keys, "name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name
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
	groups := make([]*service.Group, 0)
	for rows.Next() {
		u := &service.Group{}
		err := rows.Scan(
			&u.ID,
			&u.Name,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, u)
	}
	return groups, nil
}

func GetGroup(db *sql.DB, ctx context.Context, name string) (*service.Group, error) {
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

func getGroup(tx *sql.Tx, ctx context.Context, name string) (*service.Group, error) {
	groups, err := findGroups(tx, ctx, service.GroupFinder{Name: &name})
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, service.NotFound("group not found")
	}
	return groups[0], nil
}

func AddGroup(db *sql.DB, ctx context.Context, g *service.Group) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := service.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return service.Unauthorized("user doesn't have permission to add group: %v", user)
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

func addGroup(tx *sql.Tx, ctx context.Context, g *service.Group) error {
	if strings.Split(g.Name, "@")[0] == "everyone" {
		return fmt.Errorf("'everyone[@host]' group will be created automatically and cannot be created by a user")
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO accessors (
			is_group,
			name
		)
		VALUES (?, ?)
	`,
		true,
		g.Name,
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

func UpdateGroup(db *sql.DB, ctx context.Context, upd service.GroupUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := service.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return service.Unauthorized("user doesn't have permission to update group: %v", user)
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

func updateGroup(tx *sql.Tx, ctx context.Context, upd service.GroupUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.NewName != nil {
		if upd.Name == "admin" || strings.Split(upd.Name, "@")[0] == "everyone" {
			return fmt.Errorf("rename 'admin' or 'everyone[@host]' group is not supported: %v", upd.Name)
		}
		keys = append(keys, "name=?")
		vals = append(vals, upd.NewName)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update group: %v", upd.Name)
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
