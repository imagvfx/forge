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
			type_id INT NOT NULL,
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
			(id, path, type_id)
		VALUES
			(?, ?, ?)
	`,
		1, "/", 1, // sqlite IDs are 1 based
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
	where := ""
	if len(keys) != 0 {
		where = "WHERE " + strings.Join(keys, " AND ")
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			entries.id,
			entries.path,
			entry_types.name,
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

func SearchEntries(db *sql.DB, ctx context.Context, search service.EntrySearcher) ([]*service.Entry, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	ents, err := searchEntries(tx, ctx, search)
	if err != nil {
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return ents, nil
}

func searchEntries(tx *sql.Tx, ctx context.Context, search service.EntrySearcher) ([]*service.Entry, error) {
	keys := make([]string, 0)
	vals := make([]interface{}, 0)
	keys = append(keys, "entries.path LIKE ?")
	vals = append(vals, search.SearchRoot+`/%`)
	if search.EntryType != "" {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, search.EntryType)
	}
	for _, p := range search.Keywords {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		idxColon := strings.Index(p, ":")
		idxEqual := strings.Index(p, "=")
		if idxColon == -1 && idxEqual == -1 {
			// generic search; not tied to a property
			keys = append(keys, "(entries.path LIKE ? OR properties.val LIKE ?)")
			vals = append(vals, search.SearchRoot+`/%`+p+`%`, `%`+p+`%`)
			continue
		}
		// Check which is appeared earlier.
		if idxColon == -1 {
			idxColon = len(p)
		}
		if idxEqual == -1 {
			idxEqual = len(p)
		}
		idx := idxColon
		exactSearch := false
		if idxEqual < idxColon {
			idx = idxEqual
			exactSearch = true
		}
		k := p[:idx]
		v := p[idx+1:] // exclude colon or equal
		if exactSearch {
			keys = append(keys, "(properties.name=? AND properties.val=?)")
			vals = append(vals, k, v)
		} else {
			keys = append(keys, "(properties.name=? AND properties.val LIKE ?)")
			vals = append(vals, k, `%`+v+`%`)
		}
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
			thumbnails.id
		FROM entries
		LEFT JOIN thumbnails ON entries.id = thumbnails.entry_id
		LEFT JOIN properties ON entries.id = properties.entry_id
		LEFT JOIN entry_types ON entries.type_id = entry_types.id
		`+where+`
		GROUP BY entries.id
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

func getEntryByID(tx *sql.Tx, ctx context.Context, id int) (*service.Entry, error) {
	ents, err := findEntries(tx, ctx, service.EntryFinder{ID: &id})
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

func AddEntry(db *sql.DB, ctx context.Context, e *service.Entry) error {
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

func addEntryR(tx *sql.Tx, ctx context.Context, e *service.Entry) error {
	if e.Path != "/" {
		// Check and apply the type if it is predefined sub entry of the parent.
		parentPath := filepath.Dir(e.Path)
		entName := filepath.Base(e.Path)
		predefined, err := getProperty(tx, ctx, parentPath, ".predefined_sub_entries")
		if err != nil {
			var e *service.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
		}
		if predefined != nil {
			predefinedType := ""
			for _, sub := range strings.Split(predefined.Value, ",") {
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
	}
	err := addEntry(tx, ctx, e)
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
		defProps, err := findDefaultProperties(tx, ctx, service.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defProps {
			if !seenProp[d.Name] {
				dp := &service.Property{
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
				upd := service.PropertyUpdater{
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
		defEnvs, err := findDefaultEnvirons(tx, ctx, service.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defEnvs {
			if !seenEnv[d.Name] {
				denv := &service.Property{
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
				upd := service.PropertyUpdater{
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
		defAccs, err := findDefaultAccesses(tx, ctx, service.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defAccs {
			if !seenAcc[d.Name] {
				dacc := &service.AccessControl{
					EntryPath:    e.Path,
					Accessor:     d.Name,
					AccessorType: d.Type,
					Mode:         d.Value,
				}
				err := addAccessControl(tx, ctx, dacc)
				if err != nil {
					return err
				}
				seenAcc[d.Name] = true
			} else {
				upd := service.AccessControlUpdater{
					EntryPath: e.Path,
					Accessor:  d.Name,
					Mode:      &d.Value,
				}
				err := updateAccessControl(tx, ctx, upd)
				if err != nil {
					return err
				}
			}
		}
	}
	defSubs, err := findDefaultSubEntries(tx, ctx, service.DefaultFinder{EntryType: &e.Type})
	if err != nil {
		return err
	}
	for _, d := range defSubs {
		de := &service.Entry{
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
			parent_id
		)
		VALUES (?, ?, ?)
	`,
		e.Path,
		typeID,
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
	err = updateEntryPath(tx, ctx, path, newPath)
	if err != nil {
		return err
	}
	// Let's log only for the entry (not for sub entries).
	// This might be changed in the future.
	user := service.UserNameFromContext(ctx)
	err = addLog(tx, ctx, &service.Log{
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
