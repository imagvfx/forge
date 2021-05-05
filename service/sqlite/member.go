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
			member_id INTEGER NOT NULL,
			FOREIGN KEY (group_id) REFERENCES accessors (id),
			FOREIGN KEY (member_id) REFERENCES accessors (id),
			UNIQUE (group_id, member_id)
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
		keys = append(keys, "members.name=?")
		vals = append(vals, *find.Member)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			group_members.id,
			groups.name,
			members.name
		FROM group_members
		LEFT JOIN accessors AS groups ON group_members.group_id = groups.id
		LEFT JOIN accessors AS members ON group_members.member_id = members.id
		`+where,
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
			&m.Group,
			&m.Member,
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
		return nil, service.NotFound("member not found")
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
	g, err := getGroupByName(tx, ctx, m.Group)
	if err != nil {
		return err
	}
	u, err := getUserByName(tx, ctx, m.Member)
	if err != nil {
		return err
	}
	// TODO: check the user is a member of admin group.
	result, err := tx.ExecContext(ctx, `
		INSERT INTO group_members (
			group_id,
			member_id
		)
		VALUES (?, ?)
	`,
		g.ID,
		u.ID,
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
	if m.Group == "admin" {
		members, err := findGroupMembers(tx, ctx, service.MemberFinder{Group: &m.Group})
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
