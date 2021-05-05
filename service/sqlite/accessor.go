package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createAccessorsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS accessors (
			id INTEGER PRIMARY KEY,
			is_group BOOL NOT NULL,
			name STRING NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_accessors_name ON accessors (name)`)
	return err
}

func getAccessor(tx *sql.Tx, ctx context.Context, name string) (*service.Accessor, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
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
			name
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
		return nil, service.NotFound("accessor not found: %v", name)
	}
	a := &service.Accessor{}
	err = rows.Scan(
		&a.ID,
		&a.IsGroup,
		&a.Name,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}
