package sqlite

import (
	"database/sql"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createUsersTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			user STRING NOT NULL UNIQUE,
			name STRING NOT NULL
		)
	`)
	return err
}

func addRootUser(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO users
			(id, user, name)
		VALUES
			(?, ?, ?)
	`,
		0, "admin", "",
	)
	if err != nil {
		return err
	}
	return nil
}

func FindUsers(db *sql.DB, find service.UserFinder) ([]*service.User, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	users, err := findUsers(tx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return users, nil
}

func findUsers(tx *sql.Tx, find service.UserFinder) ([]*service.User, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
	if find.User != nil {
		keys = append(keys, "user=?")
		vals = append(vals, *find.User)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.Query(`
		SELECT
			id,
			user,
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
			&u.User,
			&u.Name,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUserByUser(db *sql.DB, user string) (*service.User, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := getUserByUser(tx, user)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return u, nil
}

func getUserByUser(tx *sql.Tx, user string) (*service.User, error) {
	users, err := findUsers(tx, service.UserFinder{User: &user})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, service.NotFoundError{"user not found"}
	}
	return users[0], nil
}

func AddUser(db *sql.DB, u *service.User) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addUser(tx, u)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addUser(tx *sql.Tx, u *service.User) error {
	result, err := tx.Exec(`
		INSERT INTO users (
			user,
			name
		)
		VALUES (?, ?)
	`,
		u.User,
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
	return nil
}
