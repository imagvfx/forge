package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/imagvfx/forge"
	"github.com/xuri/excelize/v2"
)

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

type pageHandler struct {
	server *forge.Server
	cfg    *forge.Config
	login  *loginHandler
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
				h.login.Handle(w, r)
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
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
	setting, err := h.server.GetUserSetting(ctx, user)
	if err != nil {
		return err
	}
	pageSetting, err := h.server.GetUserDataSection(ctx, user, "entry_page")
	if err != nil {
		var e *forge.NotFoundError
		if !errors.As(err, &e) {
			return err
		}
		// prevent nil dereference
		pageSetting = &forge.UserDataSection{}
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
	acs, err := h.server.EntryAccessList(ctx, path)
	if err != nil {
		return err
	}
	// Get entries from current path or search results.
	resultsFromSearch := false
	var subEnts []*forge.Entry
	search := r.FormValue("search")
	if r.FormValue("search_query") != "" {
		// previously we used "search_query" field instead of "search".
		search = r.FormValue("search_query")
	}
	searchEntryType := r.FormValue("search_entry_type")
	if _, ok := r.Form["search_entry_type"]; ok {
		if searchEntryType != setting.EntryPageSearchEntryType {
			err := h.server.UpdateUserSetting(ctx, user, "entry_page_search_entry_type", searchEntryType)
			if err != nil {
				return err
			}
			// the update doesn't affect current page
			setting.EntryPageSearchEntryType = searchEntryType
		}
	}
	queryHasType := false
	groupByOverride := ""
	getTypes := []string{}
	if search != "" || searchEntryType != "" {
		// User pressed search button,
		// Note that it rather clear the search if every search field is emtpy.
		err := func() error {
			if searchEntryType == "" && search == "" {
				// user not searching
				return nil
			}
			resultsFromSearch = true
			queries := strings.Fields(search)
			mode := ""
			query := ""
			for i, q := range queries {
				if i != 0 {
					query += " "
				}
				if strings.HasPrefix(q, "-mode:") {
					mode = q[len("-mode:"):]
					continue
				}
				if strings.HasPrefix(q, "-get:") {
					v := q[len("-get:"):]
					getTypes = strings.Split(v, ",")
					continue
				}
				if strings.HasPrefix(q, "-group:") {
					groupByOverride = q[len("-group:"):]
					continue
				}
				if strings.HasPrefix(q, "type=") || strings.HasPrefix(q, "type:") {
					queryHasType = true
				}
				query += q
			}
			if mode == "entry" {
				// sql couldn't handle query if path list is too long
				// so let's just get one by one
				paths := make([]string, 0)
				for _, q := range queries {
					if strings.HasPrefix(q, "/") {
						paths = append(paths, q)
					}
				}
				for _, p := range paths {
					ent, err := h.server.GetEntry(ctx, p)
					if err != nil {
						var e *forge.NotFoundError
						if !errors.As(err, &e) {
							return err
						}
						continue
					}
					subEnts = append(subEnts, ent)
				}
				return nil
			}
			// default search
			typeQuery := ""
			if !queryHasType && searchEntryType != "" {
				typeQuery = "type=" + searchEntryType
			}
			ents, err := h.server.SearchEntries(ctx, path, typeQuery+" "+query)
			if err != nil {
				return err
			}
			subEnts = ents
			return nil
		}()
		if err != nil {
			return err
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
	newSubEnts := make(map[string]*forge.Entry)
	if len(getTypes) > 0 {
		for _, ent := range subEnts {
			for _, typ := range getTypes {
				if typ == ent.Type {
					newSubEnts[ent.Path] = ent
					break
				}
			}
			if _, ok := newSubEnts[ent.Path]; ok {
				// ent is already one of needed types.
				continue
			}
			// get parent of the type, if there is.
			ancs, err := h.server.FindEntries(ctx, forge.EntryFinder{
				ChildPath: &ent.Path,
				Types:     getTypes,
			})
			if err != nil {
				return err
			}
			if len(ancs) == 0 {
				continue
			}
			sort.Slice(ancs, func(i, j int) bool {
				// reverse sort, so we can get a nearest ancestor.
				return strings.Compare(ancs[i].Path, ancs[j].Path) > 0
			})
			anc := ancs[0]
			newSubEnts[anc.Path] = anc
		}
		subEnts = make([]*forge.Entry, 0, len(newSubEnts))
		for _, ent := range newSubEnts {
			subEnts = append(subEnts, ent)
		}
		sort.Slice(subEnts, func(i, j int) bool {
			return strings.Compare(subEnts[i].Path, subEnts[j].Path) < 0
		})
	}
	// Organize the sub entries by type and by parent.
	subEntsByTypeByGroup := make(map[string]map[string][]*forge.Entry) // map[type]map[parent]
	if !resultsFromSearch {
		// It might not have any sub entry, still entry page needs the sub type labels.
		p := ent.Property[".sub_entry_types"]
		if p != nil {
			for _, subtyp := range strings.Split(p.Value, ",") {
				if subtyp != "" {
					t := strings.TrimSpace(subtyp)
					subEntsByTypeByGroup[t] = make(map[string][]*forge.Entry)
				}
			}
		}
	}
	entryByPath := make(map[string]*forge.Entry)
	groupByProp := setting.EntryGroupBy
	if groupByOverride != "" {
		groupByProp = groupByOverride
	}
	if groupByProp == "" {
		for _, e := range subEnts {
			// at least one group needed
			if subEntsByTypeByGroup[e.Type] == nil {
				// This should come from search results.
				subEntsByTypeByGroup[e.Type] = make(map[string][]*forge.Entry)
			}
			byGroup := subEntsByTypeByGroup[e.Type]
			if byGroup[""] == nil {
				byGroup[""] = make([]*forge.Entry, 0)
			}
			byGroup[""] = append(byGroup[""], e)
			subEntsByTypeByGroup[e.Type] = byGroup
		}
	} else if groupByProp == "parent" {
		for _, e := range subEnts {
			if subEntsByTypeByGroup[e.Type] == nil {
				// This should come from search results.
				subEntsByTypeByGroup[e.Type] = make(map[string][]*forge.Entry)
			}
			byGroup := subEntsByTypeByGroup[e.Type]
			parent := filepath.Dir(e.Path)
			if _, ok := entryByPath[parent]; !ok {
				p, err := h.server.GetEntry(ctx, parent)
				if err != nil {
					return err
				}
				entryByPath[parent] = p
			}
			if e.Path == "/" {
				parent = ""
			}
			if byGroup[parent] == nil {
				byGroup[parent] = make([]*forge.Entry, 0)
			}
			byGroup[parent] = append(byGroup[parent], e)
			subEntsByTypeByGroup[e.Type] = byGroup
		}
	} else {
		var sub, prop string
		before, after, found := strings.Cut(groupByProp, ".")
		if found {
			sub = before
			prop = after
		} else {
			prop = before
		}
		for _, ent := range subEnts {
			byGroup := subEntsByTypeByGroup[ent.Type]
			if byGroup == nil {
				// This should come from search results.
				byGroup = make(map[string][]*forge.Entry)
			}
			path := ent.Path
			if sub != "" {
				path += "/" + sub
			}
			var e *forge.Entry
			if _, ok := entryByPath[path]; !ok {
				e, err = h.server.GetEntry(ctx, path)
				if err != nil {
					var e *forge.NotFoundError
					if !errors.As(err, &e) {
						return err
					}
				}
			}
			v := ""
			if prop != "" && e != nil {
				pr := e.Property[prop]
				if pr != nil {
					v = pr.Eval
				}
			}
			if byGroup[v] == nil {
				byGroup[v] = make([]*forge.Entry, 0)
			}
			byGroup[v] = append(byGroup[v], ent)
			subEntsByTypeByGroup[ent.Type] = byGroup
		}
	}
	statusSummary := make(map[string]map[string]int)
	for typ, byType := range subEntsByTypeByGroup {
		num := make(map[string]int)
		for _, byGroup := range byType {
			for _, ent := range byGroup {
				if stat := ent.Property["status"]; stat != nil {
					num[stat.Value] += 1
				} else {
					num[""] += 1
				}
			}
		}
		statusSummary[typ] = num
	}
	searches := make([][][2]string, 0)
	searchAt := []string{"/"}
	if ent.Path != "/" {
		toks := strings.Split(ent.Path, "/")
		showPath := strings.Join(toks[:2], "/")
		searchAt = append(searchAt, showPath)
	}
	for _, at := range searchAt {
		atSearches := make([][2]string, 0)
		show, err := h.server.GetEntry(ctx, at)
		if err != nil {
			return err
		}
		search, ok := show.Property["search"]
		if ok {
			for _, s := range strings.Split(search.Eval, "\n") {
				if s == "" {
					continue
				}
				name, query, ok := strings.Cut(s, "|")
				if !ok {
					continue
				}
				atSearches = append(atSearches, [2]string{name, query})
			}
		}
		searches = append(searches, atSearches)
	}
	tag := make(map[string]map[string]bool)
	for _, ent := range subEnts {
		for _, p := range ent.Property {
			if p.Type == "tag" {
				if tag[p.Name] == nil {
					tag[p.Name] = make(map[string]bool)
				}
				for _, v := range strings.Split(p.Value, "\n") {
					if v == "" {
						continue
					}
					tag[p.Name][v] = true
				}
			}
		}
	}
	subEntryTags := make(map[string][]string)
	for t, val := range tag {
		subEntryTags[t] = make([]string, 0, len(val))
		for v := range val {
			subEntryTags[t] = append(subEntryTags[t], v)
		}
		sort.Strings(subEntryTags[t])
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
				aProp := a.Property[sortProp]
				if aProp == nil {
					return -1
				}
				bProp := b.Property[sortProp]
				if bProp == nil {
					return 1
				}
				// Even they are properties with same name, their types can be different.
				cmp := k * strings.Compare(aProp.Type, bProp.Type)
				if cmp != 0 {
					return cmp
				}
				if aProp.Value == "" {
					cmp++
				}
				if bProp.Value == "" {
					cmp--
				}
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
			cmp = k * strings.Compare(a.Path, b.Path)
			return cmp <= 0
		})
	}
	for _, byGroup := range subEntsByTypeByGroup {
		for _, subEnts := range byGroup {
			sortEntries(subEnts)
		}
	}
	// Determine property filter for entry types
	defaultProp := make(map[string]map[string]bool)
	propFilters := make(map[string][]string)
	entTypes := []string{ent.Type}
	for typ := range subEntsByTypeByGroup {
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
	baseTypes, err := h.server.FindBaseEntryTypes(ctx)
	if err != nil {
		return err
	}
	for _, typ := range baseTypes {
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
	for _, p := range ent.Property {
		if !mainEntryVisibleProp[p.Name] {
			mainEntryHiddenProps = append(mainEntryHiddenProps, p.Name)
		}
	}
	sortProps(mainEntryHiddenProps)
	// Get grand sub entries if needed.
	grandSubEntGroups := make(map[string][][]*forge.Entry)
	grandSubTypes := make(map[string][]string)
	showGrandSub := make(map[string]bool)
	summaryGrandSub := make(map[string]bool)
	validTypes, err := h.server.FindEntryTypes(ctx)
	if err != nil {
		return err
	}
	validType := make(map[string]bool)
	for _, typ := range validTypes {
		validType[typ] = true
	}
	for _, sub := range subEnts {
		_, ok := grandSubTypes[sub.Type]
		if ok {
			continue
		}
		// expose_sub_entries might have entry types to expose later.
		subtypes, err := h.server.GetGlobal(ctx, sub.Type, "expose_sub_entries")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			continue
		}
		showGrandSub[sub.Type] = true
		grandSubTypes[sub.Type] = strings.Fields(subtypes.Value)
		// summary_sub_entries needs expose_sub_entries to effect
		_, err = h.server.GetGlobal(ctx, sub.Type, "summary_sub_entries")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			continue
		}
		summaryGrandSub[sub.Type] = true
	}
	grandSubEnts := make(map[*forge.Entry][]*forge.Entry)
	for _, sub := range subEnts {
		typs, ok := grandSubTypes[sub.Type]
		if !ok {
			continue
		}
		gsubEnts := make([]*forge.Entry, 0)
		if len(typs) == 0 {
			// get direct children
			gsub, err := h.server.SubEntries(ctx, sub.Path)
			if err != nil {
				return err
			}
			gsubEnts = gsub
		} else {
			// find grand children of the type recursively
			valids := []string{}
			for _, typ := range typs {
				if !validType[typ] {
					continue
				}
				valids = append(valids, typ)
			}
			gsub, err := h.server.FindEntries(ctx, forge.EntryFinder{AncestorPath: &sub.Path, Types: valids})
			if err != nil {
				return err
			}
			gsubEnts = append(gsubEnts, gsub...)
		}
		grandSubEnts[sub] = gsubEnts
	}
	for sub, gsubEnts := range grandSubEnts {
		sortEntries(gsubEnts)
		entryOrder := make(map[string]int)
		for i, gsub := range gsubEnts {
			entryOrder[gsub.Path] = i
		}
		toks := make(map[string][]string)
		for _, gsub := range gsubEnts {
			remain := gsub.Path[len(sub.Path)+1:]
			toks[gsub.Path] = strings.Split(remain, "/")
		}
		sort.Slice(gsubEnts, func(i, j int) bool {
			a := gsubEnts[i]
			b := gsubEnts[j]
			n := len(toks[a.Path])
			if len(toks[b.Path]) < n {
				n = len(toks[b.Path])
			}
			aPath := sub.Path
			bPath := sub.Path
			for i := 0; i < n; i++ {
				aPath += "/" + toks[a.Path][i]
				bPath += "/" + toks[b.Path][i]
				aOrd := entryOrder[aPath]
				bOrd := entryOrder[bPath]
				if aOrd == bOrd {
					continue
				}
				return aOrd < bOrd
			}
			return strings.Compare(a.Path, b.Path) < 0
		})
		groups := make([][]*forge.Entry, 0)
		oldg := ""
		grp := make([]*forge.Entry, 0)
		for _, gsub := range gsubEnts {
			remain := gsub.Path[len(sub.Path)+1:]
			g, _, _ := strings.Cut(remain, "/")
			if g != oldg && len(grp) != 0 {
				groups = append(groups, grp)
				grp = make([]*forge.Entry, 0)
			}
			grp = append(grp, gsub)
			oldg = g
		}
		if len(grp) != 0 {
			groups = append(groups, grp)
		}
		grandSubEntGroups[sub.Path] = groups
	}
	hideDueStatus := make(map[string]map[string]bool)
	for typ, gtyps := range grandSubTypes {
		// TODO: weird way to get all sub types, fix when we have a global one.
		typs := gtyps
		typs = append(typs, typ)
		for _, t := range typs {
			if hideDueStatus[t] != nil {
				continue
			}
			hideDueStatus[t] = make(map[string]bool)
			g, err := h.server.GetGlobal(ctx, t, "hide_due_status")
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return err
				}
				continue
			}
			stats := strings.Fields(g.Value)
			for _, s := range stats {
				if s == "_" {
					s = ""
				}
				hideDueStatus[t][s] = true
			}
		}
	}
	// Get possible status for entry types defines it.
	possibleStatus := make(map[string][]forge.Status)
	for _, typ := range baseTypes {
		status := make([]forge.Status, 0)
		p, err := h.server.GetGlobal(ctx, typ, "possible_status")
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			possibleStatus[typ] = status
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
	var prevEntry *forge.Entry
	var nextEntry *forge.Entry
	if !resultsFromSearch && ent.Path != "/" {
		// Normal entry page
		siblings, err := h.server.SubEntries(ctx, filepath.Dir(ent.Path))
		if err != nil {
			return err
		}
		if len(siblings) > 1 {
			sort.Slice(siblings, func(i, j int) bool {
				return siblings[i].Path < siblings[j].Path
			})
			idx := -1
			for i := range siblings {
				if siblings[i].Path == ent.Path {
					idx = i
					break
				}
			}
			p := idx - 1
			if p < 0 {
				p = len(siblings) - 1
			}
			n := idx + 1
			if n >= len(siblings) {
				n = 0
			}
			prevEntry = siblings[p]
			nextEntry = siblings[n]
		}
	}
	users, err := h.server.ActiveUsers(ctx)
	if err != nil {
		return err
	}
	disabledUsers, err := h.server.DisabledUsers(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User                      *forge.User
		UserIsAdmin               bool
		UserSetting               *forge.UserSetting
		PageSetting               *forge.UserDataSection
		Entry                     *forge.Entry
		EntryByPath               map[string]*forge.Entry
		PrevEntry                 *forge.Entry
		NextEntry                 *forge.Entry
		EntryPinned               bool
		SearchEntryType           string
		Search                    string
		QueryHasType              bool
		ResultsFromSearch         bool
		GroupByOverride           string
		GroupByProp               string
		SubEntriesByTypeByGroup   map[string]map[string][]*forge.Entry
		StatusSummary             map[string]map[string]int
		HideDueStatus             map[string]map[string]bool
		Searches                  [][][2]string // [at][][name, query]
		SubEntryTags              map[string][]string
		ShowGrandSub              map[string]bool
		SummaryGrandSub           map[string]bool
		GrandSubEntGroups         map[string][][]*forge.Entry
		PropertyTypes             []string
		MainEntryHiddenProperties []string
		HiddenProperties          map[string][]string
		PropertyFilters           map[string][]string
		PossibleStatus            map[string][]forge.Status
		Properties                []*forge.Property
		Environs                  []*forge.Property
		AccessorTypes             []string
		AccessList                []*forge.Access
		ThumbnailPath             map[string]string
		BaseEntryTypes            []string
		Users                     []*forge.User
		DisabledUsers             []*forge.User
	}{
		User:                      u,
		UserIsAdmin:               isAdmin,
		UserSetting:               setting,
		PageSetting:               pageSetting,
		Entry:                     ent,
		EntryByPath:               entryByPath,
		PrevEntry:                 prevEntry,
		NextEntry:                 nextEntry,
		EntryPinned:               entryPinned,
		SearchEntryType:           searchEntryType,
		Search:                    search,
		QueryHasType:              queryHasType,
		ResultsFromSearch:         resultsFromSearch,
		GroupByOverride:           groupByOverride,
		GroupByProp:               groupByProp,
		SubEntriesByTypeByGroup:   subEntsByTypeByGroup,
		StatusSummary:             statusSummary,
		HideDueStatus:             hideDueStatus,
		Searches:                  searches,
		SubEntryTags:              subEntryTags,
		ShowGrandSub:              showGrandSub,
		SummaryGrandSub:           summaryGrandSub,
		GrandSubEntGroups:         grandSubEntGroups,
		PropertyTypes:             forge.PropertyTypes(),
		MainEntryHiddenProperties: mainEntryHiddenProps,
		HiddenProperties:          hiddenProps,
		PropertyFilters:           propFilters,
		PossibleStatus:            possibleStatus,
		Properties:                props,
		Environs:                  envs,
		AccessorTypes:             forge.AccessorTypes(),
		AccessList:                acs,
		ThumbnailPath:             thumbnailPath,
		BaseEntryTypes:            baseTypes,
		Users:                     users,
		DisabledUsers:             disabledUsers,
	}
	err = Tmpl.ExecuteTemplate(w, "entry.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleEntryLogs(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
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
		// when item type is user, we need their name alog with id
		users, err := h.server.ActiveUsers(ctx)
		if err != nil {
			return err
		}
		disabledUsers, err := h.server.DisabledUsers(ctx)
		if err != nil {
			return err
		}
		users = append(users, disabledUsers...)
		called := map[string]string{}
		for _, u := range users {
			called[u.Name] = u.Called
		}
		// history is selected set of logs of an item.
		history := make([]*forge.Log, 0)
		for _, l := range logs {
			l.When = l.When.Local()
			if l.Type == "user" {
				user := l.Value
				value := called[user]
				if user != "" {
					if value == "" {
						value = "unknown user"
					}
					value += " (" + user + ")"
				}
				l.Value = value
			}
			history = append(history, l)
		}
		recipe := struct {
			User        *forge.User
			UserIsAdmin bool
			Entry       *forge.Entry
			Category    string
			Name        string
			History     []*forge.Log
			Users       []*forge.User
		}{
			User:        u,
			UserIsAdmin: isAdmin,
			Entry:       ent,
			Category:    ctg,
			Name:        name,
			History:     history,
			Users:       users,
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
			User        *forge.User
			UserIsAdmin bool
			Entry       *forge.Entry
			Logs        []*forge.Log
		}{
			User:        u,
			UserIsAdmin: isAdmin,
			Entry:       ent,
			Logs:        logs,
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
	editMode := false
	edit := r.FormValue("edit")
	if edit != "" {
		editMode = true
	}
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
	users, err := h.server.ActiveUsers(ctx)
	if err != nil {
		return err
	}
	disabledUsers, err := h.server.DisabledUsers(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User          *forge.User
		UserIsAdmin   bool
		EditMode      bool
		Users         []*forge.User
		DisabledUsers []*forge.User
		Members       map[string][]*forge.Member
	}{
		User:          u,
		UserIsAdmin:   isAdmin,
		EditMode:      editMode,
		Users:         users,
		DisabledUsers: disabledUsers,
	}
	err = Tmpl.ExecuteTemplate(w, "users.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleGroups(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
	groups, err := h.server.AllGroups(ctx)
	if err != nil {
		return err
	}
	members := make(map[string][]*forge.Member)
	for _, g := range groups {
		mems, err := h.server.GroupMembers(ctx, g.Name)
		if err != nil {
			return err
		}
		members[g.Name] = mems
	}
	recipe := struct {
		User        *forge.User
		UserIsAdmin bool
		Groups      []*forge.Group
		Members     map[string][]*forge.Member
	}{
		User:        u,
		UserIsAdmin: isAdmin,
		Groups:      groups,
		Members:     members,
	}
	err = Tmpl.ExecuteTemplate(w, "groups.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleEntryTypes(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
	typeNames, err := h.server.FindBaseEntryTypes(ctx)
	if err != nil {
		return err
	}
	recipe := struct {
		User           *forge.User
		UserIsAdmin    bool
		EntryTypeNames []string
	}{
		User:           u,
		UserIsAdmin:    isAdmin,
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
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
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
		User        *forge.User
		UserIsAdmin bool
		EntryTypes  []*EntryType
	}{
		User:        u,
		UserIsAdmin: isAdmin,
		EntryTypes:  types,
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
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
	setting, err := h.server.GetUserSetting(ctx, user)
	if err != nil {
		return err
	}
	data, err := h.server.FindUserData(ctx, forge.UserDataFinder{User: user})
	if err != nil {
		return err
	}
	recipe := struct {
		User        *forge.User
		UserIsAdmin bool
		Setting     *forge.UserSetting
		UserData    []*forge.UserDataSection
	}{
		User:        u,
		UserIsAdmin: isAdmin,
		Setting:     setting,
		UserData:    data,
	}
	err = Tmpl.ExecuteTemplate(w, "setting.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleUserData(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	isAdmin, err := h.server.IsAdmin(ctx, user)
	if err != nil {
		return err
	}
	section := strings.TrimPrefix(r.URL.Path, "/user-data/")
	data, err := h.server.FindUserData(ctx, forge.UserDataFinder{User: user, Section: &section})
	if err != nil {
		return err
	}
	if len(data) != 1 {
		return forge.NotFound("not found user data section: " + section)
	}
	// TODO: change User as *forge.User in every templates
	recipe := struct {
		User        *forge.User
		UserIsAdmin bool
		Section     *forge.UserDataSection
	}{
		User:        u,
		UserIsAdmin: isAdmin,
		Section:     data[0],
	}
	err = Tmpl.ExecuteTemplate(w, "user-data.bml", recipe)
	if err != nil {
		return err
	}
	return nil
}

func (h *pageHandler) handleDownloadAsExcel(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	_, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}

	// not really know what is happening, but r.ParseForm cannot populate formdata to r.Form.
	// this is a temporal alternative.
	r.FormValue("paths")

	paths, ok := r.PostForm["paths"]
	if !ok {
		return fmt.Errorf("please specify paths")
	}
	exposeSub := make(map[string]bool)
	exposeSubTypes := make(map[string][]string)

	validTypes, err := h.server.FindEntryTypes(ctx)
	if err != nil {
		return err
	}
	validType := make(map[string]bool)
	for _, t := range validTypes {
		validType[t] = true
	}

	ents := make(map[string][]*forge.Entry)              // [type][]entry
	thumbnail := make(map[string]*forge.Thumbnail)       // [path]thumbnail
	allSubEntry := make(map[string]map[string]bool)      // [type][sub-entry]
	subEntry := make(map[string]map[string]*forge.Entry) // [path][sub-entry]entry
	for _, pth := range paths {
		ent, err := h.server.GetEntry(ctx, pth)
		if err != nil {
			return err
		}
		if ents[ent.Type] == nil {
			ents[ent.Type] = make([]*forge.Entry, 0)
			glbs, err := h.server.Globals(ctx, ent.Type)
			if err != nil {
				return err
			}
			expSub := false
			var exp *forge.Global
			for _, g := range glbs {
				if g.Name == "expose_sub_entries" {
					expSub = true
					exp = g
					break
				}
			}
			if expSub {
				expSubTypes := make([]string, 0)
				for _, typ := range strings.Fields(exp.Value) {
					if validType[typ] {
						expSubTypes = append(expSubTypes, typ)
					}
				}
				exposeSub[ent.Type] = expSub
				exposeSubTypes[ent.Type] = expSubTypes
			}
		}
		ents[ent.Type] = append(ents[ent.Type], ent)
		th, err := h.server.GetThumbnail(ctx, pth)
		if err != nil {
			var notFound *forge.NotFoundError
			if !errors.As(err, &notFound) {
				return err
			}
		}
		thumbnail[pth] = th
		if !exposeSub[ent.Type] {
			continue
		}
		if allSubEntry[ent.Type] == nil {
			allSubEntry[ent.Type] = make(map[string]bool)
		}
		subEnts := make([]*forge.Entry, 0)
		subTypes := exposeSubTypes[ent.Type]
		if len(subTypes) == 0 {
			subs, err := h.server.SubEntries(ctx, pth)
			if err != nil {
				return err
			}
			subEnts = subs
		} else {
			valids := []string{}
			for _, typ := range subTypes {
				if validType[typ] {
					valids = append(valids, typ)
				}
			}
			subs, err := h.server.FindEntries(ctx, forge.EntryFinder{AncestorPath: &ent.Path, Types: valids})
			if err != nil {
				return err
			}
			subEnts = append(subEnts, subs...)
		}
		subEntry[pth] = make(map[string]*forge.Entry)
		for _, sub := range subEnts {
			subName := sub.Path[len(pth)+1:]
			allSubEntry[ent.Type][subName] = true
			subEntry[pth][subName] = sub
		}
	}
	sortedSubEntries := make(map[string][]string)
	for typ, sub := range allSubEntry {
		for name := range sub {
			sortedSubEntries[typ] = append(sortedSubEntries[typ], name)
		}
		// TODO: sort by default order?
		sort.Strings(sortedSubEntries[typ])
	}
	propsExport := make(map[string][]string)     // [type][]property
	numProps := make(map[string]int)             // [type]number-of-props
	propIndex := make(map[string]map[string]int) // [type][property]index
	for typ := range ents {
		var props []string
		globals, err := h.server.Globals(ctx, typ)
		if err != nil {
			return err
		}
		for _, g := range globals {
			if g.Name == "property_filter" {
				props = strings.Fields(g.Value)
				break
			}
		}
		if props == nil {
			props = make([]string, 0)
			defs, err := h.server.Defaults(ctx, typ)
			if err != nil {
				return err
			}
			for _, d := range defs {
				if d.Category != "property" {
					continue
				}
				props = append(props, d.Name)
			}
		}
		propsExport[typ] = props
	}
	for typ, props := range propsExport {
		numProps[typ] = len(props)
		propIndex[typ] = make(map[string]int)
		for i, p := range props {
			propIndex[typ][p] = i
		}
	}
	xl := excelize.NewFile()
	sheet_idx := 0
	for typ, typeEnts := range ents {
		if sheet_idx == 0 {
			xl.SetSheetName("Sheet1", typ)
		} else {
			xl.NewSheet(typ)
		}
		sheet, err := xl.NewStreamWriter(typ)
		if err != nil {
			return err
		}
		// fit width for thumbnails, 16 here isn't sized in pixel.
		sheet.SetColWidth(1, 1, 16)

		cell, err := excelize.CoordinatesToCellName(1, 1)
		if err != nil {
			return err
		}
		labels := make([]any, numProps[typ]+3)
		labels[0] = "thumbnail"
		labels[1] = "parent"
		labels[2] = "name"
		for i, prop := range propsExport[typ] {
			labels[i+3] = prop
		}
		for _, sub := range sortedSubEntries[typ] {
			if strings.Index(sub, "/") == -1 {
				// direct sub entry
				labels = append(labels, "sub:"+sub)
			}
		}
		sheet.SetRow(cell, labels)

		// need lookup label order
		labelIndex := make(map[string]int)
		for i, l := range labels {
			labelIndex[l.(string)] = i
		}

		row := 1
		for _, ent := range typeEnts {
			row++
			// thumbnail at first column
			thumb := thumbnail[ent.Path]
			if thumb != nil {
				thumbCell, err := excelize.CoordinatesToCellName(1, row)
				if err != nil {
					return err
				}
				err = xl.AddPictureFromBytes(typ, thumbCell, &excelize.Picture{
					Extension: ".png", // thumbnails in forge always saved as png
					File:      thumb.Data,
					// It has a bug on scaling images, let's keep our eyes on the new release of excelize.
					Format: &excelize.GraphicOptions{OffsetX: 1, OffsetY: 1, ScaleX: 0.33, ScaleY: 0.18},
				})
				if err != nil {
					return err
				}
			}

			rowData := make([]any, len(labels))
			// rowData[0] is just a place holder for thumbnail
			rowData[1] = filepath.Dir(ent.Path)
			rowData[2] = filepath.Base(ent.Path)
			for prop, p := range ent.Property {
				idx, ok := labelIndex[prop]
				if ok {
					rowData[idx] = p.Value
				}
			}

			// NOTE: export of sub-entry properties is for management convinience,
			// so data could be manipulated before it is written.
			for _, sub := range sortedSubEntries[ent.Type] {
				subEnt, ok := subEntry[ent.Path][sub]
				if !ok {
					continue
				}
				dirSub, restSub, _ := strings.Cut(sub, "/")
				subIdx := labelIndex["sub:"+dirSub]
				cellData := rowData[subIdx]
				data := ""
				if cellData != nil {
					data = cellData.(string) + "\n"
				}
				if restSub != "" {
					data += restSub + ": "
				}
				assignee := subEnt.Property["assignee"]
				if assignee == nil || assignee.Value == "" {
					data += "(assignee)"
				} else {
					data += assignee.Eval // yes, Eval.
				}
				data += "  "
				status := subEnt.Property["status"]
				if status == nil || status.Value == "" {
					data += "(none)"
				} else {
					data += status.Value
				}
				data += "  "
				due := subEnt.Property["due"]
				if due == nil || due.Value == "" {
					// would not add "(due)" as the cell looks cleaner
				} else {
					data += due.Value
				}
				rowData[subIdx] = data
			}
			cell, err := excelize.CoordinatesToCellName(2, row)
			if err != nil {
				return err
			}
			sheet.SetRow(cell, rowData[1:], excelize.RowOpts{Height: 54})
		}
		sheet.Flush()

		sheet_idx++
	}
	w.Header().Set("Content-Type", "application/vnd.ms-excel")
	w.Header().Set("Content-Disposition", "attachment;") // filename should be set in the client script.
	xl.WriteTo(w)
	return nil
}

func (h *pageHandler) handleBackupAsExcel(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	_, err := h.server.GetUser(ctx, user)
	if err != nil {
		return err
	}
	root := strings.TrimSpace(r.FormValue("root"))
	if root == "" {
		return fmt.Errorf("please specify root path to backup")
	}
	rootEnt, err := h.server.GetEntry(ctx, root)
	if err != nil {
		return err
	}
	ents, err := h.server.SearchEntries(ctx, root, "name:")
	if err != nil {
		return err
	}
	ents = append(ents, rootEnt)
	entsPerType := make(map[string][]*forge.Entry) // [type][]entry
	thumbnail := make(map[string]*forge.Thumbnail) // [path]thumbnail
	for _, ent := range ents {
		if entsPerType[ent.Type] == nil {
			entsPerType[ent.Type] = make([]*forge.Entry, 0)
		}
		entsPerType[ent.Type] = append(entsPerType[ent.Type], ent)
		thumb, err := h.server.GetThumbnail(ctx, ent.Path)
		if err != nil {
			var notFound *forge.NotFoundError
			if !errors.As(err, &notFound) {
				return err
			}
		}
		thumbnail[ent.Path] = thumb
	}
	for _, ents := range entsPerType {
		sort.Slice(ents, func(i, j int) bool {
			return strings.Compare(ents[i].Path, ents[j].Path) <= 0
		})
	}
	propsPerType := make(map[string][]string)
	for typ, ents := range entsPerType {
		for _, ent := range ents {
			// only need to check one per type
			props := []string{
				"thumbnail",
				"path",
				"env",
			}
			for _, p := range ent.Property {
				props = append(props, p.Name)
			}
			propsPerType[typ] = props
			break
		}
	}
	entryEnvs := make(map[string]string)
	for _, ent := range ents {
		var envs []*forge.Property
		if ent.Path == root {
			// should save inherited environ for backup root
			envs, err = h.server.EntryEnvirons(ctx, ent.Path)
			if err != nil {
				return err
			}
		} else {
			envs, err = h.server.GetEnvirons(ctx, ent.Path)
			if err != nil {
				return err
			}
		}
		environ := make([]string, 0, len(envs))
		for _, e := range envs {
			// give type to environs was a mistake, they will eventually be simple text
			val := strings.ReplaceAll(e.Eval, "\n", "\\n") // escape newline
			environ = append(environ, e.Name+"="+val)
		}
		entryEnvs[ent.Path] = strings.Join(environ, "\n")
	}

	entryAccessList := make(map[string]string)
	for _, ent := range ents {
		var accs []*forge.Access
		if ent.Path == root {
			// should save inherited accesses for backup root
			accs, err = h.server.EntryAccessList(ctx, ent.Path)
			if err != nil {
				return err
			}
		} else {
			accs, err = h.server.GetAccessList(ctx, ent.Path)
			if err != nil {
				return err
			}
		}
		accessList := make([]string, 0, len(accs))
		for _, a := range accs {
			// give type to environs was a mistake, they will eventually be simple text
			accessList = append(accessList, a.Name+"="+a.Value)
		}
		entryAccessList[ent.Path] = strings.Join(accessList, "\n")
	}
	xl := excelize.NewFile()
	sheet_idx := 0
	for typ, ents := range entsPerType {
		if sheet_idx == 0 {
			xl.SetSheetName("Sheet1", typ)
		} else {
			xl.NewSheet(typ)
		}
		sheet, err := xl.NewStreamWriter(typ)
		if err != nil {
			return err
		}
		sheet.SetColWidth(1, 1, 16) // thumbnail column width, 16 isn't sized in pixel.
		firstEnt := ents[0]
		props := make([]string, 0, len(firstEnt.Property))
		for prop := range firstEnt.Property {
			props = append(props, prop)
		}
		sort.Strings(props)
		labels := []any{
			"thumbnail",
			"path",
			"env",
			"access",
		}
		for _, prop := range props {
			labels = append(labels, prop)
		}
		cell, _ := excelize.CoordinatesToCellName(1, 1)
		sheet.SetRow(cell, labels)

		row := 2 // first row is for labels
		for _, ent := range ents {
			// thumbnail
			thumb := thumbnail[ent.Path]
			if thumb != nil {
				thumbCell, err := excelize.CoordinatesToCellName(1, row)
				if err != nil {
					return err
				}
				err = xl.AddPictureFromBytes(typ, thumbCell, &excelize.Picture{
					Extension: ".png", // thumbnails in forge always saved as png
					File:      thumb.Data,
					// It has a bug on scaling images, let's keep our eyes on the new release of excelize.
					Format: &excelize.GraphicOptions{OffsetX: 1, OffsetY: 1, ScaleX: 0.33, ScaleY: 0.18},
				})
				if err != nil {
					return err
				}
			}
			// props
			cell, err = excelize.CoordinatesToCellName(2, row)
			if err != nil {
				return err
			}
			rowData := make([]any, 0, len(props)+1)
			rowData = append(rowData, ent.Path)
			rowData = append(rowData, entryEnvs[ent.Path])
			rowData = append(rowData, entryAccessList[ent.Path])
			for _, prop := range props {
				p := ent.Property[prop]
				rowData = append(rowData, p.Value)
			}
			err = sheet.SetRow(cell, rowData, excelize.RowOpts{Height: 54})
			if err != nil {
				return err
			}
			row++
		}
		sheet.Flush()
		sheet_idx++
	}
	w.Header().Set("Content-Type", "application/vnd.ms-excel")
	w.Header().Set("Content-Disposition", "attachment;") // filename should be set in the client script.
	xl.WriteTo(w)
	return nil
}
