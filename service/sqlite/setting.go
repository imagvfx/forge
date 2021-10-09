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
			key STRING NOT NULL,
			value STRING NOT NULL,
			FOREIGN KEY (user_id) REFERENCES accessors (id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_user_settings_user_id ON user_settings (user_id, key)`)
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
			accessors.name,
			user_settings.key,
			user_settings.value
		FROM user_settings
		LEFT JOIN accessors ON user_settings.user_id = accessors.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	setting := make(map[string]*service.UserSetting)
	for rows.Next() {
		var user, key, value string
		err := rows.Scan(
			&user,
			&key,
			&value,
		)
		if err != nil {
			return nil, err
		}
		s := setting[user]
		if s == nil {
			s = &service.UserSetting{
				User: user,
			}
		}
		switch key {
		case "entry_page_search_entry_type":
			err = json.Unmarshal([]byte(value), &s.EntryPageSearchEntryType)
		case "entry_page_property_filter":
			err = json.Unmarshal([]byte(value), &s.EntryPagePropertyFilter)
		case "entry_page_sort_property":
			err = json.Unmarshal([]byte(value), &s.EntryPageSortProperty)
		case "quick_searches":
			err = json.Unmarshal([]byte(value), &s.QuickSearches)
		case "pinned_paths":
			err = json.Unmarshal([]byte(value), &s.PinnedPaths)
		default:
			// It may have legacy settings, nothing to do with them.
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("invalid value for user setting key: %v", key)
		}
		setting[user] = s
	}
	settings := make([]*service.UserSetting, 0, len(setting))
	for _, s := range setting {
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
		return &service.UserSetting{User: user}, nil
	}
	return settings[0], nil
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
	var value []byte
	switch upd.Key {
	case "entry_page_search_entry_type":
		updateSearchEntryType, ok := upd.Value.(string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(updateSearchEntryType)
		if err != nil {
			return err
		}
	case "entry_page_property_filter":
		propFilter := setting.EntryPagePropertyFilter
		if propFilter == nil {
			propFilter = make(map[string]string)
		}
		updatePropFilter, ok := upd.Value.(map[string]string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		for entryType, f := range updatePropFilter {
			propFilter[entryType] = f
		}
		value, err = json.Marshal(propFilter)
		if err != nil {
			return err
		}
	case "entry_page_sort_property":
		sortProp := setting.EntryPageSortProperty
		if sortProp == nil {
			sortProp = make(map[string]string)
		}
		updateSortProperty, ok := upd.Value.(map[string]string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		for entryType, p := range updateSortProperty {
			if len(p) == 0 {
				return fmt.Errorf("sort order and property not defined")
			}
			order := p[:1]
			if order != "+" && order != "-" {
				// "+" means ascending, "-" means descending
				return fmt.Errorf("invalid sort order: want + or -, got %v", order)
			}
			sortProp[entryType] = p
		}
		value, err = json.Marshal(sortProp)
		if err != nil {
			return err
		}
	case "quick_searches":
		switch val := upd.Value.(type) {
		case []service.StringKV:
			updateQuickSearch := val
			quickSearches := setting.QuickSearches
			if quickSearches == nil {
				quickSearches = make([]service.StringKV, 0)
			}
			for _, updQs := range updateQuickSearch {
				if updQs.K == "" {
					// TODO: check this for other settings as well
					return fmt.Errorf("quick search name is empty")
				}
				if updQs.V == "" {
					// Remove the quick search.
					idx := -1
					for i, s := range quickSearches {
						if s.K == updQs.K {
							idx = i
							break
						}
					}
					if idx != -1 {
						quickSearches = append(quickSearches[:idx], quickSearches[idx+1:]...)
					}
				} else {
					// Add or update the quick search.
					found := false
					for i, qs := range quickSearches {
						if qs.K != updQs.K {
							continue
						}
						found = true
						quickSearches[i] = updQs
					}
					if !found {
						quickSearches = append(quickSearches, updQs)
					}
				}
			}
			value, err = json.Marshal(quickSearches)
			if err != nil {
				return err
			}
		case service.QuickSearchArranger:
			arr := val
			if arr.Name == "" {
				return fmt.Errorf("%v: name empty", upd.Key)
			}
			oldSearches := setting.QuickSearches
			if oldSearches == nil {
				oldSearches = make([]service.StringKV, 0)
			}
			searches := make([]service.StringKV, 0, len(oldSearches)+1)
			search := service.StringKV{}
			for _, s := range oldSearches {
				if s.K == arr.Name {
					search = s
					continue
				}
				searches = append(searches, s)
			}
			if search.K == "" {
				return fmt.Errorf("%v: not found quick search: %v", upd.Key, arr.Name)
			}
			switch n := arr.Index; {
			case n < 0:
				// remove: already done
			case n < len(oldSearches):
				// insert el at n
				searches = append(searches, service.StringKV{})
				copy(searches[n+1:], searches[n:])
				searches[n] = search
			default:
				searches = append(searches, search)
			}
			value, err = json.Marshal(searches)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid type of value: %v", upd.Key)
		}
	case "pinned_paths":
		updatePinnedPath, ok := upd.Value.(service.PinnedPathArranger)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		path := updatePinnedPath.Path
		n := updatePinnedPath.Index
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
		value, err = json.Marshal(pinned)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown user setting key: %v", upd.Key)
	}
	_, err = tx.ExecContext(ctx, `
		REPLACE INTO user_settings (
			user_id,
			key,
			value
		)
		VALUES (?, ?, ?)`,
		userID, upd.Key, value,
	)
	if err != nil {
		return err
	}
	return nil
}
