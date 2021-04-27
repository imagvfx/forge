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
	rows, err := tx.QueryContext(ctx, `
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
		err = attachAdditionalGroupMemberInfo(tx, ctx, m)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

func attachAdditionalGroupMemberInfo(tx *sql.Tx, ctx context.Context, m *service.Member) error {
	g, err := getGroup(tx, ctx, m.GroupID)
	if err != nil {
		return err
	}
	m.Group = g.Name
	u, err := getUser(tx, ctx, m.UserID)
	if err != nil {
		return err
	}
	m.User = u.Email
	return nil
}

func getGroupMember(tx *sql.Tx, ctx context.Context, id int) (*service.Member, error) {
	members, err := findGroupMembers(tx, ctx, service.MemberFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, service.NotFoundError{"group not found"}
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

func DeleteGroupMember(db *sql.DB, ctx context.Context, id int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteGroupMember(tx, ctx, id)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteGroupMember(tx *sql.Tx, ctx context.Context, id int) error {
	m, err := getGroupMember(tx, ctx, id)
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
		id,
	)
	if err != nil {
		return err
	}
	return nil
}
