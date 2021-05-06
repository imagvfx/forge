package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createLogsTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY,
			entry_id INTEGER,
			user STRING NOT NULL,
			action STRING NOT NULL,
			ctg STRING NOT NULL,
			name STRING NOT NULL,
			typ STRING NOT NULL,
			val STRING NOT NULL,
			time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (entry_id) REFERENCES entries (id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_logs_entry_id ON logs (entry_id)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_logs_user ON logs (user)`)
	return err
}

func FindLogs(db *sql.DB, ctx context.Context, find service.LogFinder) ([]*service.Log, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = getEntry(tx, ctx, find.EntryPath)
	if err != nil {
		return nil, err
	}
	props, err := findLogs(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return props, nil
}

// when id is empty, it will find logs of root.
func findLogs(tx *sql.Tx, ctx context.Context, find service.LogFinder) ([]*service.Log, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "entries.path=?")
	vals = append(vals, find.EntryPath)
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			logs.id,
			entries.path,
			logs.user,
			logs.action,
			logs.ctg,
			logs.name,
			logs.typ,
			logs.val,
			logs.time
		FROM logs
		LEFT JOIN entries ON logs.entry_id = entries.id
		`+where,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	logs := make([]*service.Log, 0)
	for rows.Next() {
		l := &service.Log{}
		err := rows.Scan(
			&l.ID,
			&l.EntryPath,
			&l.User,
			&l.Action,
			&l.Category,
			&l.Name,
			&l.Type,
			&l.Value,
			&l.When,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func addLog(tx *sql.Tx, ctx context.Context, l *service.Log) error {
	entryID, err := getEntryID(tx, ctx, l.EntryPath)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO logs (
			entry_id,
			user,
			action,
			ctg,
			name,
			typ,
			val
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		entryID,
		l.User,
		l.Action,
		l.Category,
		l.Name,
		l.Type,
		l.Value,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	l.ID = int(id)
	return nil
}
