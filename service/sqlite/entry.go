package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

func createEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			path TEXT NOT NULL UNIQUE,
			type_id INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL,
			archived BOOLEAN NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES entries (id),
			FOREIGN KEY (type_id) REFERENCES entry_types (id)
		)
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE entries ADD COLUMN archived NOT NULL DEFAULT false`)
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	_, err = tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_entries_path ON entries (path)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS index_entries_archived ON entries (archived)`)
	return err
}

func addRootEntry(tx *sql.Tx) error {
	_, err := tx.Exec(`
		INSERT OR IGNORE INTO entries
			(id, path, type_id, created_at, archived)
		VALUES
			(?, ?, ?, ?, ?)
	`,
		1, "/", 1, time.Now().UTC(), false, // sqlite IDs are 1 based
	)
	if err != nil {
		return err
	}
	return nil
}

func FindEntries(db *sql.DB, ctx context.Context, find forge.EntryFinder) ([]*forge.Entry, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return nil, forge.Unauthorized("context user unspecified")
	}
	archived, err := getUserSettingShowArchived(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	find.Archived = archived
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
func findEntries(tx *sql.Tx, ctx context.Context, find forge.EntryFinder) ([]*forge.Entry, error) {
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return nil, forge.Unauthorized("context user unspecified")
	}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if !find.Archived {
		keys = append(keys, "NOT entries.archived")
	}
	if find.ID != nil {
		keys = append(keys, "entries.id=?")
		vals = append(vals, *find.ID)
	}
	if find.Path != nil {
		keys = append(keys, "entries.path=?")
		vals = append(vals, *find.Path)
	}
	if find.ParentPath != nil {
		keys = append(keys, "parents.path=?")
		vals = append(vals, *find.ParentPath)
	}
	if find.AncestorPath != nil {
		keys = append(keys, "entries.path GLOB ? || '/*'")
		vals = append(vals, *find.AncestorPath)
	}
	if find.ChildPath != nil {
		if *find.ChildPath != "/" {
			keys = append(keys, "(? GLOB entries.path || '/*' OR entries.path='/')")
			vals = append(vals, *find.ChildPath)
		} else {
			// no entry is parent of root
			keys = append(keys, "FALSE")
		}
	}
	if find.Types != nil {
		qs := []string{}
		for _, typ := range find.Types {
			qs = append(qs, "?")
			vals = append(vals, typ)
		}
		keys = append(keys, "entry_types.name IN ("+strings.Join(qs, ", ")+")")
	}
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entries.id,
			entries.path,
			entry_types.name,
			entries.archived,
			entries.created_at,
			(SELECT time FROM logs WHERE logs.entry_id=entries.id ORDER BY id DESC LIMIT 1),
			thumbnails.id
		FROM entries
		LEFT JOIN entries AS parents ON entries.parent_id = parents.id
		LEFT JOIN entry_types ON entries.type_id = entry_types.id
		LEFT JOIN thumbnails ON entries.id = thumbnails.entry_id
		`+where+`
		ORDER BY entries.id ASC
	`,
		vals...,
	)
	if err != nil {
		return nil, fmt.Errorf("find entries: %w", err)
	}
	defer rows.Close()
	ents := make([]*forge.Entry, 0)
	for rows.Next() {
		e := &forge.Entry{}
		created := Time{}
		updated := sql.NullTime{}
		var thumbID *int
		err := rows.Scan(
			&e.ID,
			&e.Path,
			&e.Type,
			&e.Archived,
			&created,
			&updated,
			&thumbID,
		)
		if err != nil {
			return nil, err
		}
		e.CreatedAt = time.Time(created)
		e.UpdatedAt = updated.Time
		if !updated.Valid {
			e.UpdatedAt = e.CreatedAt
		}
		if thumbID != nil {
			e.HasThumbnail = true
		}
		err = userRead(tx, ctx, e.Path)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return nil, err
			}
			// userRead returns forge.NotFoundError
			// because of the user doesn't have permission to see the entry.
			continue
		}
		ents = append(ents, e)
	}
	for _, e := range ents {
		e.Property = make(map[string]*forge.Property)
		props, err := entryProperties(tx, ctx, e.Path)
		if err != nil {
			return nil, err
		}
		for _, p := range props {
			e.Property[p.Name] = p
		}
	}
	return ents, nil
}

func CountAllSubEntries(db *sql.DB, ctx context.Context, path string) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	n, err := countAllSubEntries(tx, ctx, path)
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return n, nil
}

func countAllSubEntries(tx *sql.Tx, ctx context.Context, path string) (int, error) {
	if path == "/" {
		path = ""
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT COUNT(*) FROM entries WHERE path GLOB ?`,
		path+"/*",
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
	}
	var n int
	err = rows.Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func GetEntry(db *sql.DB, ctx context.Context, path string) (*forge.Entry, error) {
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

func getEntry(tx *sql.Tx, ctx context.Context, path string) (*forge.Entry, error) {
	ents, err := findEntries(tx, ctx, forge.EntryFinder{Path: &path, Archived: true})
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, forge.NotFound("entry not found: %v", path)
	}
	return ents[0], nil
}

