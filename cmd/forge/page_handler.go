package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/imagvfx/forge"
)

type pageHandler struct {
	server *forge.Server
	cfg    *forge.Config
}

var pageHandlerFuncs = template.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
	"min": func(a, b int) int {
		if a < b {
			return a
		}
		return b
	},
	"has": func(s, tok string) bool {
		return strings.Contains(s, tok)
	},
	"pathLinks": func(path string) (template.HTML, error) {
		if !strings.HasPrefix(path, "/") {
			return "", fmt.Errorf("path should start with /")
		}
		full := ""
		link := ""
		ps := strings.Split(path[1:], "/")
		for _, p := range ps {
			p = "/" + p
			link += p
			full += fmt.Sprintf(`<a href="%v">%v</a>`, link, p)
		}
		return template.HTML(full), nil
	},
	"sortProperty": func(p string) string {
		if len(p) == 0 {
			return ""
		}
		if len(p) == 1 {
			// only sort order defined
			return ""
		}
		return p[1:]
	},
	"sortDesc": func(p string) bool {
		if len(p) == 0 {
			return false
		}
		order := p[0]
		// '+' means ascending, '-' means descending order
		return order == '-'
	},
	"marshalJS": func(v interface{}) (template.JS, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return template.JS(string(b)), nil
	},
	"brLines": func(s string) template.HTML {
		// Similar code is in tmpl/entry.bml.js for in-place update.
		// Modify both, if needed.
		t := ""
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				t += "<br>"
				continue
			}
			if strings.HasPrefix(line, "/") {
				t += "<div class='pathText'>" + line + "</div>"
			} else {
				t += "<div>" + line + "</div>"
			}
		}
		return template.HTML(t)
	},
	"toURL": func(s string) template.URL {
		return template.URL(s)
	},
	// topName is name of the entry directly under the root for given path.
	// Which indicates a show name.
	"topName": func(s string) string {
		return strings.Split(s, "/")[1]
	},
	"setAlphaToColor": func(c string, alpha float32) string {
		c = strings.TrimSpace(c)
		hexChar := "0123456789ABCDEF"
		if strings.HasPrefix(c, "#") {
			cc := c[1:]
			if len(cc) != 3 && len(cc) != 6 {
				return c
			}
			long := len(cc) == 6
			a := int(alpha * 255)
			astr := string(hexChar[a/16])
			if long {
				astr += string(hexChar[a%16])
			}
			return c + astr
		}
		return c
	},
	"recentlyUpdated": func(i interface{}) (bool, error) {
		var updatedAt time.Time
		switch v := i.(type) {
		case *forge.Property:
			updatedAt = v.UpdatedAt
		case *forge.AccessControl:
			updatedAt = v.UpdatedAt
		default:
			return false, fmt.Errorf("unknown type: %v", i)
		}
		delta := time.Now().UTC().Sub(updatedAt)
		if delta < 24*time.Hour {
			return true, nil
		}
		return false, nil
	},
}

func httpStatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	var notFound *forge.NotFoundError
	if errors.As(err, &notFound) {
		return http.StatusNotFound
	}
	var unauthorized *forge.UnauthorizedError
	if errors.As(err, &unauthorized) {
		return http.StatusUnauthorized
	}
	return http.StatusBadRequest
}

func handleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	status := httpStatusFromError(err)
	if status == http.StatusNotFound {
		http.Error(w, err.Error(), status)
		return
	}
	// Log unauthorized and undefined errors.
	log.Print(err)
	http.Error(w, err.Error(), status)
}

func (h *pageHandler) Handler(handleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			session, err := getSession(r)
			if err != nil {
				clearSession(w)
				return err
			}
			user := session["user"]
			if user == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return nil
			}
			ctx := forge.ContextWithUserName(r.Context(), user)
			return handleFunc(ctx, w, r)
		}()
		handleError(w, err)
	}
}

