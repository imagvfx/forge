package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createUsersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			email STRING NOT NULL UNIQUE,
			name STRING NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_users_email ON users (email)`)
	return err
}

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
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
	if find.Email != nil {
		keys = append(keys, "email=?")
		vals = append(vals, *find.Email)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			email,
			name
		FROM users
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
			&u.Email,
			&u.Name,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUserByEmail(db *sql.DB, ctx context.Context, user string) (*service.User, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := getUserByEmail(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func getUser(tx *sql.Tx, ctx context.Context, id int) (*service.User, error) {
	users, err := findUsers(tx, ctx, service.UserFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, service.NotFoundError{"user not found"}
	}
	return users[0], nil
}

func getUserByEmail(tx *sql.Tx, ctx context.Context, user string) (*service.User, error) {
	users, err := findUsers(tx, ctx, service.UserFinder{Email: &user})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, service.NotFoundError{"user not found"}
	}
	return users[0], nil
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
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addUser(tx *sql.Tx, ctx context.Context, u *service.User) error {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO users (
			email,
			name
		)
		VALUES (?, ?)
	`,
		u.Email,
		u.Name,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	u.ID = int(id)
	if u.ID == 1 {
		// make the user admin
		adminGroupID := 1
		ctx = service.ContextWithUserEmail(ctx, "system")
		err = addGroupMember(tx, ctx, &service.Member{
			GroupID: adminGroupID,
			UserID:  u.ID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
