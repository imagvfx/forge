package sqlite

import (
	"context"
	"database/sql"
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
			section STRING NOT NULL,
			key STRING NOT NULL,
			value STRING NOT NULL,
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
		secs[section].Data[key] = value
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
	if len(data) == 0 || data[0].Section != section {
		return nil, forge.NotFound("user data section is not exists: %v", section)
	}
	sec := data[0]
	return sec, nil
}

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
	data, err := findUserData(tx, ctx, forge.UserDataFinder{User: user, Section: &section, Key: &key})
	if err != nil {
		return "", err
	}
	if len(data) == 0 || data[0].Section != section {
		return "", forge.NotFound("user data is not exists: %v/%v", section, key)
	}
	sec := data[0]
	value, ok := sec.Data[key]
	if !ok {
		return "", forge.NotFound("user data is not exists: %v/%v", section, key)
	}
	return value, nil
}

func AddUserData(db *sql.DB, ctx context.Context, user, section, key, value string) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return forge.Unauthorized("cannot add user data to another user")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addUserData(tx, ctx, user, section, key, value)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addUserData(tx *sql.Tx, ctx context.Context, user, section, key, value string) error {
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
	`,
		userID, section, key, value,
	)
	if err != nil {
		return err
	}
	return nil
}

func UpdateUserData(db *sql.DB, ctx context.Context, user, section, key, value string) error {
	ctxUser := forge.UserNameFromContext(ctx)
	if ctxUser == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if ctxUser != user {
		return forge.Unauthorized("cannot update another user's data")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateUserData(tx, ctx, user, section, key, value)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateUserData(tx *sql.Tx, ctx context.Context, user, section, key, value string) error {
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	_, err = getUserData(tx, ctx, user, section, key)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE user_data
		SET
			value=?
		WHERE user_id=? AND section=? AND key=?
	`,
		value, userID, section, key,
	)
	if err != nil {
		return err
	}
	return nil
}

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
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	_, err = getUserData(tx, ctx, user, section, key)
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
