package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createUserSettingsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS user_settings (
			id INTEGER PRIMARY KEY,
			user_id INTERGER NOT NULL,
			entry_page_tab STRING,
			entry_page_property_filter STRING,
			FOREIGN KEY (user_id) REFERENCES accessors (id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_user_settings_user_id ON user_settings (user_id)`)
	return err
}

func FindUserSettings(db *sql.DB, ctx context.Context, find service.UserSettingFinder) ([]*service.UserSetting, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	settings, err := findUserSettings(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func findUserSettings(tx *sql.Tx, ctx context.Context, find service.UserSettingFinder) ([]*service.UserSetting, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.User != nil {
		keys = append(keys, "accessors.name=?")
		vals = append(vals, *find.User)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			user_settings.id,
			accessors.name,
			user_settings.entry_page_tab,
			user_settings.entry_page_property_filter
		FROM user_settings
		LEFT JOIN accessors ON user_settings.user_id = accessors.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var filter []byte
	settings := make([]*service.UserSetting, 0)
	for rows.Next() {
		s := &service.UserSetting{}
		err := rows.Scan(
			&s.ID,
			&s.User,
			&s.EntryPageTab,
			&filter,
		)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(filter, &s.EntryPagePropertyFilter)
		if err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}

func GetUserSetting(db *sql.DB, ctx context.Context, user string) (*service.UserSetting, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = getUser(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	s, err := getUserSetting(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func getUserSetting(tx *sql.Tx, ctx context.Context, user string) (*service.UserSetting, error) {
	settings, err := findUserSettings(tx, ctx, service.UserSettingFinder{User: &user})
	if err != nil {
		return nil, err
	}
	if len(settings) == 0 {
		return nil, service.NotFound("user setting not found")
	}
	return settings[0], nil
}

// addUserSetting is not exposed but called by AddUser.
func addUserSetting(tx *sql.Tx, ctx context.Context, s *service.UserSetting) error {
	userID, err := getUserID(tx, ctx, s.User)
	if err != nil {
		return err
	}
	filter, err := json.Marshal(s.EntryPagePropertyFilter)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO user_settings (
			user_id,
			entry_page_tab,
			entry_page_property_filter
		)
		VALUES (?, ?, ?)
	`,
		userID,
		s.EntryPageTab,
		filter,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = int(id)
	return nil
}

func UpdateUserSetting(db *sql.DB, ctx context.Context, upd service.UserSettingUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = updateUserSetting(tx, ctx, upd)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateUserSetting(tx *sql.Tx, ctx context.Context, upd service.UserSettingUpdater) error {
	u, err := getUserSetting(tx, ctx, upd.User)
	if err != nil {
		return err
	}
	userID, err := getUserID(tx, ctx, upd.User)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.EntryPageTab != nil {
		keys = append(keys, "entry_page_tab=?")
		vals = append(vals, *upd.EntryPageTab)
	}
	var filterBytes []byte
	if upd.EntryPagePropertyFilter != nil {
		// Update current filter.
		filter := u.EntryPagePropertyFilter
		if filter == nil {
			filter = make(map[string]string)
		}
		for entryType, f := range upd.EntryPagePropertyFilter {
			filter[entryType] = f
		}
		filterBytes, err = json.Marshal(filter)
		if err != nil {
			return err
		}
		keys = append(keys, "entry_page_property_filter=?")
		vals = append(vals, filterBytes)
	}
	vals = append(vals, userID) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE user_settings
		SET `+strings.Join(keys, ", ")+`
		WHERE user_id=?`,
		vals...,
	)
	if err != nil {
		return err
	}
	return nil
}
