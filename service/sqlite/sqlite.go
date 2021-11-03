package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// Open opens a db at path.
// It returns an error if it cannot open the db.
func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("db path required")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// Enable Write-Ahead Logging. See https://sqlite.org/wal.html
	if _, err := db.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	// Enable strict mode.
	if _, err := db.Exec(`PRAGMA strict = ON;`); err != nil {
		return nil, fmt.Errorf("strict mode: %w", err)
	}
	// Enable foreign keys.
	if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("foreign keys: %w", err)
	}
	return db, nil
}

// Init initialize the db.
// It is ok to initialize the db multiple times.
func Init(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	fmt.Println("create entries table!")
	err = createEntryTypesTable(tx)
	if err != nil {
		return err
	}
	err = addRootEntryType(tx)
	if err != nil {
		return err
	}
	err = createDefaultSubEntriesTable(tx)
	if err != nil {
		return err
	}
	err = createDefaultPropertiesTable(tx)
	if err != nil {
		return err
	}
	err = createDefaultEnvironsTable(tx)
	if err != nil {
		return err
	}
	err = createDefaultAccessesTable(tx)
	if err != nil {
		return err
	}
	err = createGlobalsTable(tx)
	if err != nil {
		return err
	}
	err = createEntriesTable(tx)
	if err != nil {
		return err
	}
	err = addRootEntry(tx)
	if err != nil {
		return err
	}
	err = createThumbnailsTable(tx)
	if err != nil {
		return err
	}
	err = createPropertiesTable(tx)
	if err != nil {
		return err
	}
	err = createEnvironsTable(tx)
	if err != nil {
		return err
	}
	err = createAccessControlsTable(tx)
	if err != nil {
		return err
	}
	err = createLogsTable(tx)
	if err != nil {
		return err
	}
	err = createAccessorsTable(tx) // for user and group
	if err != nil {
		return err
	}
	err = createUserSettingsTable(tx)
	if err != nil {
		return err
	}
	err = addEveryoneGroup(tx)
	if err != nil {
		return err
	}
	err = addAdminGroup(tx)
	if err != nil {
		return err
	}
	err = createGroupMembersTable(tx)
	if err != nil {
		return err
	}
	return tx.Commit()
}
