package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/imagvfx/forge"
	"github.com/xuri/excelize/v2"
)

type apiHandler struct {
	server *forge.Server
}

func (h *apiHandler) Handler(handleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			if r.Method != "POST" {
				return fmt.Errorf("need POST, got %v", r.Method)
			}
			session, err := getSession(r)
			if err != nil {
				clearSession(w)
				return err
			}
			user := session["user"]
			ctx := forge.ContextWithUserName(r.Context(), user)
			return handleFunc(ctx, w, r)
		}()
		handleError(w, err)
	}
}

func (h *apiHandler) WriteResponse(w http.ResponseWriter, m interface{}, e error) {
	w.WriteHeader(httpStatusFromError(e))
	resp, _ := json.Marshal(forge.APIResponse{Msg: m, Err: e})
	_, err := w.Write(resp)
	if err != nil {
		log.Print(err)
	}
}

func (h *apiHandler) handleAddEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	name := r.FormValue("name")
	err := h.server.AddEntryType(ctx, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleRenameEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	name := r.FormValue("name")
	newName := r.FormValue("new_name")
	err := h.server.RenameEntryType(ctx, name, newName)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	name := r.FormValue("name")
	err := h.server.DeleteEntryType(ctx, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddDefault(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.AddDefault(ctx, entType, ctg, name, typ, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateDefault(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.UpdateDefault(ctx, entType, ctg, name, typ, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteDefault(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	err := h.server.DeleteDefault(ctx, entType, ctg, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddGlobal(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.AddGlobal(ctx, entType, name, typ, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateGlobal(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.UpdateGlobal(ctx, entType, name, typ, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteGlobal(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	name := r.FormValue("name")
	err := h.server.DeleteGlobal(ctx, entType, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleCountAllSubEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	path := r.FormValue("path")
	n, err := h.server.CountAllSubEntries(ctx, path)
	h.WriteResponse(w, n, err)
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// parent, if suggested, will be used as prefix of the path.
	parent := r.FormValue("parent")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	for _, n := range strings.Fields(name) {
		// treat seperate field a child name
		entPath := path.Join(parent, n)
		err := h.server.AddEntry(ctx, entPath, typ)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleRenameEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	newName := r.FormValue("new-name")
	err := h.server.RenameEntry(ctx, entPath, newName)
	if err != nil {
		return err
	}
	newPath := path.Dir(entPath) + "/" + newName
	if r.FormValue("back_to_referer") != "" {
		referer := strings.Replace(r.Header.Get("Referer"), entPath, newPath, 1)
		http.Redirect(w, r, referer, http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	delFn := h.server.DeleteEntry
	recursive := r.FormValue("recursive")
	if recursive != "" {
		delFn = h.server.DeleteEntryRecursive
	}
	err := delFn(ctx, entPath)
	if err != nil {
		return err
	}
	return nil
}

func (h *apiHandler) handleAddProperty(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	err := h.server.AddProperty(ctx, entPath, name, typ, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateProperty(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		err := h.server.UpdateProperty(ctx, pth, name, value)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleGetProperty(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	p, err := h.server.GetProperty(ctx, entPath, name)
	h.WriteResponse(w, p, err)
	return nil
}

func (h *apiHandler) handleDeleteProperty(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	err := h.server.DeleteProperty(ctx, entPath, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	err := h.server.AddEnviron(ctx, entPath, name, typ, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	err := h.server.UpdateEnviron(ctx, entPath, name, value)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleGetEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	env, err := h.server.GetEnviron(ctx, entPath, name)
	h.WriteResponse(w, env, err)
	return nil
}

func (h *apiHandler) handleDeleteEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	err := h.server.DeleteEnviron(ctx, entPath, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	accessor := r.FormValue("name")
	accessor_type := r.FormValue("type")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	err := h.server.AddAccessControl(ctx, entPath, accessor, accessor_type, mode)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	accessor := r.FormValue("name")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	err := h.server.UpdateAccessControl(ctx, entPath, accessor, mode)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddOrUpdateAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	accessor := r.FormValue("name")
	accessor_type := r.FormValue("type")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		acl, err := h.server.GetAccessControl(ctx, pth, accessor)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
		}
		if acl != nil {
			if accessor_type != "" && accessor_type != acl.AccessorType {
				return fmt.Errorf("accessor exists, but with different type: %v", acl.AccessorType)
			}
			err := h.server.UpdateAccessControl(ctx, pth, accessor, mode)
			if err != nil {
				return err
			}
		} else {
			err = h.server.AddAccessControl(ctx, pth, accessor, accessor_type, mode)
			if err != nil {
				return err
			}
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return nil
	}
	return nil
}

func (h *apiHandler) handleGetAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	accessor := r.FormValue("name")
	acl, err := h.server.GetAccessControl(ctx, entPath, accessor)
	h.WriteResponse(w, acl, err)
	return nil
}

func (h *apiHandler) handleDeleteAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	err := h.server.DeleteAccessControl(ctx, entPath, name)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddGroup(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	group := r.FormValue("group")
	g := &forge.Group{
		Name: group,
	}
	err := h.server.AddGroup(ctx, g)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleRenameGroup(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	group := r.FormValue("group")
	newName := r.FormValue("new-name")
	err := h.server.RenameGroup(ctx, group, newName)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddGroupMember(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	group := r.FormValue("group")
	member := r.FormValue("member")
	err := h.server.AddGroupMember(ctx, group, member)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteGroupMember(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	group := r.FormValue("group")
	member := r.FormValue("member")
	err := h.server.DeleteGroupMember(ctx, group, member)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	err = h.server.AddThumbnail(ctx, entPath, img)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}
	err = h.server.UpdateThumbnail(ctx, entPath, img)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	err := h.server.DeleteThumbnail(ctx, entPath)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateUserSetting(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// NOTE: don't use make, maps not for the update should be nil
	if r.FormValue("update_filter") != "" {
		entryType := r.FormValue("entry_page_entry_type")
		filter := r.FormValue("entry_page_property_filter")
		propertyFilter := map[string]string{
			entryType: filter,
		}
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_property_filter", propertyFilter)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_sort") != "" {
		entryType := r.FormValue("entry_page_entry_type")
		sortProp := r.FormValue("entry_page_sort_property") // sort by entry name if empty
		sortPrefix := "+"
		if r.FormValue("entry_page_sort_desc") != "" {
			sortPrefix = "-"
		}
		sortProperty := map[string]string{
			entryType: sortPrefix + sortProp,
		}
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_sort_property", sortProperty)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_quick_search") != "" {
		name := r.FormValue("quick_search_name")
		val := r.FormValue("quick_search_value")
		quickSearch := []forge.StringKV{
			{K: name, V: val},
		}
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "quick_searches", quickSearch)
		if err != nil {
			return err
		}
	}
	if r.FormValue("arrange_quick_search") != "" {
		name := r.FormValue("quick_search_name")
		at := r.FormValue("quick_search_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return fmt.Errorf("pinned_path_at cannot be converted to int: %v", at)
		}
		arr := forge.QuickSearchArranger{Name: name, Index: n}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "quick_searches", arr)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_pinned_path") != "" {
		path := strings.TrimSpace(r.FormValue("pinned_path"))
		if path == "" {
			return fmt.Errorf("pinned_path not provided")
		}
		at := r.FormValue("pinned_path_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return fmt.Errorf("pinned_path_at cannot be converted to int: %v", at)
		}
		pinnedPath := forge.PinnedPathArranger{
			Path:  path,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "pinned_paths", pinnedPath)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

// handleCheckBulkUpdate checks what will happen when bulk update of entries processed from uploaded excel file.
//
// The result shows what entries will be added, and what entries will be updated in the following format.
// Entries not in any of the list is already match with given data.
// Msg is null when Err is not empty.
//
// {
// 	"Err": "",
// 	"Msg": {
// 		"add": {
// 			"{entry-type}": ["{entry}", ...],
// 			...
// 		}
// 		"update": {
// 			"{entry-type}": ["{entry}", ...],
// 			...
// 		}
// 	}
// }
//
func (h *apiHandler) handleDryRunBulkUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// TODO
	return nil
}

// handleBulkUpdate adds and/or updates multiple entries at once from uploaded excel file.
//
// First row of the file should hold label of columns.
// It needs 2 special columns, 'parent' and 'name'. They consist of full path of the entry, and cannot be empty.
// Other column represents a property of entry. If the property does not exist in the entry type, it will be skipped.
//
// The result shows an error if it happened during the process.
//
// {
// 	"Err": "",
// }
//
// NOTE: With non-empty "dryrun" form-value, like "dryrun=1", it will perform handleDryRunBulkUpdate and return it's result instead.
//
func (h *apiHandler) handleBulkUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()
	xlr, err := excelize.OpenReader(file)
	if err != nil {
		return err
	}
	sheet := xlr.GetSheetList()[0]
	rows, err := xlr.GetRows(sheet)
	if err != nil {
		return err
	}
	propFor := make(map[int]string)
	labelRow := rows[0]
	nameIdx := -1
	parentIdx := -1
	thumbnailIdx := -1
	for i, p := range labelRow {
		if p == "name" {
			nameIdx = i
			continue
		}
		if p == "parent" {
			parentIdx = i
			continue
		}
		if p == "thumbnail" {
			thumbnailIdx = i
			continue
		}
		propFor[i] = p
	}
	if nameIdx == -1 {
		return fmt.Errorf("'name' field not found")
	}
	if parentIdx == -1 {
		return fmt.Errorf("'parent' field not found")
	}
	// To check the ancestor with db only once.
	knownAncestor := make(map[string]bool)
	valueRows := rows[1:]
	for n, cols := range valueRows {
		if len(cols) == 0 {
			// The length can be zero when the row is entirely empty
			continue
		}
		row := n + 1 // Label row removed.
		// Create the entry.
		parent := cols[parentIdx]
		if parent == "" {
			return fmt.Errorf("'parent' field empty")
		}
		if !path.IsAbs(parent) {
			return fmt.Errorf("'parent' field should be abs path: %v", parent)
		}
		parent = path.Clean(parent)
		if parent == "/" {
			// It's uncommon to create a child of root from bulk update.
			// When it happens it will be very hard to restore.
			// Maybe it wouldn't sufficient to prevent disaster, but better than nothing.
			return fmt.Errorf("cannot create a direct child of root from bulk update")
		}
		ancestors := make([]string, 0)
		anc := ""
		for _, p := range strings.Split(parent, "/")[1:] {
			anc += "/" + p
			ancestors = append(ancestors, anc)
		}
		for _, anc := range ancestors {
			if knownAncestor[anc] {
				continue
			}
			_, err = h.server.GetEntry(ctx, anc)
			if err != nil {
				e := &forge.NotFoundError{}
				if !errors.As(err, &e) {
					return err
				}
				err := h.server.AddEntry(ctx, anc, "")
				if err != nil {
					return err
				}
			}
			knownAncestor[anc] = true
		}
		name := cols[nameIdx]
		if name == "" {
			return fmt.Errorf("'name' field empty")
		}
		_, err := h.server.GetEntry(ctx, parent)
		if err != nil {
			// parent should exist already.
			return err
		}
		entPath := path.Clean(path.Join(parent, name))
		ent, err := h.server.GetEntry(ctx, entPath)
		if err != nil {
			e := &forge.NotFoundError{}
			if !errors.As(err, &e) {
				return err
			}
			err := h.server.AddEntry(ctx, entPath, "")
			if err != nil {
				return err
			}
			// To check entry type, need to get the entry.
			ent, err = h.server.GetEntry(ctx, entPath)
			if err != nil {
				return err
			}
		}
		// Update the entry's thumbnail.
		if thumbnailIdx != -1 {
			// Excel coordinate starts from 1.
			thumbCell, err := excelize.CoordinatesToCellName(thumbnailIdx+1, row+1)
			if err != nil {
				return err
			}
			_, thumb, err := xlr.GetPicture(sheet, thumbCell)
			if err != nil {
				return err
			}
			thumbReader := bytes.NewBuffer(thumb)
			img, _, err := image.Decode(thumbReader)
			if err != nil {
				return err
			}
			if ent.HasThumbnail {
				err = h.server.UpdateThumbnail(ctx, entPath, img)
				if err != nil {
					return err
				}
			} else {
				err = h.server.AddThumbnail(ctx, entPath, img)
				if err != nil {
					return err
				}
			}
		}
		// Update the entry's properties.
		defs, err := h.server.Defaults(ctx, ent.Type)
		if err != nil {
			return err
		}
		defaultProp := make(map[string]bool)
		for _, d := range defs {
			if d.Category != "property" {
				continue
			}
			defaultProp[d.Name] = true
		}
		props := make([]string, 0)
		propValue := make(map[string]string)
		upds := make([]forge.PropertyUpdater, 0)
		for i, val := range cols {
			p := propFor[i]
			if p == "" {
				// empty or non-prop label like 'name'
				continue
			}
			plus := false
			if strings.HasSuffix(p, "+") {
				plus = true
				p = p[:len(p)-1]
			}
			if !defaultProp[p] {
				continue
			}
			oldv, seen := propValue[p]
			if !seen {
				props = append(props, p)
			}
			v := strings.TrimSpace(val)
			if plus {
				if v != "" {
					propValue[p] = oldv + "\n\n\n" + v
				}
			} else {
				if seen {
					return fmt.Errorf("got label %q more than once, use %q if you want to combine them", p, p+"+")
				}
				propValue[p] = v
			}
		}
		for _, p := range props {
			v := propValue[p]
			upds = append(upds, forge.PropertyUpdater{
				EntryPath: entPath,
				Name:      p,
				Value:     &v,
			})

		}
		err = h.server.BulkUpdateProperties(ctx, upds)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return nil
	}
	return nil
}
