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

	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/tiff"
)

type apiHandler struct {
	server *forge.Server
	apps   *AppSessionManager
}

func (h *apiHandler) Handler(handleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := func() error {
			if r.Method != "POST" {
				return fmt.Errorf("need POST, got %v", r.Method)
			}
			var session map[string]string
			if r.FormValue("session") != "" {
				// app
				err := secureCookie.Decode("session", r.FormValue("session"), &session)
				if err != nil {
					return fmt.Errorf("please app-login")
				}
			} else {
				// browser
				s, err := getSession(r)
				if err != nil {
					clearSession(w)
					return fmt.Errorf("please login")
				}
				session = s
			}
			user := session["user"]
			ctx := forge.ContextWithUserName(r.Context(), user)
			return handleFunc(ctx, w, r)
		}()
		if err != nil {
			h.WriteResponse(w, nil, err)
		}
	}
}

func (h *apiHandler) WriteResponse(w http.ResponseWriter, m any, e error) {
	w.WriteHeader(httpStatusFromError(e))
	errStr := ""
	if e != nil {
		errStr = e.Error()
	}
	resp, _ := json.Marshal(forge.APIResponse{Msg: m, Err: errStr})
	_, err := w.Write(resp)
	if err != nil {
		log.Print(err)
	}
}

func (h *apiHandler) handleNotFound(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	h.WriteResponse(w, nil, fmt.Errorf("api not found: %s", r.URL))
	return nil
}

func (h *apiHandler) handleAppLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	key := r.FormValue("key")
	sess, err := h.apps.RecieveSession(key)
	h.WriteResponse(w, sess, err)
	return nil
}

