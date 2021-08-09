package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/imagvfx/forge/service"
)

// see createAccessorTable for table creation.

func FindUsers(db *sql.DB, ctx context.Context, find service.UserFinder) ([]*service.User, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	users, err := findUsers(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func findUsers(tx *sql.Tx, ctx context.Context, find service.UserFinder) ([]*service.User, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "is_group=?")
	vals = append(vals, false)
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
	users := make([]*service.User, 0)
	for rows.Next() {
		u := &service.User{}
		err := rows.Scan(
			&u.ID,
			&u.Name,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUser(db *sql.DB, ctx context.Context, user string) (*service.User, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := getUser(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func getUser(tx *sql.Tx, ctx context.Context, user string) (*service.User, error) {
	users, err := findUsers(tx, ctx, service.UserFinder{Name: &user})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, service.NotFound("user not found")
	}
	return users[0], nil
}

func getUserID(tx *sql.Tx, ctx context.Context, user string) (int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id FROM accessors
		WHERE is_group=? AND name=?
	`,
		false, user,
	)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	if !rows.Next() {
		return -1, service.NotFound("user not found: %v", user)
	}
	var id int
	err = rows.Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func AddUser(db *sql.DB, ctx context.Context, u *service.User) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addUser(tx, ctx, u)
	if err != nil {
		return err
	}
	err = addDefaultUserSetting(tx, ctx, u.Name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addUser(tx *sql.Tx, ctx context.Context, u *service.User) error {
	users, err := findUsers(tx, ctx, service.UserFinder{})
	if err != nil {
		return err
	}
	firstUser := false
	if len(users) == 0 {
		firstUser = true
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO accessors (
			is_group,
			name
		)
		VALUES (?, ?)
	`,
		false,
		u.Name,
	)
	if err != nil {
		return err
	}
	if firstUser {
		// first user created, make the user admin
		ctx = service.ContextWithUserName(ctx, "system")
		err = addGroupMember(tx, ctx, &service.Member{
			Group:  "admin",
			Member: u.Name,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
