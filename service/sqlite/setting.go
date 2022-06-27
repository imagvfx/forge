package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/imagvfx/forge"
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

func FindUserSettings(db *sql.DB, ctx context.Context, find forge.UserSettingFinder) ([]*forge.UserSetting, error) {
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

func findUserSettings(tx *sql.Tx, ctx context.Context, find forge.UserSettingFinder) ([]*forge.UserSetting, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
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
	setting := make(map[string]*forge.UserSetting)
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
			s = &forge.UserSetting{
				User:               user,
				UpdateMarkerLasts:  -1,
				SearchResultExpand: true,
			}
		}
		switch key {
		case "entry_page_selected_category":
			err = json.Unmarshal([]byte(value), &s.EntryPageSelectedCategory)
		case "entry_page_show_hidden_property":
			err = json.Unmarshal([]byte(value), &s.EntryPageShowHiddenProperty)
		case "entry_page_search_entry_type":
			err = json.Unmarshal([]byte(value), &s.EntryPageSearchEntryType)
		case "entry_page_property_filter":
			err = json.Unmarshal([]byte(value), &s.EntryPagePropertyFilter)
		case "entry_page_sort_property":
			err = json.Unmarshal([]byte(value), &s.EntryPageSortProperty)
		case "picked_property":
			err = json.Unmarshal([]byte(value), &s.PickedProperty)
		case "quick_searches":
			err = json.Unmarshal([]byte(value), &s.QuickSearches)
		case "pinned_paths_v2":
			var pinnedEntIDs []int
			err = json.Unmarshal([]byte(value), &pinnedEntIDs)
			if err == nil {
				s.PinnedPaths = make([]string, 0, len(pinnedEntIDs))
				for _, entID := range pinnedEntIDs {
					ent, err := getEntryByID(tx, ctx, entID)
					if err != nil {
						var e *forge.NotFoundError
						if !errors.As(err, &e) {
							return nil, err
						}
						// The entry deleted, don't show it.
						continue
					}
					s.PinnedPaths = append(s.PinnedPaths, ent.Path)
				}
			}
		case "update_marker_lasts":
			err = json.Unmarshal([]byte(value), &s.UpdateMarkerLasts)
		case "search_result_expand":
			err = json.Unmarshal([]byte(value), &s.SearchResultExpand)
		case "search_view":
			err = json.Unmarshal([]byte(value), &s.SearchView)
		case "entry_group_by":
			err = json.Unmarshal([]byte(value), &s.EntryGroupBy)
		case "copy_path_remap":
			err = json.Unmarshal([]byte(value), &s.CopyPathRemap)
		default:
			// It may have legacy settings, nothing to do with them.
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("invalid value for user setting key: %v", key)
		}
		setting[user] = s
	}
	settings := make([]*forge.UserSetting, 0, len(setting))
	for _, s := range setting {
		// set default values
		if s.UpdateMarkerLasts < 0 {
			s.UpdateMarkerLasts = 1
		}
		settings = append(settings, s)
	}
	return settings, nil
}

func GetUserSetting(db *sql.DB, ctx context.Context, user string) (*forge.UserSetting, error) {
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

func getUserSetting(tx *sql.Tx, ctx context.Context, user string) (*forge.UserSetting, error) {
	settings, err := findUserSettings(tx, ctx, forge.UserSettingFinder{User: &user})
	if err != nil {
		return nil, err
	}
	if len(settings) == 0 {
		return &forge.UserSetting{User: user}, nil
	}
	return settings[0], nil
}

func UpdateUserSetting(db *sql.DB, ctx context.Context, upd forge.UserSettingUpdater) error {
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

func updateUserSetting(tx *sql.Tx, ctx context.Context, upd forge.UserSettingUpdater) error {
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	if user != upd.User {
		return forge.Unauthorized("cannot update setting for another user")
	}
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
	case "entry_page_selected_category":
		selectedCategory, ok := upd.Value.(string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(selectedCategory)
		if err != nil {
			return err
		}
	case "entry_page_show_hidden_property":
		showHidden, ok := upd.Value.(string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(showHidden)
		if err != nil {
			return err
		}
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
	case "picked_property":
		pickedProp := setting.PickedProperty
		if pickedProp == nil {
			pickedProp = make(map[string]string)
		}
		pickedProp, ok := upd.Value.(map[string]string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		for entryType, p := range pickedProp {
			pickedProp[entryType] = p
		}
		value, err = json.Marshal(pickedProp)
		if err != nil {
			return err
		}
	case "quick_searches":
		arng, ok := upd.Value.(forge.QuickSearchArranger)
		if !ok {
			return fmt.Errorf("invalid type of value: %v", upd.Key)
		}
		search := arng.KV
		if search.K == "" {
			return fmt.Errorf("%v: name empty", upd.Key)
		}
		key := func(a forge.StringKV) string { return a.K }
		searches := forge.Arrange(setting.QuickSearches, search, arng.Index, key, false)
		value, err = json.Marshal(searches)
		if err != nil {
			return err
		}
	case "pinned_paths":
		// Correct the key with internal represetation version.
		upd.Key = "pinned_paths_v2"
		pinned, ok := upd.Value.(forge.StringSliceArranger)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		_, err := getEntryID(tx, ctx, pinned.Value)
		if err != nil {
			return err
		}
		key := func(a string) string { return a }
		pinnedPaths := forge.Arrange(setting.PinnedPaths, pinned.Value, pinned.Index, key, false)
		// Convert to internal represetation.
		pinnedIDs := make([]int, 0)
		for _, p := range pinnedPaths {
			id, err := getEntryID(tx, ctx, p)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return err
				}
				continue
			}
			pinnedIDs = append(pinnedIDs, id)
		}
		value, err = json.Marshal(pinnedIDs)
		if err != nil {
			return err
		}
	case "update_marker_lasts":
		last, ok := upd.Value.(int)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(last)
		if err != nil {
			return err
		}
	case "search_result_expand":
		expand, ok := upd.Value.(bool)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(expand)
		if err != nil {
			return err
		}
	case "search_view":
		view, ok := upd.Value.(string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(view)
		if err != nil {
			return err
		}
	case "entry_group_by":
		groupBy, ok := upd.Value.(string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(groupBy)
		if err != nil {
			return err
		}
	case "copy_path_remap":
		remap, ok := upd.Value.(string)
		if !ok {
			return fmt.Errorf("invalid update value type for key: %v", upd.Key)
		}
		value, err = json.Marshal(remap)
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