func (h *apiHandler) handleGetBaseEntryTypes(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	types, err := h.server.FindBaseEntryTypes(ctx)
	h.WriteResponse(w, types, err)
	return nil
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
	newName := r.FormValue("new_name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.UpdateDefault(ctx, entType, ctg, name, &newName, &typ, &value)
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

func (h *apiHandler) handleGetGlobals(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entType := r.FormValue("entry_type")
	globals, err := h.server.Globals(ctx, entType)
	h.WriteResponse(w, globals, err)
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

func (h *apiHandler) handleSubEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	path := r.FormValue("path")
	ents, err := h.server.SubEntries(ctx, path)
	h.WriteResponse(w, ents, err)
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleParentEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	path := r.FormValue("path")
	ents, err := h.server.ParentEntries(ctx, path)
	h.WriteResponse(w, ents, err)
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleSearchEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	from := r.FormValue("from")
	// type is DEPRECATED, use q directly.
	typ := r.FormValue("type")
	q := r.FormValue("q")
	if typ != "" {
		q = "type=" + typ + " " + q
	}
	ents, err := h.server.SearchEntries(ctx, from, q)
	h.WriteResponse(w, ents, err)
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleGetEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	path := r.FormValue("path") // To parse multipart form.
	ent, err := h.server.GetEntry(ctx, path)
	h.WriteResponse(w, ent, err)
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleGetEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("")
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	ents := make([]*forge.Entry, 0)
	for _, pth := range entPaths {
		ent, err := h.server.GetEntry(ctx, pth)
		if err != nil {
			h.WriteResponse(w, nil, err)
		}
		ents = append(ents, ent)
	}
	h.WriteResponse(w, ents, nil)
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	entTypes := r.PostForm["type"]
	if len(entTypes) == 0 {
		return fmt.Errorf("type not defined")
	}
	if len(entTypes) != len(entPaths) {
		return fmt.Errorf("number of types not matched to paths")
	}
	for i, entPath := range entPaths {
		typ := entTypes[i]
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

func (h *apiHandler) handleArchiveEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	err := h.server.ArchiveEntry(ctx, entPath)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUnarchiveEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	err := h.server.UnarchiveEntry(ctx, entPath)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleDeleteEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	delFn := h.server.DeleteEntry
	recursive := r.FormValue("recursive")
	if recursive != "" {
		delFn = h.server.DeleteEntryRecursive
	}
	for _, pth := range entPaths {
		err := delFn(ctx, pth)
		if err != nil {
			return err
		}
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

	// originally I wanted to remove this nautilus drop-file header in js, but didn't succeed.
	// recent nautilus seems fix it, use it until we have this patch.
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "x-special/nautilus-clipboard\ncopy\n", "")
	value = strings.ReplaceAll(value, "file://", "")

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

func (h *apiHandler) handleAddEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		err := h.server.AddEnviron(ctx, pth, name, typ, value)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		err := h.server.UpdateEnviron(ctx, pth, name, value)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddOrUpdateEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		env, err := h.server.GetEnviron(ctx, pth, name)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
		}
		if env != nil {
			err := h.server.UpdateEnviron(ctx, pth, name, value)
			if err != nil {
				return err
			}
		} else {
			// bulk-addition only supports "text" environ, for now.
			err := h.server.AddEnviron(ctx, pth, name, "text", value)
			if err != nil {
				return err
			}
		}
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

func (h *apiHandler) handleEntryEnvirons(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	envs, err := h.server.EntryEnvirons(ctx, entPath)
	h.WriteResponse(w, envs, err)
	return nil
}

func (h *apiHandler) handleDeleteEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	generous := r.FormValue("generous") != ""
	for _, pth := range entPaths {
		if generous {
			_, err := h.server.GetEnviron(ctx, pth, name)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return err
				}
				// the environ doesn't exist, but it should be generous.
				// let's skip.
				continue
			}
		}
		err := h.server.DeleteEnviron(ctx, pth, name)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleAddAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	accessor := r.FormValue("name")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		err := h.server.AddAccess(ctx, pth, accessor, mode)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	accessor := r.FormValue("name")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		err := h.server.UpdateAccess(ctx, pth, accessor, mode)
		if err != nil {
			return err
		}
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
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		acl, err := h.server.GetAccess(ctx, pth, accessor)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
		}
		if acl != nil {
			err := h.server.UpdateAccess(ctx, pth, accessor, mode)
			if err != nil {
				return err
			}
		} else {
			err = h.server.AddAccess(ctx, pth, accessor, mode)
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
	acl, err := h.server.GetAccess(ctx, entPath, accessor)
	h.WriteResponse(w, acl, err)
	return nil
}

func (h *apiHandler) handleEntryAccessList(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	accs, err := h.server.EntryAccessList(ctx, entPath)
	h.WriteResponse(w, accs, err)
	return nil
}

func (h *apiHandler) handleDeleteAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	generous := r.FormValue("generous") != ""
	for _, pth := range entPaths {
		if generous {
			_, err := h.server.GetAccess(ctx, pth, name)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return err
				}
				// the access doesn't exist, but it should be generous.
				// let's skip.
				continue
			}
		}
		err := h.server.DeleteAccess(ctx, pth, name)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleGetPropertyHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	pth := r.FormValue("path")
	prop := r.FormValue("property")
	logs, err := h.server.GetLogs(ctx, pth, "property", prop)
	h.WriteResponse(w, logs, err)
	return nil
}

func (h *apiHandler) handleGetEnvironHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	pth := r.FormValue("path")
	env := r.FormValue("environ")
	logs, err := h.server.GetLogs(ctx, pth, "environ", env)
	h.WriteResponse(w, logs, err)
	return nil
}

func (h *apiHandler) handleGetAccessHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	pth := r.FormValue("path")
	acc := r.FormValue("access")
	logs, err := h.server.GetLogs(ctx, pth, "access", acc)
	h.WriteResponse(w, logs, err)
	return nil
}

func (h *apiHandler) handleGetAllGroups(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	grps, err := h.server.AllGroups(ctx)
	h.WriteResponse(w, grps, err)
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

func (h *apiHandler) handleGetGroupMembers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	group := r.FormValue("group")
	mems, err := h.server.GroupMembers(ctx, group)
	h.WriteResponse(w, mems, err)
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

func (h *apiHandler) handleGetThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	entPath := r.FormValue("path")
	thumb, err := h.server.GetThumbnail(ctx, entPath)
	h.WriteResponse(w, thumb, err)
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

func (h *apiHandler) handleGetSessionUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	u, err := h.server.GetUser(ctx, user)
	h.WriteResponse(w, u, err)
	return nil
}

