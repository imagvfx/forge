package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createEntryDefaultsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entry_defaults (
			id INTEGER PRIMARY KEY,
			entry_type_id INTEGER NOT NULL,
			category STRING NOT NULL,
			name STRING NOT NULL,
			type STRING NOT NULL,
			value STRING NOT NULL,
			FOREIGN KEY (entry_type_id) REFERENCES entry_types (id),
			UNIQUE (entry_type_id, category, name)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_entry_defaults_entry_type_id ON entry_defaults (entry_type_id)`)
	return err
}

func createEntryDefaultSubEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entry_default_sub_entries (
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
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_entry_default_sub_entries_entry_type_id ON entry_default_sub_entries (entry_type_id)`)
	return err
}

func createEntryDefaultPropertiesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entry_default_properties (
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
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_entry_default_properties_entry_type_id ON entry_default_properties (entry_type_id)`)
	return err
}

func createEntryDefaultEnvironsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entry_default_environs (
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
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_entry_default_environs_entry_type_id ON entry_default_environs (entry_type_id)`)
	return err
}

func FindEntryDefaults(db *sql.DB, ctx context.Context, find service.EntryDefaultFinder) ([]*service.EntryDefault, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	defaults := make([]*service.EntryDefault, 0)
	subs, err := findEntryDefaultSubEntries(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, subs...)
	props, err := findEntryDefaultProperties(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, props...)
	envs, err := findEntryDefaultEnvirons(tx, ctx, find)
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

func findEntryDefaults(tx *sql.Tx, ctx context.Context, find service.EntryDefaultFinder) ([]*service.EntryDefault, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Category != nil {
		keys = append(keys, "entry_defaults.category=?")
		vals = append(vals, *find.Category)
	}
	if find.Name != nil {
		keys = append(keys, "entry_defaults.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			entry_defaults.category,
			entry_defaults.name,
			entry_defaults.type,
			entry_defaults.value
		FROM entry_defaults
		LEFT JOIN entry_types ON entry_defaults.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.EntryDefault, 0)
	for rows.Next() {
		d := &service.EntryDefault{}
		err := rows.Scan(
			&d.EntryType,
			&d.Category,
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

func findEntryDefaultSubEntries(tx *sql.Tx, ctx context.Context, find service.EntryDefaultFinder) ([]*service.EntryDefault, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "entry_default_sub_entries.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			entry_default_sub_entries.name,
			sub_types.name
		FROM entry_default_sub_entries
		LEFT JOIN entry_types ON entry_default_sub_entries.entry_type_id = entry_types.id
		LEFT JOIN sub_entry_types ON entry_default_sub_entries.sub_entry_type_id = sub_entry_types.id
		LEFT JOIN entry_types AS sub_types ON sub_entry_types.sub_id = sub_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.EntryDefault, 0)
	for rows.Next() {
		d := &service.EntryDefault{
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

func findEntryDefaultProperties(tx *sql.Tx, ctx context.Context, find service.EntryDefaultFinder) ([]*service.EntryDefault, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "entry_default_properties.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			entry_default_properties.name,
			entry_default_properties.type,
			entry_default_properties.value
		FROM entry_default_properties
		LEFT JOIN entry_types ON entry_default_properties.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.EntryDefault, 0)
	for rows.Next() {
		d := &service.EntryDefault{
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

func findEntryDefaultEnvirons(tx *sql.Tx, ctx context.Context, find service.EntryDefaultFinder) ([]*service.EntryDefault, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "entry_default_environs.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			entry_default_environs.name,
			entry_default_environs.type,
			entry_default_environs.value
		FROM entry_default_environs
		LEFT JOIN entry_types ON entry_default_environs.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*service.EntryDefault, 0)
	for rows.Next() {
		d := &service.EntryDefault{
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

func AddEntryDefault(db *sql.DB, ctx context.Context, d *service.EntryDefault) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch d.Category {
	case "sub_entry":
		err = addEntryDefaultSubEntry(tx, ctx, d)
		if err != nil {
			return err
		}
	case "property":
		err = addEntryDefaultProperty(tx, ctx, d)
		if err != nil {
			return err
		}
	case "environ":
		err = addEntryDefaultEnviron(tx, ctx, d)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid category for entry default: %v", d.Category)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEntryDefault(tx *sql.Tx, ctx context.Context, d *service.EntryDefault) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO entry_defaults (
			entry_type_id,
			category,
			name,
			type,
			value
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		typeID,
		d.Category,
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

func addEntryDefaultSubEntry(tx *sql.Tx, ctx context.Context, d *service.EntryDefault) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	subTypeID, err := getSubEntryTypeID(tx, ctx, d.EntryType, d.Type)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO entry_default_sub_entries (
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

func addEntryDefaultProperty(tx *sql.Tx, ctx context.Context, d *service.EntryDefault) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO entry_default_properties (
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

func addEntryDefaultEnviron(tx *sql.Tx, ctx context.Context, d *service.EntryDefault) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO entry_default_environs (
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

func UpdateEntryDefault(db *sql.DB, ctx context.Context, upd service.EntryDefaultUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch upd.Category {
	case "sub_entry":
		err := updateEntryDefaultSubEntry(tx, ctx, upd)
		if err != nil {
			return err
		}
	case "property":
		err := updateEntryDefaultProperty(tx, ctx, upd)
		if err != nil {
			return err
		}
	case "environ":
		err := updateEntryDefaultEnviron(tx, ctx, upd)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid category for entry default: %v", upd.Category)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateEntryDefault(tx *sql.Tx, ctx context.Context, upd service.EntryDefaultUpdater) error {
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
		return fmt.Errorf("need at least one field to update entry default: %v %v %v", upd.EntryType, upd.Category, upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Category, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE entry_defaults
		SET `+strings.Join(keys, ", ")+`
		WHERE entry_type_id=? AND category=? AND name=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	return nil
}

func updateEntryDefaultSubEntry(tx *sql.Tx, ctx context.Context, upd service.EntryDefaultUpdater) error {
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
		return fmt.Errorf("need at least one field to update entry default: %v %v %v", upd.EntryType, "sub_entry", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE entry_default_sub_entries
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

func updateEntryDefaultProperty(tx *sql.Tx, ctx context.Context, upd service.EntryDefaultUpdater) error {
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
		return fmt.Errorf("need at least one field to update entry default: %v %v %v", upd.EntryType, "property", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE entry_default_properties
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

func updateEntryDefaultEnviron(tx *sql.Tx, ctx context.Context, upd service.EntryDefaultUpdater) error {
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
		return fmt.Errorf("need at least one field to update entry default: %v %v %v", upd.EntryType, "environ", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE entry_default_environs
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

func DeleteEntryDefault(db *sql.DB, ctx context.Context, entryType, ctg, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	switch ctg {
	case "sub_entry":
		err := deleteEntryDefaultSubEntry(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	case "property":
		err := deleteEntryDefaultProperty(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	case "environ":
		err := deleteEntryDefaultEnviron(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid category for entry default: %v", ctg)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteEntryDefault(tx *sql.Tx, ctx context.Context, entryType, ctg, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM entry_defaults
		WHERE entry_type_id=? AND category=? AND name=?
	`,
		typeID,
		ctg,
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
		return service.NotFound("no such entry default for entry type %v: %v %v", entryType, ctg, name)
	}
	return nil
}

func deleteEntryDefaultSubEntry(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM entry_default_sub_entries
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
		return service.NotFound("no such entry default for entry type %v: %v %v", entryType, "sub_entry", name)
	}
	return nil
}

func deleteEntryDefaultProperty(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM entry_default_properties
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
		return service.NotFound("no such entry default for entry type %v: %v %v", entryType, "property", name)
	}
	return nil
}

func deleteEntryDefaultEnviron(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM entry_default_environs
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
		return service.NotFound("no such entry default for entry type %v: %v %v", entryType, "environ", name)
	}
	return nil
}
