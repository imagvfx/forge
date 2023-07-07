package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/imagvfx/forge"
)

func createAccessorsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS accessors (
			id INTEGER PRIMARY KEY,
			is_group BOOL NOT NULL,
			name STRING NOT NULL UNIQUE,
			called STRING NOT NULL,
			disabled BOOL NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_accessors_name ON accessors (name)`)
	return err
}

func getAccessor(tx *sql.Tx, ctx context.Context, name string) (*forge.Accessor, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	keys = append(keys, "name=?")
	vals = append(vals, name)
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			is_group,
			name,
			called,
			disabled
		FROM accessors
		`+where+`
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, forge.NotFound("accessor not found: %v", name)
	}
	a := &forge.Accessor{}
	err = rows.Scan(
		&a.ID,
		&a.IsGroup,
		&a.Name,
		&a.Called,
		&a.Disabled,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func getAccessorByID(tx *sql.Tx, ctx context.Context, id int) (*forge.Accessor, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	keys = append(keys, "id=?")
	vals = append(vals, id)
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			is_group,
			name,
			called,
			disabled
		FROM accessors
		`+where+`
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, forge.NotFound("accessor not found: %v", id)
	}
	a := &forge.Accessor{}
	err = rows.Scan(
		&a.ID,
		&a.IsGroup,
		&a.Name,
		&a.Called,
		&a.Disabled,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}