func getEntryByID(tx *sql.Tx, ctx context.Context, id int) (*forge.Entry, error) {
	ents, err := findEntries(tx, ctx, forge.EntryFinder{ID: &id, Archived: true})
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, forge.NotFound("entry not found: %v", id)
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
		return -1, forge.NotFound("entry not found: %v", path)
	}
	var id int
	err = rows.Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func getEntryType(tx *sql.Tx, ctx context.Context, path string) (string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT entry_types.name
		FROM entries
		LEFT JOIN entry_types ON entries.type_id=entry_types.id
		WHERE entries.path=?`,
		path,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", forge.NotFound("entry not found: %v", path)
	}
	var typ string
	err = rows.Scan(&typ)
	if err != nil {
		return "", err
	}
	return typ, nil
}

func AddEntry(db *sql.DB, ctx context.Context, e *forge.Entry) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = addEntryR(tx, ctx, e)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func addEntryR(tx *sql.Tx, ctx context.Context, e *forge.Entry) error {
	e.Path = path.Clean(e.Path)
	if e.Path == "/" {
		return fmt.Errorf("root entry cannot be created or deleted by user")
	}
	e.Path = strings.TrimSuffix(e.Path, "/")
	// Check and apply the type if it is predefined sub entry of the parent.
	parentPath := filepath.Dir(e.Path)
	entName := filepath.Base(e.Path)
	validChars := strings.Join([]string{
		"abcdefghijklmnopqrstuvwxyz",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"0123456789",
		"_-/",
	}, "")
	for _, r := range entName {
		if !strings.ContainsRune(validChars, r) {
			return fmt.Errorf("entry name has invalid character '%v': %v", string(r), e.Path)
		}
	}
	parent, err := getEntry(tx, ctx, parentPath)
	if err != nil {
		return fmt.Errorf("check parent: %v", err)
	}
	_, err = getEntry(tx, ctx, e.Path)
	if err == nil {
		return fmt.Errorf("entry exists: %v", e.Path)
	}
	if e.Type == "" {
		// '.sub_entry_types' property should have only one sub entry type to fill the type.
		subTypes, err := getProperty(tx, ctx, parentPath, ".sub_entry_types")
		if err != nil {
			var e *forge.NotFoundError
			if errors.As(err, &e) {
				return fmt.Errorf("cannot guess entry type: '.sub_entry_types' property not exist on entry: %v", parentPath)
			} else {
				return err
			}
		}
		toks := strings.Split(subTypes.Value, ",")
		if len(toks) != 1 {
			return fmt.Errorf("cannot guess entry type: multiple sub entry types defined on entry: %v", parentPath)
		}
		firstType := strings.TrimSpace(toks[0])
		if firstType == "" {
			return fmt.Errorf("cannot guess entry type: no sub entry type defined on entry: %v", parentPath)
		}
		e.Type = firstType
	}
	predefinedValue := ""
	predefined, err := getProperty(tx, ctx, parent.Path, ".predefined_sub_entries")
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return err
		}
		// find in the globals
		predefinedGlobal, err := getGlobal(tx, ctx, parent.Type, "predefined_sub_entries")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
		} else {
			predefinedValue = predefinedGlobal.Value
		}
	} else {
		predefinedValue = predefined.Value
	}
	if predefinedValue != "" {
		predefinedType := ""
		for _, sub := range strings.Split(predefinedValue, ",") {
			sub = strings.TrimSpace(sub)
			toks := strings.Split(sub, ":")
			if len(toks) != 2 {
				// It's an error, but let's just continue.
				continue
			}
			subName := strings.TrimSpace(toks[0])
			subType := strings.TrimSpace(toks[1])
			if subName == "*" || subName == entName {
				// Star (*) is catch all name.
				predefinedType = subType
				break
			}
		}
		if predefinedType != "" {
			baseType := strings.Split(predefinedType, ".")[0]
			if e.Type != baseType {
				return fmt.Errorf("cannot create predefined sub entry %v as type %v, should be %v", entName, e.Type, baseType)
			}
			e.Type = predefinedType
		}
	}
	err = addEntry(tx, ctx, e)
	if err != nil {
		return err
	}
	entTypes := make([]string, 0)
	if strings.Contains(e.Type, ".") {
		baseType := strings.Split(e.Type, ".")[0]
		entTypes = append(entTypes, baseType, e.Type)
	} else {
		entTypes = append(entTypes, e.Type)
	}
	seenProp := make(map[string]bool)
	seenEnv := make(map[string]bool)
	seenAcc := make(map[string]bool)
	for _, entType := range entTypes {
		defProps, err := findDefaultProperties(tx, ctx, forge.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defProps {
			if !seenProp[d.Name] {
				dp := &forge.Property{
					EntryPath: e.Path,
					Name:      d.Name,
					Type:      d.Type,
					Value:     d.Value,
				}
				err := addProperty(tx, ctx, dp)
				if err != nil {
					return err
				}
				seenProp[d.Name] = true
			} else {
				upd := forge.PropertyUpdater{
					EntryPath: e.Path,
					Name:      d.Name,
					Value:     &d.Value,
				}
				err := updateProperty(tx, ctx, upd)
				if err != nil {
					return err
				}
			}
		}
		defEnvs, err := findDefaultEnvirons(tx, ctx, forge.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defEnvs {
			if !seenEnv[d.Name] {
				denv := &forge.Property{
					EntryPath: e.Path,
					Name:      d.Name,
					Type:      d.Type,
					Value:     d.Value,
				}
				err := addEnviron(tx, ctx, denv)
				if err != nil {
					return err
				}
				seenEnv[d.Name] = true
			} else {
				upd := forge.PropertyUpdater{
					EntryPath: e.Path,
					Name:      d.Name,
					Value:     &d.Value,
				}
				err := updateEnviron(tx, ctx, upd)
				if err != nil {
					return err
				}
			}
		}
		defAccs, err := findDefaultAccessList(tx, ctx, forge.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defAccs {
			if !seenAcc[d.Name] {
				dacc := &forge.Access{
					EntryPath: e.Path,
					Name:      d.Name,
					Type:      d.Type,
					Value:     d.Value,
				}
				err := addAccess(tx, ctx, dacc)
				if err != nil {
					return err
				}
				seenAcc[d.Name] = true
			} else {
				upd := forge.AccessUpdater{
					EntryPath: e.Path,
					Name:      d.Name,
					Value:     &d.Value,
				}
				err := updateAccess(tx, ctx, upd)
				if err != nil {
					return err
				}
			}
		}
	}
	defSubs, err := findDefaultSubEntries(tx, ctx, forge.DefaultFinder{EntryType: &e.Type})
	if err != nil {
		return err
	}
	for _, d := range defSubs {
		de := &forge.Entry{
			Path: filepath.Join(e.Path, d.Name),
			Type: d.Type,
		}
		err = addEntryR(tx, ctx, de)
		if err != nil {
			return err
		}
	}
	return nil
}

func addEntry(tx *sql.Tx, ctx context.Context, e *forge.Entry) error {
	if e.Path == "" {
		return fmt.Errorf("path unspecified")
	}
	if e.Path == "/" {
		return fmt.Errorf("cannot create root path")
	}
	if !strings.HasPrefix(e.Path, "/") {
		return fmt.Errorf("path is not started with /")
	}
	baseType := strings.Split(e.Type, ".")[0]
	typeID, err := getEntryTypeID(tx, ctx, baseType)
	if err != nil {
		return err
	}
	parent := filepath.Dir(e.Path)
	err = userWrite(tx, ctx, parent)
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
			type_id,
			parent_id,
			created_at,
			archived
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		e.Path,
		typeID,
		p.ID,
		time.Now().UTC(),
		false,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = int(id)
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
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
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("entry path should be started with /")
	}
	if strings.HasSuffix(path, "/") {
		return fmt.Errorf("entry path shouldn't be ended with /")
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
	parent := filepath.Dir(path)
	if path == "/" {
		// the result is slash(/), should be empty string.
		parent = ""
	}
	err := userWrite(tx, ctx, parent)
	if err != nil {
		return err
	}
	newPath := filepath.Join(parent, newName)
	_, err = getEntry(tx, ctx, newPath)
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return err
		}
	} else {
		return fmt.Errorf("rename target path already exists: %v", newPath)
	}
	err = updateEntryPath(tx, ctx, path, newPath)
	if err != nil {
		return err
	}
	// Let's log only for the entry (not for sub entries).
	// This might be changed in the future.
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err = addLog(tx, ctx, &forge.Log{
		EntryPath: newPath,
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
	like := path + `/*`
	rows, err := tx.QueryContext(ctx, `
		SELECT
			path
		FROM entries
		WHERE path GLOB ?
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

func ArchiveEntry(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = archiveEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func archiveEntry(tx *sql.Tx, ctx context.Context, path string) error {
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	toks := strings.Split(path, "/")
	if len(toks) != 2 {
		return fmt.Errorf("archive support only for root branches: %v", path)
	}
	if toks[0] != "" || toks[1] == "" {
		return fmt.Errorf("archive support only for root branches: %v", path)
	}
	_, err = getEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE entries
		SET archived=1
		WHERE path=? OR path GLOB ?
	`,
		path,
		path+"/*",
	)
	if err != nil {
		return err
	}
	return nil
}

func UnarchiveEntry(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = unarchiveEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func unarchiveEntry(tx *sql.Tx, ctx context.Context, path string) error {
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return forge.Unauthorized("context user unspecified")
	}
	err := userWrite(tx, ctx, path)
	if err != nil {
		return err
	}
	toks := strings.Split(path, "/")
	if len(toks) != 2 {
		return fmt.Errorf("unarchive applies only for root branches: %v", path)
	}
	if toks[0] != "" || toks[1] == "" {
		return fmt.Errorf("unarchive support only for root branches: %v", path)
	}
	_, err = getEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE entries
		SET archived=0
		WHERE path=? OR path GLOB ?
	`,
		path,
		path+"/*",
	)
	if err != nil {
		return err
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
	// Delete an entry actually affects many sub entries,
	// should be picky.
	if path == "" {
		return fmt.Errorf("need a path to delete")
	}
	if path == "/" {
		return fmt.Errorf("cannot delete root entry")
	}
	// The entry that will be deleted shouldn't have sub entries.
	like := path + `/*`
	rows, err := tx.QueryContext(ctx, `
		SELECT
			path
		FROM entries
		WHERE path GLOB ?
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

func DeleteEntryRecursive(db *sql.DB, ctx context.Context, path string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = deleteEntryR(tx, ctx, path)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func deleteEntryR(tx *sql.Tx, ctx context.Context, path string) error {
	subEnts, err := findEntries(tx, ctx, forge.EntryFinder{ParentPath: &path, Archived: true})
	if err != nil {
		return err
	}
	for _, ent := range subEnts {
		err := deleteEntryR(tx, ctx, ent.Path)
		if err != nil {
			return err
		}
	}
	err = deleteEntry(tx, ctx, path)
	if err != nil {
		return err
	}
	return nil
}
