package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/imagvfx/forge/service"
)

func createEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			path STRING NOT NULL UNIQUE,
			typ STRING NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES entries (id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_entries_path ON entries (path)`)
	return err
}

func addRootEntry(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO entries
			(id, path, typ)
		VALUES
			(?, ?, ?)
	`,
		0, "/", "root",
	)
	if err != nil {
		return err
	}
	return nil
}

func FindEntries(db *sql.DB, ctx context.Context, find service.EntryFinder) ([]*service.Entry, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ents, err := findEntries(tx, ctx, find)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ents, nil
}

// when id is empty, it will find entries of root.
func findEntries(tx *sql.Tx, ctx context.Context, find service.EntryFinder) ([]*service.Entry, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	if find.Path != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.Path)
	}
	if find.ParentPath != nil {
		keys = append(keys, "parents.path=?")
		vals = append(vals, *find.ParentPath)
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entries.id,
			entries.path,
			entries.typ,
			thumbnails.id
		FROM entries
		LEFT JOIN entries AS parents ON entries.parent_id = parents.id
		LEFT JOIN thumbnails ON entries.id = thumbnails.entry_id
		`+where+`
		ORDER BY entries.id ASC
	`,
		vals...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ents := make([]*service.Entry, 0)
	for rows.Next() {
		e := &service.Entry{}
		var thumbID *int
		err := rows.Scan(
			&e.ID,
			&e.Path,
			&e.Type,
			&thumbID,
		)
		if err != nil {
			return nil, err
		}
		if thumbID != nil {
			e.HasThumbnail = true
		}
		err = userRead(tx, ctx, e.Path)
		if err != nil {
			var e *service.NotFoundError
			if !errors.As(err, &e) {
				return nil, err
			}
			// userRead returns service.NotFoundError
			// because of the user doesn't have permission to see the entry.
			continue
		}
		ents = append(ents, e)
	}
	return ents, nil
}

func GetEntry(db *sql.DB, ctx context.Context, path string) (*service.Entry, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ent, err := getEntry(tx, ctx, path)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ent, nil
}

func getEntry(tx *sql.Tx, ctx context.Context, path string) (*service.Entry, error) {
	ents, err := findEntries(tx, ctx, service.EntryFinder{Path: &path})
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, service.NotFound("entry not found")
	}
	return ents[0], nil
}

func getEntryID(tx *sql.Tx, ctx context.Context, path string) (int, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM entries WHERE path=?", path)
	if err != nil {
		return -1, err
	}
	defer rows.Close()
	if !rows.Next() {
		return -1, service.NotFound("entry not found: %v", path)
	}
	var id int
	err = rows.Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func AddEntry(db *sql.DB, ctx context.Context, e *service.Entry, props []*service.Property, envs []*service.Property) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEntry(tx, ctx, e)
	if err != nil {
		return err
	}
	for _, p := range props {
		err := addProperty(tx, ctx, p)
		if err != nil {
			return err
		}
	}
	for _, env := range envs {
		err := addEnviron(tx, ctx, env)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEntry(tx *sql.Tx, ctx context.Context, e *service.Entry) error {
	if e.Path == "" {
		return fmt.Errorf("path unspecified")
	}
	if e.Path == "/" {
		return fmt.Errorf("cannot create root path")
	}
	if !strings.HasPrefix(e.Path, "/") {
		return fmt.Errorf("path is not started with /")
	}
	parent := filepath.Dir(e.Path)
	err := userWrite(tx, ctx, parent)
	if err != nil {
		return err
	}
	p, err := getEntry(tx, ctx, parent)
	if err != nil {
		return err
	}
	result, err := tx.Exec(`
		INSERT INTO entries (
			path,
			typ,
			parent_id
		)
		VALUES (?, ?, ?)
	`,
		e.Path,
		e.Type,
		p.ID,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = int(id)
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryPath: e.Path,
		User:      user,
		Action:    "create",
		Category:  "entry",
		Name:      e.Path,
		Type:      e.Type,
	})
	if err != nil {
		return err
	}
	return nil
}

func RenameEntry(db *sql.DB, ctx context.Context, path, newName string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = renameEntry(tx, ctx, path, newName)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func renameEntry(tx *sql.Tx, ctx context.Context, path, newName string) error {
	// Rename an entry actually affects many sub entries,
	// should be picky.
	if path == "" {
		return fmt.Errorf("need a path for rename")
	}
	if path == "/" {
		return fmt.Errorf("cannot rename root entry")
	}
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("entry path shouldn't end with /")
	}
	if newName == "" {
		return fmt.Errorf("need a new name for rename")
	}
	if strings.Contains(newName, "/") {
		return fmt.Errorf("entry name cannot have '/' in it")
	}
	base := filepath.Base(path)
	if newName == base {
		return nil
	}
	newPath := filepath.Dir(path) + "/" + newName
	err := userWrite(tx, ctx, filepath.Dir(path))
	if err != nil {
		return err
	}
	err = updateEntryPath(tx, ctx, path, newPath)
	if err != nil {
		return err
	}
	// Let's log only for the entry (not for sub entries).
	// This might be changed in the future.
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
		EntryPath: path,
		User:      user,
		Action:    "rename",
		Category:  "entry",
		Name:      newName,
	})
	if err != nil {
		return err
	}
	// root entry successfully renamed,
	// let's do it for all sub entries.
	like := path + `/%`
	rows, err := tx.QueryContext(ctx, `
		SELECT
			path
		FROM entries
		WHERE path LIKE ?
	`,
		like,
	)
	if err != nil {
		return err
	}
	subEnts := make([]string, 0)
	defer rows.Close()
	for rows.Next() {
		var path string
		err := rows.Scan(
			&path,
		)
		if err != nil {
			return err
		}
		subEnts = append(subEnts, path)
	}
	for _, subEntPath := range subEnts {
		newSubEntPath := strings.Replace(subEntPath, path, newPath, 1)
		err := updateEntryPath(tx, ctx, subEntPath, newSubEntPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateEntryPath(tx *sql.Tx, ctx context.Context, path, newPath string) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE entries
		SET path=?
		WHERE path=?
	`,
		newPath,
		path,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 property affected, got %v", n)
	}
	return nil
}

func DeleteEntry(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteEntry(tx *sql.Tx, ctx context.Context, path string) error {
	// Rename an entry actually affects many sub entries,
	// should be picky.
	if path == "" {
		return fmt.Errorf("need a path to delete")
	}
	if path == "/" {
		return fmt.Errorf("cannot delete root entry")
	}
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("entry path shouldn't end with /")
	}
	// The entry that will be deleted shouldn't have sub entries.
	like := path + `/%`
	rows, err := tx.QueryContext(ctx, `
		SELECT
			path
		FROM entries
		WHERE path LIKE ?
	`,
		like,
	)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("entry shouldn't have sub entries: %v", path)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	e, err := getEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	err = userWrite(tx, ctx, filepath.Dir(path))
	if err != nil {
		return err
	}
	relatedTables := []string{"thumbnails", "properties", "environs", "access_controls", "logs"}
	for _, table := range relatedTables {
		stmt := fmt.Sprintf(`
			DELETE FROM %v
			WHERE entry_id=?
		`, table)
		_, err := tx.ExecContext(ctx, stmt,
			e.ID,
		)
		if err != nil {
			return err
		}
	}
	result, err := tx.Exec(`
		DELETE FROM entries
		WHERE id=?
	`,
		e.ID,
	)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("want 1 property affected, got %v", n)
	}
	return nil
}
