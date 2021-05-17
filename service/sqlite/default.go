package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createDefaultSubEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS default_sub_entries (
			id INTEGER PRIMARY KEY,
			entry_type_id INTEGER NOT NULL,
			name STRING NOT NULL,
			sub_entry_type_id INTEGER NOT NULL,
			FOREIGN KEY (entry_type_id) REFERENCES entry_types (id),
			FOREIGN KEY (sub_entry_type_id) REFERENCES sub_entry_types (id),
			UNIQUE (entry_type_id, name)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_default_sub_entries_entry_type_id ON default_sub_entries (entry_type_id)`)
	return err
}

func createDefaultPropertiesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS default_properties (
			id INTEGER PRIMARY KEY,
			entry_type_id INTEGER NOT NULL,
			name STRING NOT NULL,
			type STRING NOT NULL,
			value STRING NOT NULL,
			FOREIGN KEY (entry_type_id) REFERENCES entry_types (id),
			UNIQUE (entry_type_id, name)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_default_properties_entry_type_id ON default_properties (entry_type_id)`)
	return err
}

func createDefaultEnvironsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS default_environs (
			id INTEGER PRIMARY KEY,
			entry_type_id INTEGER NOT NULL,
			name STRING NOT NULL,
			type STRING NOT NULL,
			value STRING NOT NULL,
			FOREIGN KEY (entry_type_id) REFERENCES entry_types (id),
			UNIQUE (entry_type_id, name)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_default_environs_entry_type_id ON default_environs (entry_type_id)`)
	return err
}

func FindDefaults(db *sql.DB, ctx context.Context, find service.DefaultFinder) ([]*service.Default, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	defaults := make([]*service.Default, 0)
	subs, err := findDefaultSubEntries(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, subs...)
	props, err := findDefaultProperties(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, props...)
	envs, err := findDefaultEnvirons(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, envs...)
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return defaults, nil
}

func findDefaultSubEntries(tx *sql.Tx, ctx context.Context, find service.DefaultFinder) ([]*service.Default, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "default_sub_entries.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			default_sub_entries.name,
			sub_types.name
		FROM default_sub_entries
		LEFT JOIN entry_types ON default_sub_entries.entry_type_id = entry_types.id
		LEFT JOIN sub_entry_types ON default_sub_entries.sub_entry_type_id = sub_entry_types.id
		LEFT JOIN entry_types AS sub_types ON sub_entry_types.sub_id = sub_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.Default, 0)
	for rows.Next() {
		d := &service.Default{
			Category: "sub_entry",
		}
		err := rows.Scan(
			&d.EntryType,
			&d.Name,
			&d.Type,
		)
		if err != nil {
			return nil, err
		}
		defaults = append(defaults, d)
	}
	return defaults, nil
}

func findDefaultProperties(tx *sql.Tx, ctx context.Context, find service.DefaultFinder) ([]*service.Default, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "default_properties.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			default_properties.name,
			default_properties.type,
			default_properties.value
		FROM default_properties
		LEFT JOIN entry_types ON default_properties.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.Default, 0)
	for rows.Next() {
		d := &service.Default{
			Category: "property",
		}
		err := rows.Scan(
			&d.EntryType,
			&d.Name,
			&d.Type,
			&d.Value,
		)
		if err != nil {
			return nil, err
		}
		defaults = append(defaults, d)
	}
	return defaults, nil
}

func findDefaultEnvirons(tx *sql.Tx, ctx context.Context, find service.DefaultFinder) ([]*service.Default, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "default_environs.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			default_environs.name,
			default_environs.type,
			default_environs.value
		FROM default_environs
		LEFT JOIN entry_types ON default_environs.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.Default, 0)
	for rows.Next() {
		d := &service.Default{
			Category: "environ",
		}
		err := rows.Scan(
			&d.EntryType,
			&d.Name,
			&d.Type,
			&d.Value,
		)
		if err != nil {
			return nil, err
		}
		defaults = append(defaults, d)
	}
	return defaults, nil
}

func AddDefault(db *sql.DB, ctx context.Context, d *service.Default) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := service.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return service.Unauthorized("user doesn't have permission to add default: %v", user)
	}
	switch d.Category {
	case "sub_entry":
		err = addDefaultSubEntry(tx, ctx, d)
		if err != nil {
			return err
		}
	case "property":
		err = addDefaultProperty(tx, ctx, d)
		if err != nil {
			return err
		}
	case "environ":
		err = addDefaultEnviron(tx, ctx, d)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid category for default: %v", d.Category)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addDefaultSubEntry(tx *sql.Tx, ctx context.Context, d *service.Default) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	subTypeID, err := getSubEntryTypeID(tx, ctx, d.EntryType, d.Type)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO default_sub_entries (
			entry_type_id,
			name,
			sub_entry_type_id
		)
		VALUES (?, ?, ?)
	`,
		typeID,
		d.Name,
		subTypeID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	d.ID = int(id)
	return nil
}