func (h *apiHandler) handleGetAllUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	users, err := h.server.AllUsers(ctx)
	h.WriteResponse(w, users, err)
	return nil
}

func (h *apiHandler) handleGetActiveUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	users, err := h.server.ActiveUsers(ctx)
	h.WriteResponse(w, users, err)
	return nil
}

func (h *apiHandler) handleGetDisabledUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	users, err := h.server.DisabledUsers(ctx)
	h.WriteResponse(w, users, err)
	return nil
}

func (h *apiHandler) handleAddUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctxUser := forge.UserNameFromContext(ctx)
	isAdmin, err := h.server.IsAdmin(ctx, ctxUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("non-admin user cannot add a user: %v", ctxUser)
	}
	user := r.FormValue("user")
	called := r.FormValue("called")
	_, err = h.server.GetUser(ctx, user)
	if err == nil {
		return fmt.Errorf("user already exists: %v", user)
	}
	var e *forge.NotFoundError
	if !errors.As(err, &e) {
		return err
	}
	u := &forge.User{
		Name:   user,
		Called: called,
	}
	err = h.server.AddUser(ctx, u)
	if err != nil {
		return err
	}
	return nil
}

func (h *apiHandler) handleUpdateUserCalled(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := r.FormValue("user")
	ctxUser := forge.UserNameFromContext(ctx)
	isAdmin, err := h.server.IsAdmin(ctx, ctxUser)
	if err != nil {
		return err
	}
	if ctxUser != user && !isAdmin {
		return fmt.Errorf("non-admin user cannot update another user: %v", ctxUser)
	}
	called := r.FormValue("called")
	err = h.server.UpdateUserCalled(ctx, user, called)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleUpdateUserDisabled(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := r.FormValue("user")
	ctxUser := forge.UserNameFromContext(ctx)
	isAdmin, err := h.server.IsAdmin(ctx, ctxUser)
	if err != nil {
		return err
	}
	if ctxUser != user && !isAdmin {
		return fmt.Errorf("non-admin user cannot update another user: %v", ctxUser)
	}
	v := r.FormValue("disabled")
	disabled, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	err = h.server.UpdateUserDisabled(ctx, user, disabled)
	if err != nil {
		return err
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleGetUserSetting(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := r.FormValue("user")
	u, err := h.server.GetUserSetting(ctx, user)
	h.WriteResponse(w, u, err)
	return nil
}

func (h *apiHandler) handleUpdateUserSetting(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// NOTE: don't use make, maps not for the update should be nil
	if r.FormValue("update_entry_page_hide_side_menu") != "" {
		v := r.FormValue("hide")
		hide, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "entry_page_hide_side_menu", hide)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_entry_page_selected_category") != "" {
		selCategory := r.FormValue("category")
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_selected_category", selCategory)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_entry_page_show_hidden_property") != "" {
		showHidden := r.FormValue("show_hidden")
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_show_hidden_property", showHidden)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_entry_page_expand_property") != "" {
		v := r.FormValue("expand")
		expand, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "entry_page_expand_property", expand)
		if err != nil {
			return err
		}
	}
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
	if r.FormValue("update_picked_property") != "" {
		entryType := r.FormValue("entry_type")
		picked := r.FormValue("picked_property")
		pickedProperty := map[string]string{
			entryType: picked,
		}
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "picked_property", pickedProperty)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_picked_property_input_size") != "" {
		size := strings.TrimSpace(r.FormValue("size"))
		toks := strings.Split(size, "x")
		if len(toks) != 2 {
			return fmt.Errorf("size should be a {width}x{height} string: got %v", size)
		}
		// validate user input
		sx := strings.TrimSpace(toks[0])
		sy := strings.TrimSpace(toks[1])
		x, err := strconv.Atoi(sx)
		if err != nil {
			return fmt.Errorf("size should be a {width}x{height} string: got %v", size)
		}
		y, err := strconv.Atoi(sy)
		if err != nil {
			return fmt.Errorf("size should be a {width}x{height} string: got %v", size)
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "picked_property_input_size", [2]int{x, y})
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_update_marker_lasts") != "" {
		v := r.FormValue("update_marker_lasts")
		var last int
		var err error
		if v == "" {
			last = -1
		} else {
			last, err = strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("need integer value for update_marker_lasts: %v", v)
			}
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "update_marker_lasts", last)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_quick_search") != "" {
		name := r.FormValue("quick_search_name")
		val := r.FormValue("quick_search_value")
		at := r.FormValue("quick_search_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return fmt.Errorf("pinned_path_at cannot be converted to int: %v", at)
		}
		over := false
		if r.FormValue("quick_search_override") != "" {
			over = true
		}
		arr := forge.QuickSearchArranger{KV: forge.StringKV{K: name, V: val}, Index: n, Override: over}
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
		pinnedPath := forge.StringSliceArranger{
			Value: path,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "pinned_paths", pinnedPath)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_recent_paths") != "" {
		path := strings.TrimSpace(r.FormValue("path"))
		if path == "" {
			return fmt.Errorf("path not provided")
		}
		at := r.FormValue("path_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return fmt.Errorf("path_at cannot be converted to int: %v", at)
		}
		recentPath := forge.StringSliceArranger{
			Value: path,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "recent_paths", recentPath)
		h.WriteResponse(w, "", err)
		return err
	}
	if r.FormValue("update_programs_in_use") != "" {
		prog := strings.TrimSpace(r.FormValue("program"))
		if prog == "" {
			return fmt.Errorf("program not provided")
		}
		at := r.FormValue("program_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return fmt.Errorf("program_at cannot be converted to int: %v", at)
		}
		progsInUse := forge.StringSliceArranger{
			Value: prog,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "programs_in_use", progsInUse)
		h.WriteResponse(w, "", err)
		return err
	}
	if r.FormValue("update_search_view") != "" {
		view := strings.TrimSpace(r.FormValue("view"))
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "search_view", view)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_entry_group_by") != "" {
		groupBy := strings.TrimSpace(r.FormValue("group_by"))
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_group_by", groupBy)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_copy_path_remap") != "" {
		from := strings.TrimSpace(r.FormValue("from"))
		if strings.Contains(from, ";") {
			return fmt.Errorf("remap from path cannot have semicolon(;) in it")
		}
		to := strings.TrimSpace(r.FormValue("to"))
		if strings.Contains(to, ";") {
			return fmt.Errorf("remap to path cannot have semicolon(;) in it")
		}
		remap := from + ";" + to
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "copy_path_remap", remap)
		if err != nil {
			return err
		}
	}
	if r.FormValue("update_show_archived") != "" {
		val := strings.TrimSpace(r.FormValue("show"))
		show, _ := strconv.ParseBool(val)
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "show_archived", show)
		if err != nil {
			return err
		}
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
	return nil
}

