package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/imagvfx/forge/service"
)

func createSubEntryTypesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS sub_entry_types (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NOT NULL,
			sub_id INTRGER NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES entry_types (id),
			FOREIGN KEY (sub_id) REFERENCES entry_types (id),
			UNIQUE (parent_id, sub_id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_sub_entry_types_parent_id ON sub_entry_types (parent_id)`)
	return err
}

func SubEntryTypes(db *sql.DB, ctx context.Context, parentType string) ([]string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	names, err := findSubEntryTypes(tx, ctx, parentType)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return names, nil
}

func findSubEntryTypes(tx *sql.Tx, ctx context.Context, parentType string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			subs.name
		FROM sub_entry_types
		LEFT JOIN entry_types AS parents ON sub_entry_types.parent_id = parents.id
		LEFT JOIN entry_types AS subs ON sub_entry_types.sub_id = subs.id
		WHERE parents.name=?`,
		parentType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subTypes := make([]string, 0)
	for rows.Next() {
		var t string
		err := rows.Scan(
			&t,
		)
		if err != nil {
			return nil, err
		}
		subTypes = append(subTypes, t)
	}
	return subTypes, nil
}

func getSubEntryTypeID(tx *sql.Tx, ctx context.Context, parentType, subType string) (int, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT
			sub_entry_types.id
		FROM sub_entry_types
		LEFT JOIN entry_types AS parents ON sub_entry_types.parent_id = parents.id
		LEFT JOIN entry_types AS subs ON sub_entry_types.sub_id = subs.id
		WHERE parents.name=? AND subs.name=?`,
		parentType, subType,
	)
	if err != nil {
		return -1, err
	}
	if !rows.Next() {
		return -1, fmt.Errorf("sub entry type not found")
	}
	var id int
	err = rows.Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func AddSubEntryType(db *sql.DB, ctx context.Context, parentType, subType string) error {
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
		return service.Unauthorized("user doesn't have permission to add sub-entry type: %v", user)
	}
	err = addSubEntryType(tx, ctx, parentType, subType)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addSubEntryType(tx *sql.Tx, ctx context.Context, parentType, subType string) error {
	parentTypeID, err := getEntryTypeID(tx, ctx, parentType)
	if err != nil {
		return err
	}
	subTypeID, err := getEntryTypeID(tx, ctx, subType)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO sub_entry_types (
			parent_id,
			sub_id
		)
		VALUES (?, ?)
	`,
		parentTypeID,
		subTypeID,
	)
	if err != nil {
		return err
	}
	return nil
}

func DeleteSubEntryType(db *sql.DB, ctx context.Context, parentType, subType string) error {
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
		return service.Unauthorized("user doesn't have permission to delete sub-entry type: %v", user)
	}
	err = deleteSubEntryType(tx, ctx, parentType, subType)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteSubEntryType(tx *sql.Tx, ctx context.Context, parentType, subType string) error {
	parentTypeID, err := getEntryTypeID(tx, ctx, parentType)
	if err != nil {
		return err
	}
	subTypeID, err := getEntryTypeID(tx, ctx, subType)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM sub_entry_types
		WHERE parent_id=? AND sub_id=?
	`,
		parentTypeID,
		subTypeID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no sub entry type affected")
	}
	return nil
}
