package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/imagvfx/forge"
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

func FindLogs(db *sql.DB, ctx context.Context, find forge.LogFinder) ([]*forge.Log, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
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
func findLogs(tx *sql.Tx, ctx context.Context, find forge.LogFinder) ([]*forge.Log, error) {
	keys := make([]string, 0)
	vals := make([]any, 0)
	if find.EntryPath != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.EntryPath)
	}
	if find.Category != nil {
		keys = append(keys, "logs.ctg=?")
		vals = append(vals, *find.Category)
	}
	if find.Name != nil {
		keys = append(keys, "logs.name=?")
		vals = append(vals, *find.Name)
	}
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
	logs := make([]*forge.Log, 0)
	for rows.Next() {
		l := &forge.Log{}
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

func GetLogs(db *sql.DB, ctx context.Context, path, ctg, name string) ([]*forge.Log, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	log, err := getLogs(tx, ctx, path, ctg, name)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return log, nil
}

func getLogs(tx *sql.Tx, ctx context.Context, path, ctg, name string) ([]*forge.Log, error) {
	logs, err := findLogs(tx, ctx, forge.LogFinder{
		EntryPath: &path,
		Category:  &ctg,
		Name:      &name,
	})
	for _, l := range logs {
		p := &forge.Property{
			Type:     l.Type,
			RawValue: l.Value,
		}
		evalProperty(tx, ctx, p)
		if p.ValueError != nil {
			l.Value = "eval error: " + p.RawValue
			continue
		}
		l.Value = p.Value
	}
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, forge.NotFound("log not found")
	}
	return logs, nil
}

func addLog(tx *sql.Tx, ctx context.Context, l *forge.Log) error {
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
