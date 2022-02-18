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
	"trim": strings.TrimSpace,
	"remapFrom": func(s string) string {
		return strings.Split(s, ";")[0]
	},
	"remapTo": func(s string) string {
		if !strings.Contains(s, ";") {
			return ""
		}
		return strings.Split(s, ";")[1]
	},
	"formatTime": func(t time.Time) string {
		return t.Format(time.RFC3339)
	},
	"pathLinks": func(path string) (template.HTML, error) {
		if !strings.HasPrefix(path, "/") {
			return "", fmt.Errorf("path should start with /")
		}
		full := ""
		link := ""
		ps := strings.Split(path[1:], "/")
		for _, p := range ps {
			p = template.HTMLEscapeString("/" + p)
			link += p
			full += fmt.Sprintf(`<a class="pathLink" href="%v">%v</a>`, link, p)
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
			line = template.HTMLEscapeString(line)
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
	"recent": func(t time.Time, days int) (bool, error) {
		delta := time.Now().UTC().Sub(t)
		if delta < time.Duration(days)*24*time.Hour {
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
	// Get entries from current path or search results.
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
	// Organize the sub entries by type and by parent.
	subEntsByTypeByParent := make(map[string]map[string][]*forge.Entry) // map[type]map[parent]
	entProps := make(map[string]map[string]*forge.Property)
	entProps[path] = make(map[string]*forge.Property)
	for _, p := range props {
		entProps[path][p.Name] = p
	}
	if !resultsFromSearch {
		// It might not have any sub entry, still entry page needs the sub type labels.
		p := entProps[path][".sub_entry_types"]
		if p != nil {
			for _, subtyp := range strings.Split(p.Value, ",") {
				if subtyp != "" {
					t := strings.TrimSpace(subtyp)
					subEntsByTypeByParent[t] = make(map[string][]*forge.Entry)
				}
			}
		}
	}
	parentEnts := make(map[string]*forge.Entry)
	for _, e := range subEnts {
		if subEntsByTypeByParent[e.Type] == nil {
			// This should come from search results.
			subEntsByTypeByParent[e.Type] = make(map[string][]*forge.Entry)
		}
		byParent := subEntsByTypeByParent[e.Type]
		parent := filepath.Dir(e.Path)
		if _, ok := parentEnts[parent]; !ok {
			p, err := h.server.GetEntry(ctx, parent)
			if err != nil {
				return err
			}
			parentEnts[parent] = p
			if parent != ent.Path {
				props, err := h.server.EntryProperties(ctx, parent)
				if err != nil {
					var e *forge.NotFoundError
					if !errors.As(err, &e) {
						return err
					}
				}
				entProps[parent] = make(map[string]*forge.Property)
				for _, p := range props {
					entProps[parent][p.Name] = p
				}
			}
		}
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
		propmap := make(map[string]*forge.Property)
		for _, p := range props {
			propmap[p.Name] = p
		}
		entProps[e.Path] = propmap
	}
	// Sort sub entries.
	entrySortProp := make(map[string]string)
	entrySortDesc := make(map[string]bool)
	for typ, prop := range setting.EntryPageSortProperty {
		if prop == "" {
			continue
		}
		desc := false
		prefix := string(prop[0])
		if prefix == "+" {
		} else if prefix == "-" {
			desc = true
		} else {
			continue
		}
		prop = prop[1:]
		entrySortProp[typ] = prop
		entrySortDesc[typ] = desc
	}
	sortEntries := func(ents []*forge.Entry) {
		sort.Slice(ents, func(i, j int) bool {
			a := ents[i]
			b := ents[j]
			cmp := strings.Compare(a.Type, b.Type)
			if cmp != 0 {
				return cmp < 0
			}
			typ := a.Type
			k := 1
			if entrySortDesc[typ] {
				k = -1
			}
			cmp = func() int {
				sortProp := entrySortProp[typ]
				if sortProp == "" {
					return 0
				}
				aProp := entProps[a.Path][sortProp]
				if aProp == nil {
					return -1
				}
				bProp := entProps[b.Path][sortProp]
				if bProp == nil {
					return 1
				}
				// Even they are properties with same name, their types can be different.
				cmp := k * strings.Compare(aProp.Type, bProp.Type)
				if cmp != 0 {
					return cmp
				}
				cmp = k * forge.CompareProperty(aProp.Type, aProp.Value, bProp.Value)
				if cmp != 0 {
					return cmp
				}
				// It always sorts entries with their name in ascending order.
				return strings.Compare(a.Name(), b.Name())
			}()
			if cmp != 0 {
				return cmp < 0
			}
			// It sorts entries with their name with order set by entrySortDesc.
			cmp = k * strings.Compare(a.Name(), b.Name())
			return cmp <= 0
		})
	}
	for _, byParent := range subEntsByTypeByParent {
		for _, subEnts := range byParent {
			sortEntries(subEnts)
		}
	}
	// Determine property filter for entry types
	defaultProp := make(map[string]map[string]bool)
	propFilters := make(map[string][]string)
	entTypes := []string{ent.Type}
	for typ := range subEntsByTypeByParent {
		if typ != ent.Type {
			entTypes = append(entTypes, typ)
		}
	}
	sortProps := func(props []string) {
		sort.Slice(props, func(i, j int) bool {
			a := props[i]
			b := props[j]
			if !strings.HasPrefix(a, ".") && strings.HasPrefix(b, ".") {
				return true
			}
			if strings.HasPrefix(a, ".") && !strings.HasPrefix(b, ".") {
				return false
			}
			cmp := strings.Compare(a, b)
			return cmp <= 0
		})
	}
	for _, typ := range entTypes {
		defaults, err := h.server.Defaults(ctx, typ)
		if err != nil {
			return err
		}
		for _, d := range defaults {
			if d.Category == "property" && !strings.HasPrefix(d.Name, ".") {
				if defaultProp[typ] == nil {
					defaultProp[typ] = make(map[string]bool)
				}
				defaultProp[typ][d.Name] = true
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
			props := make([]string, 0)
			for p := range defaultProp[typ] {
				if !strings.HasPrefix(p, ".") {
					props = append(props, p)
				}
			}
			sortProps(props)
			propFilters[typ] = props
			continue
		}
		propFilters[typ] = strings.Fields(g.Value)
	}
	for typ := range propFilters {
		for _, p := range propFilters[typ] {
			delete(defaultProp[typ], p)
		}
	}
	hiddenProps := make(map[string][]string)
	for typ, prop := range defaultProp {
		hiddenProps[typ] = make([]string, 0)
		for p := range prop {
			hiddenProps[typ] = append(hiddenProps[typ], p)
		}
	}
	for _, props := range hiddenProps {
		sortProps(props)
	}
	mainEntryVisibleProp := make(map[string]bool)
	for _, p := range propFilters[ent.Type] {
		mainEntryVisibleProp[p] = true
	}
	mainEntryHiddenProps := make([]string, 0)
	for _, p := range entProps[path] {
		if !mainEntryVisibleProp[p.Name] {
			mainEntryHiddenProps = append(mainEntryHiddenProps, p.Name)
		}
	}
	sortProps(mainEntryHiddenProps)
	// Get grand sub entries if needed.
	grandSubEntries := make(map[string][]*forge.Entry)
	showGrandSub := make(map[string]bool)
	for _, sub := range subEnts {
		_, ok := showGrandSub[sub.Type]
		if ok {
			continue
		}
		g, err := h.server.GetGlobal(ctx, sub.Type, "expose_sub_entries")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
		}
		show := false
		if g != nil {
			show = true
		}
		showGrandSub[sub.Type] = show
	}
	for _, sub := range subEnts {
		if !showGrandSub[sub.Type] {
			continue
		}
		gsubEnts, err := h.server.SubEntries(ctx, sub.Path)
		if err != nil {
			return err
		}
		for _, gs := range gsubEnts {
			if _, ok := entProps[gs.Path]; !ok {
				entProps[gs.Path] = make(map[string]*forge.Property)
				props, err := h.server.EntryProperties(ctx, gs.Path)
				if err != nil {
					return err
				}
				for _, p := range props {
					entProps[gs.Path][p.Name] = p
				}
			}
			sortProp := entrySortProp[gs.Type]
			if entProps[gs.Path][sortProp] == nil {
				entrySortProp[gs.Type] = ""
			}
		}
		sortEntries(gsubEnts)
		grandSubEntries[sub.Path] = gsubEnts
	}
	// Get possible status for entry types defines it.
	possibleStatus := make(map[string][]forge.Status)
	baseTypes, err := h.server.FindBaseEntryTypes(ctx)
	if err != nil {
		return err
	}
	for _, typ := range baseTypes {
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
	// Get thumbnail paths of entries.
	hasThumbnail := make(map[string]bool)
	thumbnailPath := make(map[string]string)
	allEnts := append(subEnts, ent)
	for _, ent := range allEnts {
		if ent.HasThumbnail {
			hasThumbnail[ent.Path] = true
			thumbnailPath[ent.Path] = ent.Path
			continue
		}
		// The entry doesn't have thumbnail, try inherit from the ancestor's.
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
	entryPinned := false
	for _, p := range setting.PinnedPaths {
		if p == path {
			entryPinned = true
			break
		}
	}
	allUsers, err := h.server.Users(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User                      string
		UserSetting               *forge.UserSetting
		Entry                     *forge.Entry
		ParentEntries             map[string]*forge.Entry
		EntryPinned               bool
		SearchEntryType           string
		SearchQuery               string
		ResultsFromSearch         bool
		SubEntriesByTypeByParent  map[string]map[string][]*forge.Entry
		EntryProperties           map[string]map[string]*forge.Property
		ShowGrandSub              map[string]bool
		GrandSubEntries           map[string][]*forge.Entry
		PropertyTypes             []string
		MainEntryHiddenProperties []string
		HiddenProperties          map[string][]string
		PropertyFilters           map[string][]string
		PossibleStatus            map[string][]forge.Status
		Properties                []*forge.Property
		Environs                  []*forge.Property
		AccessorTypes             []string
		AccessControls            []*forge.AccessControl
		ThumbnailPath             map[string]string
		BaseEntryTypes            []string
		AllUsers                  []*forge.User
	}{
		User:                      user,
		UserSetting:               setting,
		Entry:                     ent,
		ParentEntries:             parentEnts,
		EntryPinned:               entryPinned,
		SearchEntryType:           searchEntryType,
		SearchQuery:               searchQuery,
		ResultsFromSearch:         resultsFromSearch,
		SubEntriesByTypeByParent:  subEntsByTypeByParent,
		EntryProperties:           entProps,
		ShowGrandSub:              showGrandSub,
		GrandSubEntries:           grandSubEntries,
		PropertyTypes:             forge.PropertyTypes(),
		MainEntryHiddenProperties: mainEntryHiddenProps,
		HiddenProperties:          hiddenProps,
		PropertyFilters:           propFilters,
		PossibleStatus:            possibleStatus,
		Properties:                props,
		Environs:                  envs,
		AccessorTypes:             forge.AccessorTypes(),
		AccessControls:            acs,
		ThumbnailPath:             thumbnailPath,
		BaseEntryTypes:            baseTypes,
		AllUsers:                  allUsers,
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

func (h *pageHandler) handleSetting(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	setting, err := h.server.GetUserSetting(ctx, user)
	if err != nil {
		return err
	}
	// TODO: change User as *forge.User in every templates
	recipe := struct {
		User    string
		I       *forge.User
		Setting *forge.UserSetting
	}{
		User:    user,
		I:       u,
		Setting: setting,
	}
	err = Tmpl.ExecuteTemplate(w, "setting.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}
