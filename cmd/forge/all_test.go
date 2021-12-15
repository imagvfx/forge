package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service/sqlite"
)

// testDB returns a sql db for a test, caller is responsible for close when the process ends.
func testDB(t *testing.T) (*sql.DB, error) {
	tempDir := t.TempDir()
	tempDB := filepath.Join(tempDir, "temp.db")
	return sqlite.Open(tempDB)
}

// testServer returns a forge server for test.
func testServer(t *testing.T, db *sql.DB) (*forge.Server, error) {
	err := sqlite.Init(db)
	if err != nil {
		return nil, err
	}
	svc := sqlite.NewService(db)
	cfg := &forge.Config{}
	server := forge.NewServer(svc, cfg)
	return server, nil
}
