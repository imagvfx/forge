package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
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
	Cmp     string
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

func (w where) Values() []string {
	if w.Val == "" {
		return nil
	}
	vals := make([]string, 0)
	for _, v := range strings.Split(w.Val, ",") {
		if !w.Exact {
			v = "*" + v + "*"
		}
		vals = append(vals, v)
	}
	return vals
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
	wheres := make([]where, 0, len(search.Keywords))
	for _, kwd := range search.Keywords {
		kwd = strings.TrimSpace(kwd)
		if kwd == "" {
			continue
		}
		wh := where{}
		cmp := ""
		idx := len(kwd)
		// order of cmps are important. don't let prior values shadow later.
		// ex) if compare a keyword with "<" earlier, "<=" cannot be compared.
		cmps := []string{"=", "!=", ":", "!:", "<=", ">=", "<", ">"}
		for _, c := range cmps {
			i := strings.Index(kwd, c)
			if i != -1 && i < idx {
				idx = i
				cmp = c
			}
		}
		if idx == len(kwd) {
			wh.Val = kwd
			wheres = append(wheres, wh)
			continue
		}
		key, val, _ := strings.Cut(kwd, cmp)
		if key == "" {
			// Invalid search. Having ':' or '=' without the keyword.
			continue
		}
		for _, ch := range cmp {
			if ch == '=' {
				wh.Exact = true
			}
			if ch == '!' {
				wh.Exclude = true
			}
		}
		wh.Key = key
		wh.Cmp = cmp
		wh.Val = val // exclude colon or equal
		// special keywords those aren't actual properties.
		// multiple queries on special keywords aren't supported yet and will pick up the last one.
		wheres = append(wheres, wh)
	}
	if len(wheres) == 0 {
		return nil, nil
	}

	innerQueries := make([]string, 0)
	innerVals := make([]any, 0)
	for _, wh := range wheres {
		key := wh.Key
		rawval := wh.Val
		eq := wh.Equal()
		findParent := false
		queries := make([]string, 0)
		queryVals := make([]any, 0)
		expandSpecialValue := func(v string) string {
			if strings.HasPrefix(v, "@today") {
				day := time.Now().Local()
				if v == "@today" {
					return day.Format("2006/01/02")
				}
				// check it is @today+n, @today-n form
				suffix := v[len("@today"):]
				op := suffix[0]
				n, err := strconv.Atoi(suffix[1:])
				if err != nil {
					return v
				}
				if n > 10000 {
					// time.Hour is already a big number and
					// if n is also too big, i.e 10000000000,
					// ParseDuration will raise error.
					// let's limit n to +- 10000
					n = 10000
				}
				d := time.Duration(n) * 24 * time.Hour
				if op == '+' {
					day = day.Add(d)
				} else {
					day = day.Add(-d)
				}
				return day.Format("2006/01/02")
			}
			if v == "@user" {
				return user
			}
			return v
		}
		if wh.Key == "" {
			rawval := expandSpecialValue(rawval)
			val := "*" + rawval + "*"
			// Generic search. Not tied to a property.
			queries = append(queries, `
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
			queryVals = append(queryVals, pathl, val, val, val)
		} else if key == "path" {
			// special keyword "path"
			vals := wh.Values()
			if len(vals) != 0 {
				q := "("
				for i, v := range vals {
					if i != 0 {
						q += " OR "
					}
					q += "entries.path " + wh.Not() + wh.Equal() + " ?"
					queryVals = append(queryVals, v)
				}
				q += ")"
				queries = append(queries, q)
			}
		} else if key == "name" {
			// special keyword "name"
			// workaround to glob limitation.
			// user should provide the exact name.
			wh.Exact = false
			vals := wh.Values()
			if len(vals) != 0 {
				q := "("
				for i, v := range vals {
					if i != 0 {
						q += " OR "
					}
					q += "entries.path " + wh.Not() + wh.Equal() + " ?"
					queryVals = append(queryVals, "*/"+v)
				}
				q += ")"
				queries = append(queries, q)
			}
		} else if key == "type" {
			// special keyword "type"
			// could't think of in-exact type search
			wh.Exact = true
			vals := wh.Values()
			if len(vals) != 0 {
				q := "("
				for i, v := range vals {
					if i != 0 {
						q += " OR "
					}
					q += "entry_types.name " + wh.Not() + wh.Equal() + " ?"
					queryVals = append(queryVals, v)
				}
				q += ")"
				queries = append(queries, q)
			}
		} else {
			// generic keyword search
			sub := ""
			toks := strings.SplitN(key, ".", 2)
			if len(toks) == 2 {
				sub = toks[0]
				key = toks[1]
			}
			q := fmt.Sprintf("(default_properties.name=? AND ")
			queryVals = append(queryVals, key)
			if sub != "" {
				findParent = true
				if sub != "(sub)" {
					q += "entries.path GLOB ? AND"
					queryVals = append(queryVals, "*/"+sub)
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
				v = expandSpecialValue(v)
				vl := v
				if !wh.Exact {
					vl = "*" + v + "*"
				}
				itemGlob := ""
				if wh.Exact {
					if v == "" {
						itemGlob = "''"
					} else {
						itemGlob = "'*' || char(10) || '" + v + "' || char(10) || '*'"
					}
				} else {
					itemGlob = "'*" + v + "*'"
				}
				dateCmp := ""
				dateVal := ""
				if wh.Cmp == "<" || wh.Cmp == "<=" || wh.Cmp == ">" || wh.Cmp == ">=" {
					if wh.Cmp == "<" || wh.Cmp == "<=" {
						dateCmp = "!= '' AND properties.val" + wh.Cmp
					} else if wh.Cmp == ">" || wh.Cmp == ">=" {
						dateCmp = "!= '' AND properties.val" + wh.Cmp
					}
					// rest fills rest date when user put imcomplete yy or yy/mm format
					rest := "0000/00/00"
					if wh.Cmp == ">" || wh.Cmp == "<=" {
						rest = "9999/99/99"
					}
					dateVal = v
					if len(v) < len(rest) {
						dateVal += rest[len(v):]
					}
				} else {
					dateCmp = eq
					dateVal = vl
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
						(default_properties.type NOT IN ('tag', 'entry_link', 'user', 'date') AND properties.val %s ?) OR
						(default_properties.type IN ('tag', 'entry_link') AND properties.val GLOB %s) OR
						(default_properties.type='date' AND properties.val %s ?) OR
						(default_properties.type='user' AND properties.id IN
							(SELECT properties.id FROM properties
								LEFT JOIN accessors ON properties.val=accessors.id
								LEFT JOIN default_properties ON properties.default_id=default_properties.id
								WHERE default_properties.type='user' AND %s
							)
						)
					)
				`, eq, itemGlob, dateCmp, userWhere)
				queryVals = append(queryVals, vl, dateVal)
				queryVals = append(queryVals, whereVals...)
				q += vq
			}
			q += "))"
			queries = append(queries, q)
		}
		where := ""
		if len(queries) != 0 {
			where = "WHERE " + strings.Join(queries, " AND ")
		}
		target := "entries"
		if findParent {
			target = "parents"
		}
		query := fmt.Sprintf(`
			SELECT %s.id FROM entries
			LEFT JOIN entries AS parents ON entries.parent_id=parents.id
			LEFT JOIN properties ON entries.id=properties.entry_id
			LEFT JOIN default_properties ON properties.default_id=default_properties.id
			LEFT JOIN entry_types ON entries.type_id = entry_types.id
			%v
		`, target, where)
		innerQueries = append(innerQueries, query)
		innerVals = append(innerVals, queryVals...)
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
		WHERE %s AND %s AND %s
	`
	vals := make([]any, 0)
	whereArchived := "TRUE"
	if !showArchived {
		whereArchived = "entries.archived=0"
	}
	whereRoot := "entries.path GLOB ?"
	vals = append(vals, search.SearchRoot+`/*`)
	whereInner := "TRUE"
	if len(innerQueries) != 0 {
		whereInner = fmt.Sprintf("entries.id IN (%s)", strings.Join(innerQueries, " INTERSECT "))
		vals = append(vals, innerVals...)
	}
	query := fmt.Sprintf(queryTmpl, whereArchived, whereRoot, whereInner)
	if false {
		// We need these prints time to time. Do not delete.
		fmt.Println(query)
		fmt.Println(vals)
	}
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
