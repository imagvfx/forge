package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/imagvfx/forge"
)

func createEntriesTable(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER,
			path STRING NOT NULL UNIQUE,
			type_id INTEGER NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES entries (id),
			FOREIGN KEY (type_id) REFERENCES entry_types (id)
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

func FindEntries(db *sql.DB, ctx context.Context, find forge.EntryFinder) ([]*forge.Entry, error) {
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
func findEntries(tx *sql.Tx, ctx context.Context, find forge.EntryFinder) ([]*forge.Entry, error) {
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
	ents := make([]*forge.Entry, 0)
	for rows.Next() {
		e := &forge.Entry{}
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
	return ents, nil
}

func SearchEntries(db *sql.DB, ctx context.Context, search forge.EntrySearcher) ([]*forge.Entry, error) {
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

func searchEntries(tx *sql.Tx, ctx context.Context, search forge.EntrySearcher) ([]*forge.Entry, error) {
	if search.SearchRoot == "/" {
		// Prevent search root become two slashes by adding slash again.
		search.SearchRoot = ""
	}
	keywords := make([]string, 0, len(search.Keywords))
	for _, kwd := range search.Keywords {
		kwd = strings.TrimSpace(kwd)
		if kwd == "" {
			continue
		}
		keywords = append(keywords, kwd)
	}
	if len(keywords) == 0 {
		// at least one loop needed
		keywords = append(keywords, "")
	}
	queries := make([]string, 0)
	queryTmpl := `
		SELECT
			entries.id,
			entries.path,
			entry_types.name,
			thumbnails.id
		FROM properties
		LEFT JOIN entries ON entries.id = properties.entry_id
		LEFT JOIN thumbnails ON entries.id = thumbnails.entry_id
		LEFT JOIN entry_types ON entries.type_id = entry_types.id
		%s
		GROUP BY entries.id
	`
	// vals will contain info for entire queries.
	vals := make([]interface{}, 0)
	for _, kwd := range keywords {
		keys := make([]string, 0)
		// redundant, but needed in every loop for INTERSECT
		keys = append(keys, "entries.path LIKE ?")
		vals = append(vals, search.SearchRoot+`/%`)
		if search.EntryType != "" {
			keys = append(keys, "entry_types.name=?")
			vals = append(vals, search.EntryType)
		}
		if kwd != "" {
			idxColon := strings.Index(kwd, ":")
			idxEqual := strings.Index(kwd, "=")
			if idxColon == -1 && idxEqual == -1 {
				// generic search; not tied to a property
				keys = append(keys, `
					(entries.path LIKE ? OR
						(properties.name NOT LIKE '.%' AND
							(
								(properties.typ!='user' AND properties.val LIKE ?) OR
								(properties.typ='user' AND properties.id IN
									(SELECT properties.id FROM properties LEFT JOIN accessors ON properties.val=accessors.id
										WHERE properties.typ='user' AND (accessors.called LIKE ? OR accessors.name LIKE ?)
									)
								)
							)
						)
					)
				`)
				kwdl := `%` + kwd + `%`
				pathl := search.SearchRoot + `/%` + kwd
				if strings.HasSuffix(kwd, "/") {
					pathl += "%"
				}
				vals = append(vals, pathl, kwdl, kwdl, kwdl)
			} else {
				// Check which is appeared earlier.
				if idxColon == -1 {
					idxColon = len(kwd)
				}
				if idxEqual == -1 {
					idxEqual = len(kwd)
				}
				idx := idxColon
				exactSearch := false
				if idxEqual < idxColon {
					idx = idxEqual
					exactSearch = true
				}
				k := kwd[:idx]
				v := kwd[idx+1:] // exclude colon or equal
				// NOTE: The line with 'properties.val in (?)' is weird in a look, but it was the only query I can think of
				// that checks empty 'user' properties when a user searches it. (eg. not assigned entries).
				eq := " = "
				if !exactSearch {
					eq = " LIKE "
				}
				q := fmt.Sprintf(`
					(properties.name=? AND
						(
							(properties.typ!='user' AND properties.val %s ?) OR
							(properties.typ='user' AND properties.val='' AND properties.val in (?)) OR
							(properties.typ='user' AND properties.id IN
								(SELECT properties.id FROM properties LEFT JOIN accessors ON properties.val=accessors.id
									WHERE properties.typ='user' AND (accessors.called %s ? OR accessors.name %s ?)
								)
							)
						)
					)
				`, eq, eq, eq)
				keys = append(keys, q)
				vl := v
				if !exactSearch {
					vl = "%" + v + "%"
				}
				vals = append(vals, k, vl, v, vl, vl)

			}
		}
		where := ""
		if len(keys) != 0 {
			where = "WHERE " + strings.Join(keys, " AND ")
		}
		query := fmt.Sprintf(queryTmpl, where)
		queries = append(queries, query)
	}
	rows, err := tx.QueryContext(ctx, strings.Join(queries, " INTERSECT "), vals...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ents := make([]*forge.Entry, 0)
	for rows.Next() {
		e := &forge.Entry{}
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
		SELECT COUNT(*) FROM entries WHERE path LIKE ?`,
		path+"/%",
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
	ents, err := findEntries(tx, ctx, forge.EntryFinder{Path: &path})
	if err != nil {
		return nil, err
	}
	if len(ents) == 0 {
		return nil, forge.NotFound("entry not found: %v", path)
	}
	return ents[0], nil
}

func getEntryByID(tx *sql.Tx, ctx context.Context, id int) (*forge.Entry, error) {
	ents, err := findEntries(tx, ctx, forge.EntryFinder{ID: &id})
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
		defAccs, err := findDefaultAccesses(tx, ctx, forge.DefaultFinder{EntryType: &entType})
		if err != nil {
			return err
		}
		for _, d := range defAccs {
			if !seenAcc[d.Name] {
				dacc := &forge.AccessControl{
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
				upd := forge.AccessControlUpdater{
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
	user := forge.UserNameFromContext(ctx)
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
	err = updateEntryPath(tx, ctx, path, newPath)
	if err != nil {
		return err
	}
	// Let's log only for the entry (not for sub entries).
	// This might be changed in the future.
	user := forge.UserNameFromContext(ctx)
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
	subEnts, err := findEntries(tx, ctx, forge.EntryFinder{ParentPath: &path})
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
