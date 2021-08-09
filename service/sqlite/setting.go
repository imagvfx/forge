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
			entry_page_search_entry_type STRING,
			entry_page_property_filter STRING,
			entry_page_sort_property STRING,
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
			user_settings.entry_page_search_entry_type,
			user_settings.entry_page_property_filter,
			user_settings.entry_page_sort_property
		FROM user_settings
		LEFT JOIN accessors ON user_settings.user_id = accessors.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := make([]*service.UserSetting, 0)
	for rows.Next() {
		s := &service.UserSetting{}
		var filter []byte
		var sortProp []byte
		err := rows.Scan(
			&s.ID,
			&s.User,
			&s.EntryPageSearchEntryType,
			&filter,
			&sortProp,
		)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(filter, &s.EntryPagePropertyFilter)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(sortProp, &s.EntryPageSortProperty)
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

// addDefaultUserSetting is not exposed to server but called by AddUser.
func addDefaultUserSetting(tx *sql.Tx, ctx context.Context, user string) error {
	userID, err := getUserID(tx, ctx, user)
	if err != nil {
		return err
	}
	filter, err := json.Marshal(map[string]string{})
	if err != nil {
		return err
	}
	sortProp, err := json.Marshal(map[string]string{})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_settings (
			user_id,
			entry_page_search_entry_type,
			entry_page_property_filter,
			entry_page_sort_property
		)
		VALUES (?, ?, ?, ?)
	`,
		userID,
		"",
		filter,
		sortProp,
	)
	if err != nil {
		return err
	}
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
	setting, err := getUserSetting(tx, ctx, upd.User)
	if err != nil {
		return err
	}
	userID, err := getUserID(tx, ctx, upd.User)
	if err != nil {
		return err
	}
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.EntryPageSearchEntryType != nil {
		keys = append(keys, "entry_page_search_entry_type=?")
		vals = append(vals, *upd.EntryPageSearchEntryType)
	}
	var filterBytes []byte
	if upd.EntryPagePropertyFilter != nil {
		filter := setting.EntryPagePropertyFilter
		if filter == nil {
			filter = make(map[string]string)
		}
		for entryType, f := range upd.EntryPagePropertyFilter {
			// update
			filter[entryType] = f
		}
		filterBytes, err = json.Marshal(filter)
		if err != nil {
			return err
		}
		keys = append(keys, "entry_page_property_filter=?")
		vals = append(vals, filterBytes)
	}
	var sortPropBytes []byte
	if upd.EntryPageSortProperty != nil {
		sortProp := setting.EntryPageSortProperty
		if sortProp == nil {
			sortProp = make(map[string]string)
		}
		for entryType, prop := range upd.EntryPageSortProperty {
			// update
			sortProp[entryType] = prop
		}
		sortPropBytes, err = json.Marshal(sortProp)
		if err != nil {
			return err
		}
		keys = append(keys, "entry_page_sort_property=?")
		vals = append(vals, sortPropBytes)
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
