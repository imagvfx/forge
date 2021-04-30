package sqlite

import (
	"context"
	"database/sql"
	"fmt"
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
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_group_members_group_id ON group_members (group_id)`)
	return err
}

func FindGroupMembers(db *sql.DB, ctx context.Context, find service.MemberFinder) ([]*service.Member, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	groups, err := findGroupMembers(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func findGroupMembers(tx *sql.Tx, ctx context.Context, find service.MemberFinder) ([]*service.Member, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "group_members.id=?")
		vals = append(vals, *find.ID)
	}
	if find.GroupID != nil {
		keys = append(keys, "group_members.group_id=?")
		vals = append(vals, *find.GroupID)
	}
	if find.Group != nil {
		keys = append(keys, "groups.name=?")
		vals = append(vals, *find.Group)
	}
	if find.Member != nil {
		keys = append(keys, "users.email=?")
		vals = append(vals, *find.Member)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			group_members.id,
			group_members.group_id,
			group_members.user_id,
			groups.name,
			users.email
		FROM group_members
		LEFT JOIN groups ON group_members.group_id = groups.id
		LEFT JOIN users ON group_members.user_id = users.id
		`+where+`
		ORDER BY group_members.id ASC
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
			&m.Group,
			&m.User,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func getGroupMember(tx *sql.Tx, ctx context.Context, group, member string) (*service.Member, error) {
	members, err := findGroupMembers(tx, ctx, service.MemberFinder{Group: &group, Member: &member})
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, service.NotFoundError{"member not found"}
	}
	return members[0], nil
}

func AddGroupMember(db *sql.DB, ctx context.Context, m *service.Member) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addGroupMember(tx, ctx, m)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addGroupMember(tx *sql.Tx, ctx context.Context, m *service.Member) error {
	// TODO: check the user is a member of admin group.
	result, err := tx.ExecContext(ctx, `
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

func DeleteGroupMember(db *sql.DB, ctx context.Context, group, member string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteGroupMember(tx, ctx, group, member)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteGroupMember(tx *sql.Tx, ctx context.Context, group, member string) error {
	m, err := getGroupMember(tx, ctx, group, member)
	if err != nil {
		return err
	}
	adminGroupID := 1
	if m.GroupID == adminGroupID {
		members, err := findGroupMembers(tx, ctx, service.MemberFinder{GroupID: &adminGroupID})
		if err != nil {
			return err
		}
		if len(members) == 1 {
			return fmt.Errorf("need at least 1 admin")
		}
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM group_members
		WHERE id=?
	`,
		m.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
