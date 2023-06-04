package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

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
	return "GLOB"
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
	return `*` + w.Val + `*`
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
		idx := len(kwd)
		for _, cmp := range ":=" {
			i := strings.Index(kwd, string(cmp))
			if i != -1 && i < idx {
				idx = i
			}
		}
		if idx == len(kwd) {
			wh.Val = kwd
			wheres = append(wheres, wh)
			continue
		}
		cmp := kwd[idx]
		if cmp == '=' {
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
				(entries.path GLOB ? OR
					(default_properties.name NOT GLOB '.*' AND
						(
							(default_properties.type!='user' AND properties.val GLOB ?) OR
							(default_properties.type='user' AND properties.id IN
								(SELECT properties.id FROM properties
									LEFT JOIN accessors ON properties.val=accessors.id
									LEFT JOIN default_properties ON properties.default_id=default_properties.id
									WHERE default_properties.type='user' AND (accessors.called GLOB ? OR accessors.name GLOB ?)
								)
							)
						)
					)
				)
			`)
			pathl := rawval + "*"
			if !strings.HasPrefix(rawval, "/") {
				// relative path
				pathl = search.SearchRoot + "*" + rawval + "*"
			}
			innerVals = append(innerVals, pathl, val, val, val)
		} else {
			// keyword search
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
					q += "entries.path GLOB ? AND"
					innerVals = append(innerVals, "*/"+sub)
				}
			}
			not := ""
			if wh.Exclude {
				not = "NOT"
			}
			q += " " + not + " ("
			vs := strings.Split(rawval, ",")
			for i, v := range vs {
				// multiple values separated by comma
				if i != 0 {
					q += " OR "
				}
				vl := v
				if !wh.Exact {
					vl = "*" + v + "*"
				}
				tagGlob := ""
				if wh.Exact {
					if v == "" {
						tagGlob = "''"
					} else {
						tagGlob = "'*' || char(10) || '" + v + "' || char(10) || '*'"
					}
				} else {
					tagGlob = "'*" + v + "*'"
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
	if !showArchived {
		whereArchived = "entries.archived=0"
	}
	whereType := "TRUE"
	if whType != nil {
		whereType = "entry_types.name " + whType.Not() + whType.Equal() + " ?"
		vals = append(vals, whType.Value())
	}
	whereRoot := "entries.path GLOB ?"
	vals = append(vals, search.SearchRoot+`/*`)
	wherePath := "TRUE"
	if whPath != nil {
		wherePath = "entries.path " + whPath.Not() + whPath.Equal() + " ?"
		vals = append(vals, whPath.Value())
	}
	whereName := "TRUE"
	if whName != nil {
		// workaround of glob limitation.
		// user should provide the exact name.
		whName.Exact = false
		whereName = "entries.path " + whName.Not() + whName.Equal() + " ?"
		vals = append(vals, "*/"+whName.Val)
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
