package main

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service/sqlite"
)

// testDB returns a sql db and a server for a test, caller is responsible for close when the process ends.
func testDB(t *testing.T) (*sql.DB, *forge.Server, error) {
	tempDir := t.TempDir()
	tempDB := filepath.Join(tempDir, "temp.db")
	db, err := sqlite.Open(tempDB)
	if err != nil {
		return nil, nil, err
	}
	err = sqlite.Init(db)
	if err != nil {
		return nil, nil, err
	}
	svc := sqlite.NewService(db)
	cfg := &forge.Config{}
	server := forge.NewServer(svc, cfg)
	return db, server, nil
}
