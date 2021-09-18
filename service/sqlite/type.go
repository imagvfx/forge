package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

// TODO: it should get EntryTypeFinder as an argument.
func FindEntryTypes(db *sql.DB, ctx context.Context) ([]string, error) {
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

func FindBaseEntryTypes(db *sql.DB, ctx context.Context) ([]string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	names, err := findBaseEntryTypes(tx, ctx)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return names, nil
}

func findBaseEntryTypes(tx *sql.Tx, ctx context.Context) ([]string, error) {
	ts, err := findEntryTypes(tx, ctx)
	if err != nil {
		return nil, err
	}
	origTs := make([]string, 0, len(ts))
	for _, t := range ts {
		if !strings.Contains(t, ".") {
			origTs = append(origTs, t)
		}
	}
	return origTs, nil
}

func FindOverrideEntryTypes(db *sql.DB, ctx context.Context) ([]string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	names, err := findOverrideEntryTypes(tx, ctx)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return names, nil
}

func findOverrideEntryTypes(tx *sql.Tx, ctx context.Context) ([]string, error) {
	ts, err := findEntryTypes(tx, ctx)
	if err != nil {
		return nil, err
	}
	overTs := make([]string, 0, len(ts))
	for _, t := range ts {
		if strings.Contains(t, ".") {
			overTs = append(overTs, t)
		}
	}
	return overTs, nil
}

func getEntryTypeByID(tx *sql.Tx, ctx context.Context, id int) (string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT name FROM entry_types WHERE id=?", id)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", service.NotFound("entry type not found with id: %v", id)
	}
	var name string
	err = rows.Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
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
	origType := strings.Split(name, ".")[0]
	if name != origType {
		allTypes, err := findEntryTypes(tx, ctx)
		if err != nil {
			return err
		}
		found := false
		for _, t := range allTypes {
			if t == origType {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("not found original entry type of the override entry type: %v", name)
		}
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
	origType := strings.Split(name, ".")[0]
	if name != origType {
		newOrigType := strings.Split(newName, ".")[0]
		if origType != newOrigType {
			return fmt.Errorf("new override entry type name should not change it's original entry type name")
		}
		allTypes, err := findEntryTypes(tx, ctx)
		if err != nil {
			return err
		}
		found := false
		for _, t := range allTypes {
			if t == origType {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("not found original entry type of the override entry type: %v", name)
		}
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
