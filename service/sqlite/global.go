package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createGlobalsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS globals (
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
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_globals_entry_type_id ON globals (entry_type_id)`)
	return err
}

func FindGlobals(db *sql.DB, ctx context.Context, find service.GlobalFinder) ([]*service.Global, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	gls, err := findGlobals(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	return gls, nil
}

func findGlobals(tx *sql.Tx, ctx context.Context, find service.GlobalFinder) ([]*service.Global, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.EntryType != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.EntryType)
	}
	if find.Name != nil {
		keys = append(keys, "globals.name=?")
		vals = append(vals, *find.Name)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entry_types.name,
			globals.name,
			globals.type,
			globals.value
		FROM globals
		LEFT JOIN entry_types ON globals.entry_type_id = entry_types.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	globals := make([]*service.Global, 0)
	for rows.Next() {
		g := &service.Global{}
		err := rows.Scan(
			&g.EntryType,
			&g.Name,
			&g.Type,
			&g.Value,
		)
		if err != nil {
			return nil, err
		}
		globals = append(globals, g)
	}
	return globals, nil
}

func GetGlobal(db *sql.DB, ctx context.Context, entryType, name string) (*service.Global, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	g, err := getGlobal(tx, ctx, entryType, name)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func getGlobal(tx *sql.Tx, ctx context.Context, entryType, name string) (*service.Global, error) {
	find := service.GlobalFinder{
		EntryType: &entryType,
		Name:      &name,
	}
	globals, err := findGlobals(tx, ctx, find)
	if len(globals) == 0 {
		return nil, service.NotFound("global not found on %v: %v", entryType, name)
	}
	return globals[0], err
}

func AddGlobal(db *sql.DB, ctx context.Context, g *service.Global) error {
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
		return service.Unauthorized("user doesn't have permission to add global: %v", user)
	}
	err = addGlobal(tx, ctx, g)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addGlobal(tx *sql.Tx, ctx context.Context, g *service.Global) error {
	typeID, err := getEntryTypeID(tx, ctx, g.EntryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO globals (
			entry_type_id,
			name,
			type,
			value
		)
		VALUES (?, ?, ?, ?)
	`,
		typeID,
		g.Name,
		g.Type,
		g.Value,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	g.ID = int(id)
	return nil
}

func UpdateGlobal(db *sql.DB, ctx context.Context, upd service.GlobalUpdater) error {
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
		return service.Unauthorized("user doesn't have permission to update global: %v", user)
	}
	updateGlobal(tx, ctx, upd)
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func updateGlobal(tx *sql.Tx, ctx context.Context, upd service.GlobalUpdater) error {
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
		return fmt.Errorf("need at least one field to update global: %v %v", upd.EntryType, upd.Name)
	}
	typeID, err := getEntryTypeID(tx, ctx, upd.EntryType)
	if err != nil {
		return err
	}
	vals = append(vals, typeID, upd.Name) // for where clause
	_, err = tx.ExecContext(ctx, `
		UPDATE globals
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

func DeleteGlobal(db *sql.DB, ctx context.Context, entryType, name string) error {
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
	err = deleteGlobal(tx, ctx, entryType, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteGlobal(tx *sql.Tx, ctx context.Context, entryType, name string) error {
	typeID, err := getEntryTypeID(tx, ctx, entryType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM globals
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
		return service.NotFound("no such global for entry type %v: %v", entryType, name)
	}
	return nil
}