func (h *apiHandler) handleEnsureUserDataSection(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	_, err := h.server.GetUserDataSection(ctx, user, section)
	if err == nil {
		h.WriteResponse(w, nil, nil)
		return nil
	}
	e := &forge.NotFoundError{}
	if !errors.As(err, &e) {
		h.WriteResponse(w, nil, err)
		return nil
	}
	err = h.server.AddUserDataSection(ctx, user, section)
	h.WriteResponse(w, nil, err)
	return nil
}

func (h *apiHandler) handleGetUserDataSection(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	sec, err := h.server.GetUserDataSection(ctx, user, section)
	h.WriteResponse(w, sec, err)
	return nil
}

func (h *apiHandler) handleDeleteUserDataSection(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	err := h.server.DeleteUserDataSection(ctx, user, section)
	h.WriteResponse(w, nil, err)
	return nil
}

func (h *apiHandler) handleSetUserData(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	key := r.FormValue("key")
	value := r.FormValue("value")
	err := h.server.SetUserData(ctx, user, section, key, value)
	h.WriteResponse(w, nil, err)
	return nil
}

func (h *apiHandler) handleDeleteUserData(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	key := r.FormValue("key")
	err := h.server.DeleteUserData(ctx, user, section, key)
	h.WriteResponse(w, nil, err)
	return nil
}

