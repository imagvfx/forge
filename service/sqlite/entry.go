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
			path STRING NOT NULL UNIQUE,
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
	showArchived, err := getUserSettingShowArchived(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	admin, err := isAdmin(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0)
	vals := make([]any, 0)
	if !admin || !showArchived {
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
	if find.ChildPath != nil {
		if *find.ChildPath != "/" {
			keys = append(keys, "(? GLOB entries.path || '/*') OR (entries.path='/')")
			vals = append(vals, *find.ChildPath)
		} else {
			// no entry is parent of root
			keys = append(keys, "FALSE")
		}
	}
	if find.Type != nil {
		keys = append(keys, "entry_types.name=?")
		vals = append(vals, *find.Type)
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

type where struct {
	Key     string
	Val     string
	Exact   bool
	Exclude bool
}

func (w where) Equal() string {
	if w.Exact {
		return "="
	}
	return "LIKE"
}

func (w where) Not() string {
	if !w.Exclude {
		return ""
	}
	if w.Exact {
		return "!"
	}
	return "NOT "
}

func (w where) Value() string {
	if w.Exact {
		return w.Val
	}
	if w.Val == "" {
		return w.Val
	}
	return `%` + w.Val + `%`
}

func searchEntries(tx *sql.Tx, ctx context.Context, search forge.EntrySearcher) ([]*forge.Entry, error) {
	user := forge.UserNameFromContext(ctx)
	if user == "" {
		return nil, forge.Unauthorized("context user unspecified")
	}
	showArchived, err := getUserSettingShowArchived(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	admin, err := isAdmin(tx, ctx, user)
	if err != nil {
		return nil, err
	}
	if search.SearchRoot == "/" {
		// Prevent search root become two slashes by adding slash again.
		search.SearchRoot = ""
	}
	var (
		whPath *where
		whName *where
		whType *where
	)
	wheres := make([]where, 0, len(search.Keywords))
	for _, kwd := range search.Keywords {
		kwd = strings.TrimSpace(kwd)
		if kwd == "" {
			continue
		}
		wh := where{}
		idxColon := strings.Index(kwd, ":")
		idxEqual := strings.Index(kwd, "=")
		if idxColon == -1 && idxEqual == -1 {
			// Generic search
			wh.Val = kwd
			wheres = append(wheres, wh)
			continue
		}
		if idxColon == -1 {
			idxColon = len(kwd)
		}
		if idxEqual == -1 {
			idxEqual = len(kwd)
		}
		idx := idxColon
		if idxEqual < idxColon {
			idx = idxEqual
			wh.Exact = true
		}
		k := kwd[:idx]
		if len(k) != 0 && k[len(k)-1] == '!' {
			k = k[:len(k)-1]
			wh.Exclude = true
		}
		if k == "" {
			// Invalid search. Having ':' or '=' without the keyword.
			continue
		}
		wh.Key = k
		wh.Val = kwd[idx+1:] // exclude colon or equal
		// special keywords those aren't actual properties.
		// multiple queries on special keywords aren't supported yet and will pick up the last one.
		if k == "path" {
			whPath = &wh
			continue
		}
		if k == "name" {
			whName = &wh
			continue
		}
		if k == "type" {
			whType = &wh
			continue
		}
		wheres = append(wheres, wh)
	}
	if len(wheres) == 0 && whPath == nil && whName == nil && whType == nil {
		return nil, nil
	}
	innerQueries := make([]string, 0)
	// vals will contain info for entire queries.
	innerVals := make([]any, 0)
	for _, wh := range wheres {
		key := wh.Key
		rawval := wh.Val
		val := wh.Value()
		eq := wh.Equal()
		findParent := false
		innerKeys := make([]string, 0)
		if wh.Key == "" {
			// Generic search. Not tied to a property.
			innerKeys = append(innerKeys, `
				(entries.path LIKE ? OR
					(default_properties.name NOT LIKE '.%' AND
						(
							(default_properties.type!='user' AND properties.val LIKE ?) OR
							(default_properties.type='user' AND properties.id IN
								(SELECT properties.id FROM properties
									LEFT JOIN accessors ON properties.val=accessors.id
									LEFT JOIN default_properties ON properties.default_id=default_properties.id
									WHERE default_properties.type='user' AND (accessors.called LIKE ? OR accessors.name LIKE ?)
								)
							)
						)
					)
				)
			`)
			pathl := search.SearchRoot + `/%` + rawval
			if strings.HasSuffix(pathl, "/") {
				pathl += `%`
			}
			innerVals = append(innerVals, pathl, val, val, val)
		} else {
			sub := ""
			toks := strings.SplitN(key, ".", 2)
			if len(toks) == 2 {
				sub = toks[0]
				key = toks[1]
			}
			q := fmt.Sprintf("(default_properties.name=? AND ")
			innerVals = append(innerVals, key)
			if sub != "" {
				findParent = true
				if sub != "(sub)" {
					q += "entries.path LIKE ? AND"
					innerVals = append(innerVals, "%/"+sub)
				}
			}
			not := ""
			if wh.Exclude {
				not = "NOT"
			}
			q += " " + not + " ("
			vs := strings.Split(rawval, ",")
			for i, v := range vs {
				if i != 0 {
					q += " OR "
				}
				vl := v
				if !wh.Exact {
					vl = "%" + v + "%"
				}
				tagGlob := "'*" + v + "*'"
				if wh.Exact {
					tagGlob = "'*' || char(10) || '" + v + "' || char(10) || '*'"
				}
				userWhere := ""
				whereVals := make([]any, 0)
				if v != "" {
					userWhere = fmt.Sprintf("(accessors.called %s ? OR accessors.name %s ?)", eq, eq)
					whereVals = append(whereVals, vl, vl)
				} else {
					userWhere = "(accessors.id IS NULL)"
					if !wh.Exact {
						userWhere = "TRUE"
					}
				}
				vq := fmt.Sprintf(`
					(
						(default_properties.type!='tag' AND default_properties.type!='user' AND properties.val %s ?) OR
						(default_properties.type='tag' AND properties.val GLOB %s) OR
						(default_properties.type='user' AND properties.id IN
							(SELECT properties.id FROM properties
								LEFT JOIN accessors ON properties.val=accessors.id
								LEFT JOIN default_properties ON properties.default_id=default_properties.id
								WHERE default_properties.type='user' AND %s
							)
						)
					)
				`, eq, tagGlob, userWhere)
				innerVals = append(innerVals, vl)
				innerVals = append(innerVals, whereVals...)
				q += vq
			}
			q += "))"
			innerKeys = append(innerKeys, q)
		}
		where := ""
		if len(innerKeys) != 0 {
			where = "WHERE " + strings.Join(innerKeys, " AND ")
		}
		target := "entries"
		if findParent {
			target = "parents"
		}
		innerQuery := fmt.Sprintf(`
			SELECT %s.id FROM entries
			LEFT JOIN entries AS parents ON entries.parent_id=parents.id
			LEFT JOIN properties ON entries.id=properties.entry_id
			LEFT JOIN default_properties ON properties.default_id=default_properties.id
			LEFT JOIN entry_types ON entries.type_id = entry_types.id
			%v
		`, target, where)
		innerQueries = append(innerQueries, innerQuery)
	}
	queryTmpl := `
		SELECT
			entries.id,
			entries.path,
			entry_types.name,
			entries.archived,
			entries.created_at,
			(SELECT time FROM logs WHERE logs.entry_id=entries.id ORDER BY id DESC LIMIT 1),
			thumbnails.id
		FROM entries
		LEFT JOIN entry_types ON entries.type_id = entry_types.id
		LEFT JOIN thumbnails ON entries.id = thumbnails.entry_id
		WHERE %s AND %s AND %s AND %s AND %s AND %s
	`
	vals := make([]any, 0)
	whereArchived := "TRUE"
	if !admin || !showArchived {
		whereArchived = "entries.archived=0"
	}
	whereType := "TRUE"
	if whType != nil {
		whereType = "entry_types.name " + whType.Not() + whType.Equal() + " ?"
		vals = append(vals, whType.Value())
	}
	whereRoot := "entries.path LIKE ?"
	vals = append(vals, search.SearchRoot+`/%`)
	wherePath := "TRUE"
	if whPath != nil {
		wherePath = "entries.path " + whPath.Not() + whPath.Equal() + " ?"
		vals = append(vals, whPath.Value())
	}
	whereName := "TRUE"
	if whName != nil {
		// Need in-exact search.
		whName.Exact = false
		whereName = "entries.path " + whName.Not() + whName.Equal() + " ?"
		vals = append(vals, "%/"+whName.Value())
	}
	whereInner := "TRUE"
	if len(innerQueries) != 0 {
		whereInner = fmt.Sprintf("entries.id IN (%s)", strings.Join(innerQueries, " INTERSECT "))
		vals = append(vals, innerVals...)
	}
	query := fmt.Sprintf(queryTmpl, whereArchived, whereType, whereRoot, wherePath, whereName, whereInner)
	// We need these prints time to time. Do not delete.
	// fmt.Println(query)
	// fmt.Println(vals)
	valNeeds := strings.Count(query, "?")
	if len(vals) != valNeeds {
		return nil, fmt.Errorf("query doesn't get exact amount of values: got %v, want %v", len(vals), valNeeds)
	}
	rows, err := tx.QueryContext(ctx, query, vals...)
	if err != nil {
		return nil, err
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
	admin, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !admin {
		return forge.Unauthorized("user doesn't have permission: %v", user)
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
	admin, err := isAdmin(tx, ctx, user)
	if err != nil {
		return err
	}
	if !admin {
		return forge.Unauthorized("user doesn't have permission: %v", user)
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
