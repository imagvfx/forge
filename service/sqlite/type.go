package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/imagvfx/forge/service"
)

func createEntryTypesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entry_types (
			id INTEGER PRIMARY KEY,
			name STRING NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_entry_types_name ON entry_types (name)`)
	return err
}

func addRootEntryType(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO entry_types
			(name)
		VALUES
			(?)
	`,
		"root",
	)
	if err != nil {
		return err
	}
	return nil
}

func EntryTypes(db *sql.DB, ctx context.Context) ([]string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	names, err := findEntryTypes(tx, ctx)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return names, nil
}

func findEntryTypes(tx *sql.Tx, ctx context.Context) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			name
		FROM entry_types
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	typs := make([]string, 0)
	for rows.Next() {
		var t string
		err := rows.Scan(
			&t,
		)
		if err != nil {
			return nil, err
		}
		typs = append(typs, t)
	}
	return typs, nil
}

func getEntryTypeID(tx *sql.Tx, ctx context.Context, name string) (int, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM entry_types WHERE name=?", name)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	if !rows.Next() {
		return -1, service.NotFound("entry type not found: %v", name)
	}
	var id int
	err = rows.Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func AddEntryType(db *sql.DB, ctx context.Context, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEntryType(tx, ctx, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEntryType(tx *sql.Tx, ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("entry type name not specified")
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO entry_types (
			name
		)
		VALUES (?)
	`,
		name,
	)
	if err != nil {
		return err
	}
	return nil
}

func RenameEntryType(db *sql.DB, ctx context.Context, name, newName string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = renameEntryType(tx, ctx, name, newName)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func renameEntryType(tx *sql.Tx, ctx context.Context, name, newName string) error {
	if name == "" {
		return fmt.Errorf("entry type name not specified")
	}
	if newName == "" {
		return fmt.Errorf("new name of entry type not specified: %v", newName)
	}
	result, err := tx.ExecContext(ctx, `
		UPDATE entry_types
		SET name=?
		WHERE name=?
	`,
		newName,
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
		return fmt.Errorf("no entry type affected")
	}
	return nil
}

func DeleteEntryType(db *sql.DB, ctx context.Context, name string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteEntryType(tx, ctx, name)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteEntryType(tx *sql.Tx, ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("entry type name not specified")
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM entry_types
		WHERE name=?
	`,
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
		return fmt.Errorf("no entry type affected")
	}
	return nil
}
