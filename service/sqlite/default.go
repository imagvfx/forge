package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

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

func createDefaultAccessesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS default_accesses (
			id INTEGER PRIMARY KEY,
			entry_type_id INTEGER NOT NULL,
			accessor_id INTEGER NOT NULL,
			mode INTEGER NOT NULL,
			FOREIGN KEY (entry_type_id) REFERENCES entry_types (id),
			FOREIGN KEY (accessor_id) REFERENCES accessors (id),
			UNIQUE (entry_type_id, accessor_id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_default_accesses_entry_type_id ON default_accesses (entry_type_id)`)
	return err
}

func createDefaultSubEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS default_sub_entries (
			id INTEGER PRIMARY KEY,
			entry_type_id INTEGER NOT NULL,
			name STRING NOT NULL,
			sub_entry_type_id INTEGER NOT NULL,
			value STRING NOT NULL,
			FOREIGN KEY (entry_type_id) REFERENCES entry_types (id),
			FOREIGN KEY (sub_entry_type_id) REFERENCES entry_types (id),
			UNIQUE (entry_type_id, name)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_default_sub_entries_entry_type_id ON default_sub_entries (entry_type_id)`)
	return err
}

func FindDefaults(db *sql.DB, ctx context.Context, find forge.DefaultFinder) ([]*forge.Default, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	defaults := make([]*forge.Default, 0)
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
	accs, err := findDefaultAccesses(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, accs...)
	subs, err := findDefaultSubEntries(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	defaults = append(defaults, subs...)
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return defaults, nil
}

func findDefaultProperties(tx *sql.Tx, ctx context.Context, find forge.DefaultFinder) ([]*forge.Default, error) {
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
			default_properties.id,
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
	defaults := make([]*forge.Default, 0)
	for rows.Next() {
		d := &forge.Default{
			Category: "property",
		}
		err := rows.Scan(
			&d.ID,
			&d.EntryType,
			&d.Name,
			&d.Type,
			&d.Value,
		)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(d.Name, ".") {
			_, d.Value, _ = evalSpecialProperty(tx, ctx, d.Name, d.Value)
			// TODO: what should I do when there was an evaluation error?
		}
		defaults = append(defaults, d)
	}
	return defaults, nil
}

func getDefaultProperty(tx *sql.Tx, ctx context.Context, entry_type, name string) (*forge.Default, error) {
	find := forge.DefaultFinder{
		EntryType: &entry_type,
		Name:      &name,
	}
	defaults, err := findDefaultProperties(tx, ctx, find)
	if len(defaults) == 0 {
		return nil, forge.NotFound("default not found: %v: %v", entry_type, name)
	}
	return defaults[0], err
}

func findDefaultEnvirons(tx *sql.Tx, ctx context.Context, find forge.DefaultFinder) ([]*forge.Default, error) {
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
	defaults := make([]*forge.Default, 0)
	for rows.Next() {
		d := &forge.Default{
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

func getDefaultEnviron(tx *sql.Tx, ctx context.Context, entry_type, name string) (*forge.Default, error) {
	find := forge.DefaultFinder{
		EntryType: &entry_type,
		Name:      &name,
	}
	defaults, err := findDefaultEnvirons(tx, ctx, find)
	if len(defaults) == 0 {
		return nil, forge.NotFound("default not found: %v: %v", entry_type, name)
	}
	return defaults[0], err
}

func findDefaultAccesses(tx *sql.Tx, ctx context.Context, find forge.DefaultFinder) ([]*forge.Default, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "default_accesses.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			default_accesses.accessor_id,
			default_accesses.mode
		FROM default_accesses
		LEFT JOIN entry_types ON default_accesses.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*forge.Default, 0)
	for rows.Next() {
		var acID int
		var mode int
		d := &forge.Default{
			Category: "access",
		}
		err := rows.Scan(
			&d.EntryType,
			&acID,
			&mode,
		)
		if err != nil {
			return nil, err
		}
		ac, err := getAccessorByID(tx, ctx, acID)
		if err != nil {
			return nil, err
		}
		d.Name = ac.Name
		d.Type = "user"
		if ac.IsGroup {
			d.Type = "group"
		}
		d.Value = "r"
		if mode == 1 {
			d.Value = "rw"
		}
		defaults = append(defaults, d)
	}
	return defaults, nil
}

func getDefaultAccess(tx *sql.Tx, ctx context.Context, entry_type, name string) (*forge.Default, error) {
	find := forge.DefaultFinder{
		EntryType: &entry_type,
		Name:      &name,
	}
	defaults, err := findDefaultAccesses(tx, ctx, find)
	if len(defaults) == 0 {
		return nil, forge.NotFound("default not found: %v: %v", entry_type, name)
	}
	return defaults[0], err
}

func findDefaultSubEntries(tx *sql.Tx, ctx context.Context, find forge.DefaultFinder) ([]*forge.Default, error) {
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
			sub_entry_types.name,
			default_sub_entries.value
		FROM default_sub_entries
		LEFT JOIN entry_types ON default_sub_entries.entry_type_id = entry_types.id
		LEFT JOIN entry_types AS sub_entry_types ON default_sub_entries.sub_entry_type_id = sub_entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	defaults := make([]*forge.Default, 0)
	for rows.Next() {
		d := &forge.Default{
			Category: "sub_entry",
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

func AddDefault(db *sql.DB, ctx context.Context, d *forge.Default) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := forge.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return forge.Unauthorized("user doesn't have permission to add default: %v", user)
	}
	switch d.Category {
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
	case "access":
		err = addDefaultAccess(tx, ctx, d)
		if err != nil {
			return err
		}
	case "sub_entry":
		err = addDefaultSubEntry(tx, ctx, d)
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

func addDefaultProperty(tx *sql.Tx, ctx context.Context, d *forge.Default) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	d.Value, err = validateProperty(tx, ctx, "", d.Name, d.Type, d.Value)
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
	_, err = tx.ExecContext(ctx, `
		INSERT INTO properties (
			entry_id,
			default_id,
			val,
			updated_at
		) SELECT id, ?, ?, ? FROM entries WHERE type_id=?
		ON CONFLICT DO NOTHING
	`,
		d.ID, d.Value, time.Now().UTC(), typeID,
	)
	if err != nil {
		return err
	}
	return nil
}

func addDefaultEnviron(tx *sql.Tx, ctx context.Context, d *forge.Default) error {
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
	_, err = tx.ExecContext(ctx, `
		INSERT INTO environs (
			entry_id,
			name,
			typ,
			val,
			updated_at
		) SELECT id, ?, ?, ?, ? FROM entries WHERE type_id=?
		ON CONFLICT DO NOTHING
	`,
		d.Name, d.Type, d.Value, time.Now().UTC(), typeID,
	)
	if err != nil {
		return err
	}
	return nil
}

func addDefaultAccess(tx *sql.Tx, ctx context.Context, d *forge.Default) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	if d.Type != "user" && d.Type != "group" {
		return fmt.Errorf("invalid default access type (want 'user' or 'group'): %v", d.Type)
	}
	if d.Value != "r" && d.Value != "rw" {
		return fmt.Errorf("invalid default access value (want 'r' or 'rw'): %v", d.Value)
	}
	mode := 0
	if d.Value == "rw" {
		mode = 1
	}
	ac, err := getAccessor(tx, ctx, d.Name)
	if err != nil {
		return fmt.Errorf("invalid accessor name: %v", d.Name)
	}
	acType := "user"
	if ac.IsGroup {
		acType = "group"
	}
	if d.Type != acType {
		return fmt.Errorf("mismatch accessor type: got %v, want %v", d.Type, acType)
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO default_accesses (
			entry_type_id,
			accessor_id,
			mode
		)
		VALUES (?, ?, ?)
	`,
		typeID,
		ac.ID,
		mode,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	d.ID = int(id)
	// Things can be chaotic if we change pre-existing accesses for entry.
	// Let's not do that.
	return nil
}

func addDefaultSubEntry(tx *sql.Tx, ctx context.Context, d *forge.Default) error {
	typeID, err := getEntryTypeID(tx, ctx, d.EntryType)
	if err != nil {
		return err
	}
	subTypeID, err := getEntryTypeID(tx, ctx, d.Type)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO default_sub_entries (
			entry_type_id,
			name,
			sub_entry_type_id,
			value
		)
		VALUES (?, ?, ?, ?)
	`,
		typeID,
		d.Name,
		subTypeID,
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
	// TODO: should I add sub_entry when a default created?
	return nil
}

func UpdateDefault(db *sql.DB, ctx context.Context, upd forge.DefaultUpdater) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := forge.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return forge.Unauthorized("user doesn't have permission to update default: %v", user)
	}
	switch upd.Category {
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
	case "access":
		err := updateDefaultAccess(tx, ctx, upd)
		if err != nil {
			return err
		}
	case "sub_entry":
		err := updateDefaultSubEntry(tx, ctx, upd)
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

func updateDefaultProperty(tx *sql.Tx, ctx context.Context, upd forge.DefaultUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Type != nil {
		keys = append(keys, "type=?")
		vals = append(vals, *upd.Type)
	}
	d, err := getDefaultProperty(tx, ctx, upd.EntryType, upd.Name)
	if err != nil {
		return err
	}
	if upd.Value != nil {
		typ := d.Type
		if upd.Type != nil {
			typ = *upd.Type
		}
		*upd.Value, err = validateProperty(tx, ctx, "", d.Name, typ, *upd.Value)
		if err != nil {
			return err
		}
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
	// For value, we will only update properties having old default value with the new one.
	if d.Value != *upd.Value {
		_, err := tx.ExecContext(ctx, `
		UPDATE properties
		SET val = ?
		WHERE
			val = ? AND
			id IN (
				SELECT properties.id FROM properties
				LEFT JOIN entries ON properties.entry_id = entries.id
				WHERE entries.type_id=? AND properties.name=?
			)
	`,
			*upd.Value, d.Value, typeID, upd.Name,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateDefaultEnviron(tx *sql.Tx, ctx context.Context, upd forge.DefaultUpdater) error {
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
	old, err := getDefaultEnviron(tx, ctx, upd.EntryType, upd.Name)
	if err != nil {
		return err
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
	// Update existing entries.
	if old.Type != *upd.Type {
		_, err := tx.ExecContext(ctx, `
		UPDATE environs
		SET typ = ?
		WHERE
			id IN (
				SELECT environs.id FROM environs
				LEFT JOIN entries ON environs.entry_id = entries.id
				WHERE entries.type_id=? AND environs.name=?
			)
	`,
			*upd.Type, typeID, upd.Name,
		)
		if err != nil {
			return err
		}
	}
	// For value, we will only update environs having old default value with the new one.
	if old.Value != *upd.Value {
		_, err := tx.ExecContext(ctx, `
		UPDATE environs
		SET val = ?
		WHERE
			val = ? AND
			id IN (
				SELECT environs.id FROM environs
				LEFT JOIN entries ON environs.entry_id = entries.id
				WHERE entries.type_id=? AND environs.name=?
			)
	`,
			*upd.Value, old.Value, typeID, upd.Name,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateDefaultAccess(tx *sql.Tx, ctx context.Context, upd forge.DefaultUpdater) error {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if upd.Type != nil {
		return fmt.Errorf("cannot change default accessor type")
	}
	if upd.Value != nil {
		if *upd.Value != "r" && *upd.Value != "rw" {
			return fmt.Errorf("invalid default access value (want 'r' or 'rw'): %v", upd.Type)
		}
		mode := 0
		if *upd.Value == "rw" {
			mode = 1
		}
		keys = append(keys, "value=?")
		vals = append(vals, mode)
	}
	if len(keys) == 0 {
		return fmt.Errorf("need at least one field to update default: %v %v %v", upd.EntryType, "access", upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE default_accesses
		SET `+strings.Join(keys, ", ")+`
		WHERE entry_type_id=? AND name=?
	`,
		vals...,
	)
	if err != nil {
		return err
	}
	// Things can be chaotic if we change pre-existing accesses on entry.
	// Let's not do that.
	return nil
}

func updateDefaultSubEntry(tx *sql.Tx, ctx context.Context, upd forge.DefaultUpdater) error {
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
		keys = append(keys, "value=?")
		vals = append(vals, *upd.Value)
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
	// Will not update sub-entries of the entries unlike properties or environs.
	// As sub-entries are more complicated then others and hard to update correctly.
	return nil
}

func DeleteDefault(db *sql.DB, ctx context.Context, entryType, ctg, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	user := forge.UserNameFromContext(ctx)
	yes, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !yes {
		return forge.Unauthorized("user doesn't have permission to delete default: %v", user)
	}
	switch ctg {
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
	case "access":
		err := deleteDefaultAccess(tx, ctx, entryType, name)
		if err != nil {
			return err
		}
	case "sub_entry":
		err := deleteDefaultSubEntry(tx, ctx, entryType, name)
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
		return forge.NotFound("no such default for entry type %v: %v %v", entryType, "property", name)
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
		return forge.NotFound("no such default for entry type %v: %v %v", entryType, "environ", name)
	}
	return nil
}

func deleteDefaultAccess(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	ac, err := getAccessor(tx, ctx, name)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM default_accesses
		WHERE entry_type_id=? AND accessor_id=?
	`,
		typeID,
		ac.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return forge.NotFound("no such default for entry type %v: %v %v", entryType, "access", name)
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
		return forge.NotFound("no such default for entry type %v: %v %v", entryType, "sub_entry", name)
	}
	return nil
}
