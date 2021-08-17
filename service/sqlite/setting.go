package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
			entry_page_quick_search STRING,
			pinned_paths STRING,
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
			user_settings.entry_page_sort_property,
			user_settings.entry_page_quick_search,
			user_settings.pinned_paths
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
		var quickSearch []byte
		var pinnedPaths []byte
		err := rows.Scan(
			&s.ID,
			&s.User,
			&s.EntryPageSearchEntryType,
			&filter,
			&sortProp,
			&quickSearch,
			&pinnedPaths,
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
		err = json.Unmarshal(quickSearch, &s.EntryPageQuickSearch)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(pinnedPaths, &s.PinnedPaths)
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
	quickSearch, err := json.Marshal(map[string]string{})
	if err != nil {
		return err
	}
	pinnedPaths, err := json.Marshal([]string{})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO user_settings (
			user_id,
			entry_page_search_entry_type,
			entry_page_property_filter,
			entry_page_sort_property,
			entry_page_quick_search,
			pinned_paths
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		userID,
		"",
		filter,
		sortProp,
		quickSearch,
		pinnedPaths,
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
	var pinnedBytes []byte
	if upd.PinnedPath != nil {
		path := upd.PinnedPath.Path
		n := upd.PinnedPath.Index
		// Insert/Remove pinned paths by specifiying the index n.
		// n < 0 means remove the path,
		// n >= len(oldPinned) means append it to the last.
		//
		// ex) when pinned is initialized as []string{"a", "b", "c"}
		//
		//     pinned = []string{"b", "c"}       where path = "a" and n = -1
		//     pinned = []string{"a", "b", "c"}  where path = "a" and n = 0
		//     pinned = []string{"b", "a", "c"}  where path = "a" and n = 1
		//     pinned = []string{"b", "c", "a"}  where path = "a" and n = 2
		//
		oldPinned := setting.PinnedPaths
		if oldPinned == nil {
			oldPinned = make([]string, 0)
		}
		pinned := make([]string, 0, len(oldPinned)+1)
		for _, p := range oldPinned {
			// Remove the path from currently pinned, if exists.
			if p != path {
				pinned = append(pinned, p)
			}
		}
		if n < 0 {
			// remove: already done
		} else if n < len(oldPinned) {
			// insert el at n
			pinned = append(pinned, "")
			copy(pinned[n+1:], pinned[n:])
			pinned[n] = path
		} else {
			pinned = append(pinned, path)
		}
		pinnedBytes, err = json.Marshal(pinned)
		if err != nil {
			return err
		}
		keys = append(keys, "pinned_paths=?")
		vals = append(vals, pinnedBytes)
	}
	var quickSearchBytes []byte
	if upd.EntryPageQuickSearch != nil {
		quickSearch := setting.EntryPageQuickSearch
		if quickSearch == nil {
			quickSearch = make(map[string]string)
		}
		for name, query := range upd.EntryPageQuickSearch {
			if name == "" {
				// TODO: check this for other map[string]string settings as well
				return fmt.Errorf("quick search name is empty")
			}
			if query == "" {
				// remove the quick search instead of add
				delete(quickSearch, name)
			} else {
				quickSearch[name] = query
			}
		}
		quickSearchBytes, err = json.Marshal(quickSearch)
		if err != nil {
			return err
		}
		keys = append(keys, "entry_page_quick_search=?")
		vals = append(vals, quickSearchBytes)
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
