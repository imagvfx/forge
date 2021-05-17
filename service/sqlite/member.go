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

func isGroupMember(tx *sql.Tx, ctx context.Context, group, member string) (bool, error) {
	mems, err := findGroupMembers(tx, ctx, service.MemberFinder{Group: &group, Member: &member})
	if err != nil {
		return false, err
	}
	if len(mems) == 0 {
		return false, nil
	}
	return true, nil
}

func AddGroupMember(db *sql.DB, ctx context.Context, m *service.Member) error {
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
		return service.Unauthorized("user doesn't have permission to add group member: %v", user)
	}
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
	g, err := getGroup(tx, ctx, m.Group)
	if err != nil {
		return err
	}
	u, err := getUser(tx, ctx, m.Member)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
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
	return nil
}

func DeleteGroupMember(db *sql.DB, ctx context.Context, group, member string) error {
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
		return service.Unauthorized("user doesn't have permission to delete group member: %v", user)
	}
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
	if group == "admin" {
		members, err := findGroupMembers(tx, ctx, service.MemberFinder{Group: &group})
		if err != nil {
			return err
		}
		if len(members) == 1 {
			return fmt.Errorf("need at least 1 admin")
		}
	}
	g, err := getGroup(tx, ctx, group)
	if err != nil {
		return err
	}
	u, err := getUser(tx, ctx, member)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM group_members
		WHERE group_id=? AND member_id=?
	`,
		g.ID,
		u.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return service.NotFound("%q is not a member of group %q", member, group)
	}
	return nil
}