func addDefaultProperty(tx *sql.Tx, ctx context.Context, d *service.Default) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO default_properties (
			entry_type_id,
			name,
			type,
			value
		)
		VALUES (?, ?, ?, ?)
	`,
		typeID,
		d.Name,
		d.Type,
		d.Value,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	d.ID = int(id)
	return nil
}

func addDefaultEnviron(tx *sql.Tx, ctx context.Context, d *service.Default) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO default_environs (
			entry_type_id,
			name,
			type,
			value
		)
		VALUES (?, ?, ?, ?)
	`,
		typeID,
		d.Name,
		d.Type,
		d.Value,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	d.ID = int(id)
	return nil
}

func UpdateDefault(db *sql.DB, ctx context.Context, upd service.DefaultUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := service.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return service.Unauthorized("user doesn't have permission to update default: %v", user)
	}
	switch upd.Category {
	case "sub_entry":
		err := updateDefaultSubEntry(tx, ctx, upd)
		if err != nil {
			return err
		}
	case "property":
		err := updateDefaultProperty(tx, ctx, upd)
		if err != nil {
			return err
		}
	case "environ":
		err := updateDefaultEnviron(tx, ctx, upd)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid category for default: %v", upd.Category)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateDefaultSubEntry(tx *sql.Tx, ctx context.Context, upd service.DefaultUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Type != nil {
		keys = append(keys, "sub_entry_type_id=?")
		subTypeID, err := getEntryTypeID(tx, ctx, *upd.Type)
		if err != nil {
			return err
		}
		vals = append(vals, subTypeID)
	}
	if upd.Value != nil {
		return fmt.Errorf("default sub-entry updater shouldn't have value")
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update default: %v %v %v", upd.EntryType, "sub_entry", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE default_sub_entries
		SET `+strings.Join(keys, ", ")+`
		WHERE entry_type_id=? AND name=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	return nil
}

func updateDefaultProperty(tx *sql.Tx, ctx context.Context, upd service.DefaultUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Type != nil {
		keys = append(keys, "type=?")
		vals = append(vals, *upd.Type)
	}
	if upd.Value != nil {
		keys = append(keys, "value=?")
		vals = append(vals, *upd.Value)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update default: %v %v %v", upd.EntryType, "property", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE default_properties
		SET `+strings.Join(keys, ", ")+`
		WHERE entry_type_id=? AND name=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	return nil
}

func updateDefaultEnviron(tx *sql.Tx, ctx context.Context, upd service.DefaultUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Type != nil {
		keys = append(keys, "type=?")
		vals = append(vals, *upd.Type)
	}
	if upd.Value != nil {
		keys = append(keys, "value=?")
		vals = append(vals, *upd.Value)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update default: %v %v %v", upd.EntryType, "environ", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE default_environs
		SET `+strings.Join(keys, ", ")+`
		WHERE entry_type_id=? AND name=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	return nil
}

func DeleteDefault(db *sql.DB, ctx context.Context, entryType, ctg, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := service.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return service.Unauthorized("user doesn't have permission to delete default: %v", user)
	}
	switch ctg {
	case "sub_entry":
		err := deleteDefaultSubEntry(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	case "property":
		err := deleteDefaultProperty(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	case "environ":
		err := deleteDefaultEnviron(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid category for default: %v", ctg)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteDefaultSubEntry(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM default_sub_entries
		WHERE entry_type_id=? AND name=?
	`,
		typeID,
		name,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("no such default for entry type %v: %v %v", entryType, "sub_entry", name)
	}
	return nil
}

func deleteDefaultProperty(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM default_properties
		WHERE entry_type_id=? AND name=?
	`,
		typeID,
		name,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("no such default for entry type %v: %v %v", entryType, "property", name)
	}
	return nil
}

func deleteDefaultEnviron(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM default_environs
		WHERE entry_type_id=? AND name=?
	`,
		typeID,
		name,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return service.NotFound("no such default for entry type %v: %v %v", entryType, "environ", name)
	}
	return nil
}
