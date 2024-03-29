package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/imagvfx/forge"
)

// see createAccessorTable for table creation.

func FindUsers(db *sql.DB, ctx context.Context, find forge.UserFinder) ([]*forge.User, error) {
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

func findUsers(tx *sql.Tx, ctx context.Context, find forge.UserFinder) ([]*forge.User, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	keys = append(keys, "is_group=?")
	vals = append(vals, false)
	if find.ID != nil {
		keys = append(keys, "id=?")
		vals = append(vals, *find.ID)
	}
	if find.Name != nil {
		keys = append(keys, "name=?")
		vals = append(vals, *find.Name)
	}
	if find.Disabled != nil {
		keys = append(keys, "disabled=?")
		vals = append(vals, *find.Disabled)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			name,
			called,
			disabled
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
	users := make([]*forge.User, 0)
	for rows.Next() {
		u := &forge.User{}
		err := rows.Scan(
			&u.ID,
			&u.Name,
			&u.Called,
			&u.Disabled,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func GetUser(db *sql.DB, ctx context.Context, user string) (*forge.User, error) {
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

func getUser(tx *sql.Tx, ctx context.Context, user string) (*forge.User, error) {
	users, err := findUsers(tx, ctx, forge.UserFinder{Name: &user})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, forge.NotFound("user not found")
	}
	return users[0], nil
}

func getUserByID(tx *sql.Tx, ctx context.Context, id int) (*forge.User, error) {
	users, err := findUsers(tx, ctx, forge.UserFinder{ID: &id})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, forge.NotFound("user not found")
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
		return -1, forge.NotFound("user not found: %v", user)
	}
	var id int
	err = rows.Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func AddUser(db *sql.DB, ctx context.Context, u *forge.User) error {
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

// splitUserName splits a username into a name and a domain parts.
func splitUserName(username string) (string, string, error) {
	if strings.Contains(username, " ") {
		return "", "", fmt.Errorf("username should not contains spaces: %v", username)
	}
	toks := strings.Split(username, "@")
	if len(toks) != 2 {
		return "", "", fmt.Errorf("username should be '{user}@{domain}' form: %v", username)
	}
	name := toks[0]
	if name == "everyone" {
		return "", "", fmt.Errorf("'everyone@{domain}' is reserved for groups: %v", username)
	}
	domain := toks[1]
	return name, domain, nil
}

func addUser(tx *sql.Tx, ctx context.Context, u *forge.User) error {
	users, err := findUsers(tx, ctx, forge.UserFinder{})
	if err != nil {
		return err
	}
	firstUser := false
	if len(users) == 0 {
		firstUser = true
	}
	_, domain, err := splitUserName(u.Name)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO accessors (
			is_group,
			name,
			called,
			disabled
		)
		VALUES (?, ?, ?, ?)
	`,
		false,
		u.Name,
		u.Called,
		false,
	)
	if err != nil {
		return err
	}
	if firstUser {
		// first user created, make the user admin
		ctx = forge.ContextWithUserName(ctx, "system")
		err = addGroupMember(tx, ctx, &forge.Member{
			Group:  "admin",
			Member: u.Name,
		})
		if err != nil {
			return err
		}
	}
	// add everyone group if the user is first one who is signed with this domain.
	everyone := "everyone@" + domain
	_, err = getGroup(tx, ctx, everyone)
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return err
		}
		err := addGroup(tx, ctx, &forge.Group{Name: everyone})
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateUser(db *sql.DB, ctx context.Context, upd forge.UserUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateUser(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateUser(tx *sql.Tx, ctx context.Context, upd forge.UserUpdater) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != upd.Name {
		admin, err := isAdmin(tx, ctx, ctxUser)
		if err != nil {
			return err
		}
		if !admin {
			return fmt.Errorf("non-admin user cannot modify other user: %v", ctxUser)
		}
	}
	u, err := getUser(tx, ctx, upd.Name)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if upd.Called != nil {
		called := *upd.Called
		called = strings.TrimSpace(called)
		if strings.Contains(called, "\n") {
			return fmt.Errorf("called shouldn't contain new line characters: %v", upd.Called)
		}
		if called != u.Called {
			keys = append(keys, "called=?")
			vals = append(vals, called)
		}
	}
	if upd.Disabled != nil {
		admin, err := isAdmin(tx, ctx, upd.Name)
		if err != nil {
			return err
		}
		if admin {
			return fmt.Errorf("admin user cannot be disabled: %v", upd.Name) // or enabled, too
		}
		disabled := *upd.Disabled
		if disabled != u.Disabled {
			keys = append(keys, "disabled=?")
			vals = append(vals, disabled)
		}
	}
	if len(keys) == 0 {
		return nil
	}
	vals = append(vals, upd.Name)
	result, err := tx.ExecContext(ctx, `
		UPDATE accessors
		SET `+strings.Join(keys, ", ")+`
		WHERE is_group=0 AND name=?
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
	if n == 0 {
		return fmt.Errorf("no user affected")
	}
	return nil
}
