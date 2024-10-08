package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/imagvfx/forge"
)

func createUserDataTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS user_data (
			id INTEGER PRIMARY KEY,
			user_id INTERGER NOT NULL,
			section TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES accessors (id)
			UNIQUE (user_id, section, key)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_user_data_user_id_section ON user_data (user_id, section)`)
	return err
}

func FindUserData(db *sql.DB, ctx context.Context, find forge.UserDataFinder) ([]*forge.UserDataSection, error) {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return nil, forge.Unauthorized("context user unspecified")
	}
	if ctxUser != find.User {
		return nil, forge.Unauthorized("cannot get another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	data, err := findUserData(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func findUserData(tx *sql.Tx, ctx context.Context, find forge.UserDataFinder) ([]*forge.UserDataSection, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	keys = append(keys, "accessors.name=?")
	vals = append(vals, find.User)
	if find.Section != nil {
		keys = append(keys, "user_data.section=?")
		vals = append(vals, *find.Section)
	}
	if find.Key != nil {
		keys = append(keys, "user_data.key=?")
		vals = append(vals, *find.Key)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			user_data.section,
			user_data.key,
			user_data.value
		FROM user_data
		LEFT JOIN accessors ON user_data.user_id = accessors.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	secs := make(map[string]*forge.UserDataSection)
	for rows.Next() {
		var section, key, value string
		err := rows.Scan(
			&section,
			&key,
			&value,
		)
		if err != nil {
			return nil, err
		}
		if secs[section] == nil {
			secs[section] = &forge.UserDataSection{
				Section: section,
				Data:    make(map[string]string),
			}
		}
		if key != "" {
			secs[section].Data[key] = value
		}
	}
	data := make([]*forge.UserDataSection, 0)
	for _, s := range secs {
		data = append(data, s)
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i].Section < data[j].Section
	})
	return data, nil
}

func AddUserDataSection(db *sql.DB, ctx context.Context, user, section string) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return forge.Unauthorized("cannot get another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addUserDataSection(tx, ctx, user, section)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addUserDataSection(tx *sql.Tx, ctx context.Context, user, section string) error {
	if section == "" {
		return fmt.Errorf("user data section cannot be empty")
	}
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	_, err = getUserDataSection(tx, ctx, user, section)
	if err == nil {
		return fmt.Errorf("user data section is already exists")
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_data (
			user_id,
			section,
			key,
			value
		)
		VALUES (?, ?, '', '')
	`,
		userID, section,
	)
	if err != nil {
		return err
	}
	return nil
}

func GetUserDataSection(db *sql.DB, ctx context.Context, user, section string) (*forge.UserDataSection, error) {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return nil, forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return nil, forge.Unauthorized("cannot get another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	sec, err := getUserDataSection(tx, ctx, user, section)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return sec, nil
}

func getUserDataSection(tx *sql.Tx, ctx context.Context, user, section string) (*forge.UserDataSection, error) {
	data, err := findUserData(tx, ctx, forge.UserDataFinder{User: user, Section: &section})
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, forge.NotFound("user data section is not exists: %v", section)
	}
	sec := data[0]
	if sec.Section != section {
		return nil, fmt.Errorf("got wrong user data section: want %v, got %v", section, sec.Section)
	}
	return sec, nil
}

// GetUserData returns a user data from the sql file.
// NOTE: It doesn't raise error even if the key (even the section) doesn't exists in user_data table.
// It returns an empty string, instead.
func GetUserData(db *sql.DB, ctx context.Context, user, section, key string) (string, error) {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return "", forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return "", forge.Unauthorized("cannot get another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	value, err := getUserData(tx, ctx, user, section, key)
	err = tx.Commit()
	if err != nil {
		return "", err
	}
	return value, nil
}

func getUserData(tx *sql.Tx, ctx context.Context, user, section, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("user data key cannot be empty")
	}
	data, err := findUserData(tx, ctx, forge.UserDataFinder{User: user, Section: &section, Key: &key})
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	sec := data[0]
	if sec.Section != section {
		return "", fmt.Errorf("got wrong section of user_data: want %v, got %v", section, sec.Section)
	}
	value, ok := sec.Data[key]
	if !ok {
		return "", forge.NotFound("user data is not exists: %v/%v", section, key)
	}
	return value, nil
}

// SetUserData sets a user data to the sql file.
// It adds if the user data is not exists.
// It will update the value instead, if the user data is already exists.
func SetUserData(db *sql.DB, ctx context.Context, user, section, key, value string) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return forge.Unauthorized("cannot set user-data to another user")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = setUserData(tx, ctx, user, section, key, value)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func setUserData(tx *sql.Tx, ctx context.Context, user, section, key, value string) error {
	if section == "" {
		return fmt.Errorf("user data section cannot be empty")
	}
	if key == "" {
		return fmt.Errorf("user data key cannot be empty")
	}
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_data (
			user_id,
			section,
			key,
			value
		)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (user_id, section, key) DO
		UPDATE SET value=?
	`,
		userID, section, key, value, value,
	)
	if err != nil {
		return err
	}
	return nil
}

// DeleteUserData deletes a user data from the sql file.
// NOTE: It will not return an error even the user data wasn't existed.
// The user should check it explicitly, if needed.
func DeleteUserData(db *sql.DB, ctx context.Context, user, section, key string) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return forge.Unauthorized("cannot delete another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteUserData(tx, ctx, user, section, key)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteUserData(tx *sql.Tx, ctx context.Context, user, section, key string) error {
	if key == "" {
		return fmt.Errorf("user data key cannot be empty")
	}
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM user_data
		WHERE user_id=? AND section=? AND key=?
	`,
		userID, section, key,
	)
	if err != nil {
		return err
	}
	return nil
}

func DeleteUserDataSection(db *sql.DB, ctx context.Context, user, section string) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return forge.Unauthorized("cannot delete another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteUserDataSection(tx, ctx, user, section)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteUserDataSection(tx *sql.Tx, ctx context.Context, user, section string) error {
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	_, err = getUserDataSection(tx, ctx, user, section)
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return fmt.Errorf("user data section is not exists")
		}
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM user_data
		WHERE user_id=? AND section=?
	`,
		userID, section,
	)
	if err != nil {
		return err
	}
	return nil
}
