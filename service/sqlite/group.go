package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createGroupsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			id INTEGER PRIMARY KEY,
			name STRING NOT NULL UNIQUE
		)
	`)
	return err
}

// addAdminGroup adds 'admin' group to groups table.
// Members of admin group are able to see/modify any entry.
// The group isn't allowed to be renamed or deleted.
func addAdminGroup(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO groups
			(name)
		VALUES
			(?)
	`,
		"admin",
	)
	if err != nil {
		return err
	}
	return nil
}

func FindGroups(db *sql.DB, find service.GroupFinder) ([]*service.Group, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	groups, err := findGroups(tx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func findGroups(tx *sql.Tx, find service.GroupFinder) ([]*service.Group, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
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
			name
		FROM groups
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

func GetGroupByName(db *sql.DB, name string) (*service.Group, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := getGroupByName(tx, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func getGroup(tx *sql.Tx, id int) (*service.Group, error) {
	groups, err := findGroups(tx, service.GroupFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, service.NotFoundError{"group not found"}
	}
	return groups[0], nil
}

func getGroupByName(tx *sql.Tx, name string) (*service.Group, error) {
	groups, err := findGroups(tx, service.GroupFinder{Name: &name})
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, service.NotFoundError{"group not found"}
	}
	return groups[0], nil
}

func AddGroup(db *sql.DB, user string, g *service.Group) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addGroup(tx, user, g)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addGroup(tx *sql.Tx, user string, g *service.Group) error {
	// TODO: check the user is a member of admin group.
	result, err := tx.Exec(`
		INSERT INTO groups (
			name
		)
		VALUES (?)
	`,
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

func UpdateGroup(db *sql.DB, user string, upd service.GroupUpdater) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateGroup(tx, user, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateGroup(tx *sql.Tx, user string, upd service.GroupUpdater) error {
	// TODO: check the user is a member of admin group.
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Name != nil {
		keys = append(keys, "name=?")
		vals = append(vals, *upd.Name)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update group: %v", upd.ID)
	}
	vals = append(vals, upd.ID) // for where clause
	result, err := tx.Exec(`
		UPDATE groups
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