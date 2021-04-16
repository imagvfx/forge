package sqlite

import (
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
	return err
}

func FindLogs(db *sql.DB, user string, find service.LogFinder) ([]*service.Log, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, err = getEntry(tx, user, find.EntryID)
	if err != nil {
		return nil, err
	}
	props, err := findLogs(tx, user, find)
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
func findLogs(tx *sql.Tx, user string, find service.LogFinder) ([]*service.Log, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "entry_id=?")
	vals = append(vals, find.EntryID)
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.Query(`
		SELECT
			id,
			entry_id,
			user,
			action,
			ctg,
			name,
			typ,
			val,
			time
		FROM logs
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
			&l.EntryID,
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

func addLog(tx *sql.Tx, l *service.Log) error {
	result, err := tx.Exec(`
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
		l.EntryID,
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
