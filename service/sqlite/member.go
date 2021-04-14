package sqlite

import (
	"database/sql"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createGroupMembersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS group_members (
			id INTEGER PRIMARY KEY,
			group_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL
		)
	`)
	return err
}

func FindGroupMembers(db *sql.DB, find service.MemberFinder) ([]*service.Member, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	groups, err := findGroupMembers(tx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func findGroupMembers(tx *sql.Tx, find service.MemberFinder) ([]*service.Member, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
	if find.GroupID != nil {
		keys = append(keys, "group_id=?")
		vals = append(vals, *find.GroupID)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.Query(`
		SELECT
			id,
			group_id,
			user_id
		FROM group_members
		`+where+`
		ORDER BY id ASC
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := make([]*service.Member, 0)
	for rows.Next() {
		m := &service.Member{}
		err := rows.Scan(
			&m.ID,
			&m.GroupID,
			&m.UserID,
		)
		if err != nil {
			return nil, err
		}
		err = attachAdditionalGroupMemberInfo(tx, m)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func attachAdditionalGroupMemberInfo(tx *sql.Tx, m *service.Member) error {
	g, err := getGroup(tx, m.GroupID)
	if err != nil {
		return err
	}
	m.Group = g.Name
	u, err := getUser(tx, m.UserID)
	if err != nil {
		return err
	}
	m.User = u.User
	return nil
}

func getGroupMember(tx *sql.Tx, id int) (*service.Member, error) {
	members, err := findGroupMembers(tx, service.MemberFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, service.NotFoundError{"group not found"}
	}
	return members[0], nil
}

func AddGroupMember(db *sql.DB, user string, m *service.Member) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addGroupMember(tx, user, m)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addGroupMember(tx *sql.Tx, user string, m *service.Member) error {
	// TODO: check the user is a member of admin group.
	result, err := tx.Exec(`
		INSERT INTO group_members (
			group_id,
			user_id
		)
		VALUES (?, ?)
	`,
		m.GroupID,
		m.UserID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = int(id)
	return nil
}

func DeleteGroupMember(db *sql.DB, user string, id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteGroupMember(tx, user, id)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteGroupMember(tx *sql.Tx, user string, id int) error {
	// TODO: check the user is a member of admin group.
	_, err := getGroupMember(tx, id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		DELETE FROM group_members
		WHERE id=?
	`,
		id,
	)
	if err != nil {
		return err
	}
	return nil
}