func (h *pageHandler) handleEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	setting, err := h.server.GetUserSetting(ctx, user)
	if err != nil {
		return err
	}
	path := r.URL.Path
	ent, err := h.server.GetEntry(ctx, path)
	if err != nil {
		return err
	}
	props, err := h.server.EntryProperties(ctx, path)
	if err != nil {
		return err
	}
	envs, err := h.server.EntryEnvirons(ctx, path)
	if err != nil {
		return err
	}
	acs, err := h.server.EntryAccessControls(ctx, path)
	if err != nil {
		return err
	}
	pinned := false
	for _, p := range setting.PinnedPaths {
		if p == path {
			pinned = true
			break
		}
	}
	resultsFromSearch := false
	var subEnts []*forge.Entry
	search := r.FormValue("search")
	searchEntryType := r.FormValue("search_entry_type")
	searchQuery := r.FormValue("search_query")
	if search != "" {
		// User pressed search button,
		// Note that it rather clear the search if every search field is emtpy.
		if searchEntryType != setting.EntryPageSearchEntryType {
			// Whether perform search or not, it will remeber the search entry type.
			err := h.server.UpdateUserSetting(ctx, user, "entry_page_search_entry_type", searchEntryType)
			if err != nil {
				return err
			}
			// the update doesn't affect current page
			setting.EntryPageSearchEntryType = searchEntryType
		}
		if searchEntryType != "" || searchQuery != "" {
			resultsFromSearch = true
			subEnts, err = h.server.SearchEntries(ctx, path, searchEntryType, searchQuery)
			if err != nil {
				return err
			}
		}
	}
	if !resultsFromSearch {
		// Normal entry page
		subEnts, err = h.server.SubEntries(ctx, path)
		if err != nil {
			return err
		}
		searchEntryType = setting.EntryPageSearchEntryType
	}
	subEntsByTypeByParent := make(map[string]map[string][]*forge.Entry)
	if !resultsFromSearch {
		// If the entry have .sub_entry_types property,
		// default sub entry types are overrided with the property.
		subtyps := make([]string, 0)
		for _, p := range props {
			if p.Name == ".sub_entry_types" {
				for _, subtyp := range strings.Split(p.Value, ",") {
					if subtyp == "" {
						continue
					}
					subtyps = append(subtyps, strings.TrimSpace(subtyp))
				}
				break
			}
		}
		for _, t := range subtyps {
			subEntsByTypeByParent[t] = make(map[string][]*forge.Entry)
		}
	}
	subEntProps := make(map[string]map[string]*forge.Property)
	for _, e := range subEnts {
		if subEntsByTypeByParent[e.Type] == nil {
			// This should come from search results.
			subEntsByTypeByParent[e.Type] = make(map[string][]*forge.Entry)
		}
		byParent := subEntsByTypeByParent[e.Type]
		parent := filepath.Dir(e.Path)
		if e.Path == "/" {
			parent = ""
		}
		if byParent[parent] == nil {
			byParent[parent] = make([]*forge.Entry, 0)
		}
		byParent[parent] = append(byParent[parent], e)
		subEntsByTypeByParent[e.Type] = byParent
		// subProps
		props, err := h.server.EntryProperties(ctx, e.Path)
		if err != nil {
			return err
		}
		subProps := make(map[string]*forge.Property)
		for _, p := range props {
			subProps[p.Name] = p
		}
		subEntProps[e.Path] = subProps
	}
	// sort
	for t, byParent := range subEntsByTypeByParent {
		var prop string
		var desc bool
		sortProp := setting.EntryPageSortProperty[t]
		if sortProp != "" {
			prefix := sortProp[0]
			prop = sortProp[1:]
			if prefix == '-' {
				desc = true
			}
		}
		for _, ents := range byParent {
			sort.Slice(ents, func(i, j int) bool {
				if prop == "" {
					if !desc {
						return ents[i].Name() < ents[j].Name()
					}
					return ents[i].Name() > ents[j].Name()
				}
				ip := subEntProps[ents[i].Path][prop]
				jp := subEntProps[ents[j].Path][prop]
				iv := ip.Value
				jv := jp.Value
				var less bool
				if ip.Type != jp.Type {
					less = ip.Type < jp.Type
				} else if iv == jv {
					less = ents[i].Name() < ents[j].Name()
				} else {
					less = forge.LessProperty(ip.Type, iv, jv)
				}
				if desc {
					less = !less
				}
				if iv == "" || jv == "" {
					// Entry with empty value should stand behind of non-empty value
					// regardless of the order type.
					if iv == "" {
						less = false
					} else {
						less = true
					}
				}
				return less
			})
		}

	}
	// property filter
	defaultProps := make(map[string][]string)
	propFilters := make(map[string][]string)
	for typ := range subEntsByTypeByParent {
		defaults, err := h.server.Defaults(ctx, typ)
		if err != nil {
			return err
		}
		for _, d := range defaults {
			if d.Category == "property" && !strings.HasPrefix(d.Name, ".") {
				defaultProps[typ] = append(defaultProps[typ], d.Name)
			}
		}
		// user property filter
		if setting.EntryPagePropertyFilter != nil && setting.EntryPagePropertyFilter[typ] != "" {
			filter := setting.EntryPagePropertyFilter[typ]
			propFilters[typ] = strings.Fields(filter)
			continue
		}
		// global property filter
		g, err := h.server.GetGlobal(ctx, typ, "property_filter")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			// neither filter exists
			propFilters[typ] = defaultProps[typ]
			continue
		}
		propFilters[typ] = strings.Fields(g.Value)
	}
	type entStatus struct {
		Name   string
		Type   string
		Status string
	}
	grandSubTypes := make(map[string]bool, 0)
	grandSubStatus := make(map[string][]entStatus)
	showGrandSub := make(map[string]bool)
	for _, sub := range subEnts {
		show, ok := showGrandSub[sub.Type]
		if !ok {
			g, err := h.server.GetGlobal(ctx, sub.Type, "expose_sub_entries")
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return err
				}
			}
			// show == false
			if g != nil {
				show = true
			}
			showGrandSub[sub.Type] = show
		}
		if !show {
			continue
		}
		gsubEnts, err := h.server.SubEntries(ctx, sub.Path)
		if err != nil {
			return err
		}
		sort.Slice(gsubEnts, func(i, j int) bool {
			// TODO: sort by the property defined in user preference
			return gsubEnts[i].Name() < gsubEnts[j].Name()
		})
		gsubStatus := make([]entStatus, 0)
		for _, gs := range gsubEnts {
			grandSubTypes[gs.Type] = true
			p, err := h.server.GetProperty(ctx, gs.Path, "status")
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return err
				}
			}
			stat := ""
			if p != nil {
				stat = p.Value
			}
			gsubStatus = append(gsubStatus, entStatus{Name: gs.Name(), Type: gs.Type, Status: stat})
		}
		grandSubStatus[sub.Path] = gsubStatus
	}
	// TODO: maybe do it with allTypes?
	possibleTypes := make([]string, 0)
	for typ := range subEntsByTypeByParent {
		possibleTypes = append(possibleTypes, typ)
	}
	for typ := range grandSubTypes {
		possibleTypes = append(possibleTypes, typ)
	}
	possibleStatus := make(map[string][]forge.Status)
	for _, typ := range possibleTypes {
		status := make([]forge.Status, 0)
		p, err := h.server.GetGlobal(ctx, typ, "possible_status")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			continue
		}
		for _, ps := range strings.Fields(p.Value) {
			toks := strings.Split(ps, ":")
			name := toks[0]
			color := ""
			if len(toks) > 1 {
				color = toks[1]
			}
			status = append(status, forge.Status{Name: name, Color: color})
		}
		possibleStatus[typ] = status
	}
	hasThumbnail := make(map[string]bool)
	thumbnailPath := make(map[string]string)
	allEnts := append(subEnts, ent)
	for _, ent := range allEnts {
		if ent.HasThumbnail {
			hasThumbnail[ent.Path] = true
			thumbnailPath[ent.Path] = ent.Path
			continue
		}
		pth := ent.Path
		for {
			if hasThumbnail[pth] {
				thumbnailPath[ent.Path] = pth
				break
			}
			e, err := h.server.GetEntry(ctx, pth)
			if err != nil {
				return fmt.Errorf("check thumbnail info: %w", err)
			}
			if e.HasThumbnail {
				hasThumbnail[pth] = true
				thumbnailPath[ent.Path] = pth
				break
			}
			if pth == "/" {
				break
			}
			pth = filepath.Dir(pth)
		}
	}
	baseTypes, err := h.server.FindBaseEntryTypes(ctx)
	if err != nil {
		return err
	}
	allUsers, err := h.server.Users(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User                     string
		UserSetting              *forge.UserSetting
		Entry                    *forge.Entry
		EntryPinned              bool
		SearchEntryType          string
		SearchQuery              string
		ResultsFromSearch        bool
		SubEntriesByTypeByParent map[string]map[string][]*forge.Entry
		SubEntryProperties       map[string]map[string]*forge.Property
		GrandSubStatus           map[string][]entStatus
		PropertyTypes            []string
		DefaultProperties        map[string][]string
		PropertyFilters          map[string][]string
		PossibleStatus           map[string][]forge.Status
		Properties               []*forge.Property
		Environs                 []*forge.Property
		AccessorTypes            []string
		AccessControls           []*forge.AccessControl
		ThumbnailPath            map[string]string
		BaseEntryTypes           []string
		AllUsers                 []*forge.User
	}{
		User:                     user,
		UserSetting:              setting,
		Entry:                    ent,
		EntryPinned:              pinned,
		SearchEntryType:          searchEntryType,
		SearchQuery:              searchQuery,
		ResultsFromSearch:        resultsFromSearch,
		SubEntriesByTypeByParent: subEntsByTypeByParent,
		SubEntryProperties:       subEntProps,
		GrandSubStatus:           grandSubStatus,
		PropertyTypes:            forge.PropertyTypes(),
		DefaultProperties:        defaultProps,
		PropertyFilters:          propFilters,
		PossibleStatus:           possibleStatus,
		Properties:               props,
		Environs:                 envs,
		AccessorTypes:            forge.AccessorTypes(),
		AccessControls:           acs,
		ThumbnailPath:            thumbnailPath,
		BaseEntryTypes:           baseTypes,
		AllUsers:                 allUsers,
	}
	err = Tmpl.ExecuteTemplate(w, "entry.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleEntryLogs(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	path := r.FormValue("path")
	ent, err := h.server.GetEntry(ctx, path)
	if err != nil {
		return err
	}
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	if ctg != "" || name != "" {
		logs, err := h.server.GetLogs(ctx, path, ctg, name)
		if err != nil {
			return err
		}
		// history is selected set of logs of an item.
		history := make([]*forge.Log, 0)
		for _, l := range logs {
			if l.Value == "" {
				continue
			}
			l.When = l.When.Local()
			history = append(history, l)
		}
		recipe := struct {
			User     string
			Entry    *forge.Entry
			Category string
			Name     string
			History  []*forge.Log
		}{
			User:     user,
			Entry:    ent,
			Category: ctg,
			Name:     name,
			History:  history,
		}
		err = Tmpl.ExecuteTemplate(w, "entry-item-history.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	} else {
		logs, err := h.server.EntryLogs(ctx, path)
		if err != nil {
			return err
		}
		for _, l := range logs {
			l.When = l.When.Local()
		}
		recipe := struct {
			User  string
			Entry *forge.Entry
			Logs  []*forge.Log
		}{
			User:  user,
			Entry: ent,
			Logs:  logs,
		}
		err = Tmpl.ExecuteTemplate(w, "entry-logs.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}
}

func (h *pageHandler) handleThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if !strings.HasPrefix(r.URL.Path, "/thumbnail/") {
		return fmt.Errorf("invalid thumbnail path")
	}
	path := strings.TrimPrefix(r.URL.Path, "/thumbnail")
	thumb, err := h.server.GetThumbnail(ctx, path)
	if err != nil {
		return err
	}
	sum := md5.Sum(thumb.Data)
	hash := base64.URLEncoding.EncodeToString(sum[:])
	if r.Header.Get("If-None-Match") == hash {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", hash)
	_, err = io.Copy(w, bytes.NewReader(thumb.Data))
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	users, err := h.server.Users(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User    string
		Users   []*forge.User
		Members map[string][]*forge.Member
	}{
		User:  user,
		Users: users,
	}
	err = Tmpl.ExecuteTemplate(w, "users.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleGroups(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	groups, err := h.server.FindAllGroups(ctx)
	if err != nil {
		return err
	}
	members := make(map[string][]*forge.Member)
	for _, g := range groups {
		mems, err := h.server.FindGroupMembers(ctx, g.Name)
		if err != nil {
			return err
		}
		members[g.Name] = mems
	}
	recipe := struct {
		User    string
		Groups  []*forge.Group
		Members map[string][]*forge.Member
	}{
		User:    user,
		Groups:  groups,
		Members: members,
	}
	err = Tmpl.ExecuteTemplate(w, "groups.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleEntryTypes(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	typeNames, err := h.server.FindBaseEntryTypes(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User           string
		EntryTypeNames []string
	}{
		User:           user,
		EntryTypeNames: typeNames,
	}
	err = Tmpl.ExecuteTemplate(w, "types.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleEachEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	toks := strings.Split(r.URL.Path, "/")
	tname := toks[2]
	if tname == "" {
		return fmt.Errorf("entry type not specified")
	}
	allTypeNames, err := h.server.FindEntryTypes(ctx)
	if err != nil {
		return err
	}
	typeNames := make([]string, 0)
	for _, tn := range allTypeNames {
		if tn == tname {
			typeNames = append(typeNames, tn)
			continue
		}
		if strings.HasPrefix(tn, tname+".") {
			typeNames = append(typeNames, tn)
			continue
		}
	}
	if len(typeNames) == 0 {
		return fmt.Errorf("entry type not found: %v", tname)
	}
	sort.Strings(typeNames)
	// TODO: maybe better to have this in package forge?
	type EntryType struct {
		Name     string
		Globals  []*forge.Global
		Defaults []*forge.Default
	}
	types := make([]*EntryType, 0)
	for _, tname := range typeNames {
		globals, err := h.server.Globals(ctx, tname)
		if err != nil {
			return err
		}
		defaults, err := h.server.Defaults(ctx, tname)
		if err != nil {
			return err
		}
		t := &EntryType{
			Name:     tname,
			Globals:  globals,
			Defaults: defaults,
		}
		types = append(types, t)
	}
	recipe := struct {
		User       string
		EntryTypes []*EntryType
	}{
		User:       user,
		EntryTypes: types,
	}
	err = Tmpl.ExecuteTemplate(w, "type.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}