// handleCheckBulkUpdate checks what will happen when bulk update of entries processed from uploaded excel file.
//
// The result shows what entries will be added, and what entries will be updated in the following format.
// Entries not in any of the list is already match with given data.
// Msg is null when Err is not empty.
//
//	{
//		"Err": "",
//		"Msg": {
//			"add": {
//				"{entry-type}": ["{entry}", ...],
//				...
//			}
//			"update": {
//				"{entry-type}": ["{entry}", ...],
//				...
//			}
//		}
//	}
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
//	{
//		"Err": "",
//	}
//
// NOTE: With non-empty "dryrun" form-value, like "dryrun=1", it will perform handleDryRunBulkUpdate and return it's result instead.
func (h *apiHandler) handleBulkUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return err
	}
	defer file.Close()
	addMode := false
	if r.FormValue("add") != "" {
		addMode = true
	}
	xlr, err := excelize.OpenReader(file)
	if err != nil {
		return err
	}
	sheet := xlr.GetSheetList()[0]
	rows, err := xlr.GetRows(sheet)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("no data to update in the first sheet: %v", sheet)
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
	valueRows := rows[1:]
	for n, cols := range valueRows {
		if len(cols) == 0 {
			// The length can be zero when the row is entirely empty
			continue
		}
		row := n + 1 // Label row removed.
		vis, err := xlr.GetRowVisible(sheet, row+1)
		if err != nil {
			return err
		}
		if !vis {
			continue
		}
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
		name := cols[nameIdx]
		if name == "" {
			return fmt.Errorf("'name' field empty")
		}
		entPath := path.Clean(path.Join(parent, name))
		ent, err := h.server.GetEntry(ctx, entPath)
		if err != nil {
			e := &forge.NotFoundError{}
			if !errors.As(err, &e) {
				return err
			}
			if addMode {
				err := h.server.AddEntry(ctx, entPath, "")
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("add a new entry is not allowed in update mode: %s", entPath)
			}
			// To check entry type, need to get the entry.
			ent, err = h.server.GetEntry(ctx, entPath)
			if err != nil {
				return err
			}
		}
		// Update the entry's thumbnail.
		if thumbnailIdx != -1 {
			err := func() error {
				// Excel coordinate starts from 1.
				thumbCell, err := excelize.CoordinatesToCellName(thumbnailIdx+1, row+1)
				if err != nil {
					return err
				}
				thumbs, err := xlr.GetPictures(sheet, thumbCell)
				if err != nil {
					return err
				}
				if len(thumbs) == 0 {
					return nil
				}
				thumb := thumbs[0]
				r := bytes.NewBuffer(thumb.File)
				img, _, err := image.Decode(r)
				if err != nil {
					// Error on thumbnail parsing shouldn't interrupt whole update process.
					// This is an TEMPORARY fix.
					// TODO: think better way to handle these errors
					log.Printf("failed to decode image on %v/%v: %v\n", parent, name, err)
					return nil
				}
				if ent.HasThumbnail {
					orig, err := h.server.GetThumbnail(ctx, entPath)
					if err != nil {
						return err
					}
					if bytes.Equal(thumb.File, orig.Data) {
						return nil
					}
					return h.server.UpdateThumbnail(ctx, entPath, img)
				}
				return h.server.AddThumbnail(ctx, entPath, img)
			}()
			if err != nil {
				return err
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
				if plus {
					old, err := h.server.GetProperty(ctx, entPath, p)
					if err != nil {
						return err
					}
					if old.ValueError != nil {
						return fmt.Errorf("cannot evaluate the previous value: %v.%v", entPath, p)
					}
					oldv = old.Value
				}
			}
			v := strings.TrimSpace(val)
			if plus {
				if v != "" {
					propValue[p] = oldv + "\n" + v
				}
			} else {
				if seen {
					return fmt.Errorf("got label %q more than once, use %q if you want to combine them", p, p+"+")
				}
				propValue[p] = v
			}
		}
		for _, p := range props {
			v := strings.TrimSpace(propValue[p])
			upds = append(upds, forge.PropertyUpdater{
				EntryPath: entPath,
				Name:      p,
				Value:     &v,
			})

		}
		err = h.server.UpdateProperties(ctx, upds)
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
