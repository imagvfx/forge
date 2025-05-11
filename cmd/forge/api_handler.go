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

func (h *apiHandler) Handler(handleFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := func() (any, error) {
			if r.Method != "POST" {
				return nil, fmt.Errorf("need POST, got %v", r.Method)
			}
			var session map[string]string
			if r.FormValue("session") != "" {
				// app
				err := secureCookie.Decode("session", r.FormValue("session"), &session)
				if err != nil {
					return nil, fmt.Errorf("please app-login")
				}
			} else {
				// browser
				s, err := getSession(r)
				if err != nil {
					clearSession(w)
					return nil, fmt.Errorf("please login")
				}
				session = s
			}
			user := session["user"]
			ctx := forge.ContextWithUserName(r.Context(), user)
			return handleFunc(ctx, w, r)
		}()
		h.WriteResponse(w, msg, err)
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

func (h *apiHandler) handleNotFound(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	return nil, fmt.Errorf("api not found: %s", r.URL)
}

func (h *apiHandler) handleAppLogin(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	key := r.FormValue("key")
	return h.apps.RecieveSession(key)
}

func (h *apiHandler) handleGetBaseEntryTypes(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	return h.server.FindBaseEntryTypes(ctx)
}

func (h *apiHandler) handleAddEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	name := r.FormValue("name")
	err := h.server.AddEntryType(ctx, name)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *apiHandler) handleRenameEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	name := r.FormValue("name")
	newName := r.FormValue("new_name")
	err := h.server.RenameEntryType(ctx, name, newName)
	return nil, err
}

func (h *apiHandler) handleDeleteEntryType(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	name := r.FormValue("name")
	err := h.server.DeleteEntryType(ctx, name)
	return nil, err
}

func (h *apiHandler) handleAddDefault(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.AddDefault(ctx, entType, ctg, name, typ, value)
	return nil, err
}

func (h *apiHandler) handleUpdateDefault(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	newName := r.FormValue("new_name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.UpdateDefault(ctx, entType, ctg, name, &newName, &typ, &value)
	return nil, err
}

func (h *apiHandler) handleDeleteDefault(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	ctg := r.FormValue("category")
	name := r.FormValue("name")
	err := h.server.DeleteDefault(ctx, entType, ctg, name)
	return nil, err
}

func (h *apiHandler) handleGetGlobals(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	return h.server.Globals(ctx, entType)
}

func (h *apiHandler) handleAddGlobal(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.AddGlobal(ctx, entType, name, typ, value)
	return nil, err
}

func (h *apiHandler) handleUpdateGlobal(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	err := h.server.UpdateGlobal(ctx, entType, name, typ, value)
	return nil, err
}

func (h *apiHandler) handleDeleteGlobal(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entType := r.FormValue("entry_type")
	name := r.FormValue("name")
	err := h.server.DeleteGlobal(ctx, entType, name)
	return nil, err
}

func (h *apiHandler) handleCountAllSubEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	path := r.FormValue("path")
	return h.server.CountAllSubEntries(ctx, path)
}

func (h *apiHandler) handleSubEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	path := r.FormValue("path")
	return h.server.SubEntries(ctx, path)
}

func (h *apiHandler) handleParentEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	path := r.FormValue("path")
	return h.server.ParentEntries(ctx, path)
}

func (h *apiHandler) handleSearchEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	from := r.FormValue("from")
	// type is DEPRECATED, use q directly.
	typ := r.FormValue("type")
	q := r.FormValue("q")
	if typ != "" {
		q = "type=" + typ + " " + q
	}
	return h.server.SearchEntries(ctx, from, q)
}

func (h *apiHandler) handleGetEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	path := r.FormValue("path") // To parse multipart form.
	return h.server.GetEntry(ctx, path)
}

func (h *apiHandler) handleGetEntries(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("")
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	ents := make([]*forge.Entry, 0)
	for _, pth := range entPaths {
		ent, err := h.server.GetEntry(ctx, pth)
		if err != nil {
			return nil, err
		}
		ents = append(ents, ent)
	}
	return ents, nil
}

func (h *apiHandler) handleAddEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	entTypes := r.PostForm["type"]
	if len(entTypes) == 0 {
		return nil, fmt.Errorf("type not defined")
	}
	if len(entTypes) != len(entPaths) {
		return nil, fmt.Errorf("number of types not matched to paths")
	}
	for i, entPath := range entPaths {
		typ := entTypes[i]
		err := h.server.AddEntry(ctx, entPath, typ)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleRenameEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	newName := r.FormValue("new-name")
	err := h.server.RenameEntry(ctx, entPath, newName)
	return nil, err
}

func (h *apiHandler) handleArchiveEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	err := h.server.ArchiveEntry(ctx, entPath)
	return nil, err
}

func (h *apiHandler) handleUnarchiveEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	err := h.server.UnarchiveEntry(ctx, entPath)
	return nil, err
}

func (h *apiHandler) handleDeleteEntry(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	delFn := h.server.DeleteEntry
	recursive := r.FormValue("recursive")
	if recursive != "" {
		delFn = h.server.DeleteEntryRecursive
	}
	for _, pth := range entPaths {
		err := delFn(ctx, pth)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleUpdateProperty(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
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
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleGetProperty(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	return h.server.GetProperty(ctx, entPath, name)
}

func (h *apiHandler) handleGetProperties(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	names := r.PostForm["name"]
	if len(names) == 0 {
		return nil, fmt.Errorf("name not defined")
	}
	if len(entPaths) != len(names) {
		return nil, fmt.Errorf("number of paths and names should be matched")
	}
	ps := make([]*forge.Property, 0, len(entPaths))
	for i := range entPaths {
		entPath := entPaths[i]
		name := names[i]
		p, err := h.server.GetProperty(ctx, entPath, name)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func (h *apiHandler) handleAddEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	typ := r.FormValue("type")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		err := h.server.AddEnviron(ctx, pth, name, typ, value)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleUpdateEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		err := h.server.UpdateEnviron(ctx, pth, name, value)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleAddOrUpdateEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	value := r.FormValue("value")
	value = strings.TrimSpace(value)
	for _, pth := range entPaths {
		env, err := h.server.GetEnviron(ctx, pth, name)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return nil, err
			}
		}
		if env != nil {
			err := h.server.UpdateEnviron(ctx, pth, name, value)
			if err != nil {
				return nil, err
			}
		} else {
			// bulk-addition only supports "text" environ, for now.
			err := h.server.AddEnviron(ctx, pth, name, "text", value)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (h *apiHandler) handleGetEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	name := r.FormValue("name")
	return h.server.GetEnviron(ctx, entPath, name)
}

func (h *apiHandler) handleEntryEnvirons(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	return h.server.EntryEnvirons(ctx, entPath)
}

func (h *apiHandler) handleDeleteEnviron(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	generous := r.FormValue("generous") != ""
	for _, pth := range entPaths {
		if generous {
			_, err := h.server.GetEnviron(ctx, pth, name)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return nil, err
				}
				// the environ doesn't exist, but it should be generous.
				// let's skip.
				continue
			}
		}
		err := h.server.DeleteEnviron(ctx, pth, name)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleAddAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	accessor := r.FormValue("name")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		err := h.server.AddAccess(ctx, pth, accessor, mode)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleUpdateAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	accessor := r.FormValue("name")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		err := h.server.UpdateAccess(ctx, pth, accessor, mode)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleAddOrUpdateAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	accessor := r.FormValue("name")
	mode := r.FormValue("value")
	mode = strings.TrimSpace(mode)
	for _, pth := range entPaths {
		acl, err := h.server.GetAccess(ctx, pth, accessor)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return nil, err
			}
		}
		if acl != nil {
			err := h.server.UpdateAccess(ctx, pth, accessor, mode)
			if err != nil {
				return nil, err
			}
		} else {
			err = h.server.AddAccess(ctx, pth, accessor, mode)
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func (h *apiHandler) handleGetAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	accessor := r.FormValue("name")
	return h.server.GetAccess(ctx, entPath, accessor)
}

func (h *apiHandler) handleEntryAccessList(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	return h.server.EntryAccessList(ctx, entPath)
}

func (h *apiHandler) handleDeleteAccess(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	r.FormValue("") // To parse multipart form.
	entPaths := r.PostForm["path"]
	if len(entPaths) == 0 {
		return nil, fmt.Errorf("path not defined")
	}
	name := r.FormValue("name")
	generous := r.FormValue("generous") != ""
	for _, pth := range entPaths {
		if generous {
			_, err := h.server.GetAccess(ctx, pth, name)
			if err != nil {
				var e *forge.NotFoundError
				if !errors.As(err, &e) {
					return nil, err
				}
				// the access doesn't exist, but it should be generous.
				// let's skip.
				continue
			}
		}
		err := h.server.DeleteAccess(ctx, pth, name)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (h *apiHandler) handleGetPropertyHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	pth := r.FormValue("path")
	prop := r.FormValue("property")
	return h.server.GetLogs(ctx, pth, "property", prop)
}

func (h *apiHandler) handleGetEnvironHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	pth := r.FormValue("path")
	env := r.FormValue("environ")
	return h.server.GetLogs(ctx, pth, "environ", env)
}

func (h *apiHandler) handleGetAccessHistory(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	pth := r.FormValue("path")
	acc := r.FormValue("access")
	return h.server.GetLogs(ctx, pth, "access", acc)
}

func (h *apiHandler) handleGetAllGroups(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	return h.server.AllGroups(ctx)
}

func (h *apiHandler) handleAddGroup(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	group := r.FormValue("group")
	g := &forge.Group{
		Name: group,
	}
	err := h.server.AddGroup(ctx, g)
	return nil, err
}

func (h *apiHandler) handleRenameGroup(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	group := r.FormValue("group")
	newName := r.FormValue("new-name")
	err := h.server.RenameGroup(ctx, group, newName)
	return nil, err
}

func (h *apiHandler) handleGetGroupMembers(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	group := r.FormValue("group")
	return h.server.GroupMembers(ctx, group)
}

func (h *apiHandler) handleAddGroupMember(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	group := r.FormValue("group")
	member := r.FormValue("member")
	err := h.server.AddGroupMember(ctx, group, member)
	return nil, err
}

func (h *apiHandler) handleDeleteGroupMember(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	group := r.FormValue("group")
	member := r.FormValue("member")
	err := h.server.DeleteGroupMember(ctx, group, member)
	return nil, err
}

func (h *apiHandler) handleAddThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	err = h.server.AddThumbnail(ctx, entPath, img)
	return nil, err
}

func (h *apiHandler) handleGetThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	return h.server.GetThumbnail(ctx, entPath)
}

func (h *apiHandler) handleUpdateThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	err = h.server.UpdateThumbnail(ctx, entPath, img)
	return nil, err
}

func (h *apiHandler) handleDeleteThumbnail(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	entPath := r.FormValue("path")
	err := h.server.DeleteThumbnail(ctx, entPath)
	return nil, err
}

func (h *apiHandler) handleGetSessionUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := forge.UserNameFromContext(ctx)
	return h.server.GetUser(ctx, user)
}

func (h *apiHandler) handleTestSession(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	// reaching here is the evidence of success.
	return true, nil
}

func (h *apiHandler) handleGetAllUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	return h.server.AllUsers(ctx)
}

func (h *apiHandler) handleGetActiveUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	return h.server.ActiveUsers(ctx)
}

func (h *apiHandler) handleGetDisabledUsers(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	return h.server.DisabledUsers(ctx)
}

func (h *apiHandler) handleAddUser(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	ctxUser := forge.UserNameFromContext(ctx)
	isAdmin, err := h.server.IsAdmin(ctx, ctxUser)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, fmt.Errorf("non-admin user cannot add a user: %v", ctxUser)
	}
	user := r.FormValue("user")
	called := r.FormValue("called")
	_, err = h.server.GetUser(ctx, user)
	if err == nil {
		return nil, fmt.Errorf("user already exists: %v", user)
	}
	var e *forge.NotFoundError
	if !errors.As(err, &e) {
		return nil, err
	}
	u := &forge.User{
		Name:   user,
		Called: called,
	}
	err = h.server.AddUser(ctx, u)
	return nil, err
}

func (h *apiHandler) handleUpdateUserCalled(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := r.FormValue("user")
	ctxUser := forge.UserNameFromContext(ctx)
	isAdmin, err := h.server.IsAdmin(ctx, ctxUser)
	if err != nil {
		return nil, err
	}
	if ctxUser != user && !isAdmin {
		return nil, fmt.Errorf("non-admin user cannot update another user: %v", ctxUser)
	}
	called := r.FormValue("called")
	err = h.server.UpdateUserCalled(ctx, user, called)
	return nil, err
}

func (h *apiHandler) handleUpdateUserDisabled(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := r.FormValue("user")
	ctxUser := forge.UserNameFromContext(ctx)
	isAdmin, err := h.server.IsAdmin(ctx, ctxUser)
	if err != nil {
		return nil, err
	}
	if ctxUser != user && !isAdmin {
		return nil, fmt.Errorf("non-admin user cannot update another user: %v", ctxUser)
	}
	v := r.FormValue("disabled")
	disabled, err := strconv.ParseBool(v)
	if err != nil {
		return nil, err
	}
	err = h.server.UpdateUserDisabled(ctx, user, disabled)
	return nil, err
}

func (h *apiHandler) handleGetUserSetting(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := r.FormValue("user")
	return h.server.GetUserSetting(ctx, user)
}

func (h *apiHandler) handleUpdateUserSetting(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	// NOTE: don't use make, maps not for the update should be nil
	if r.FormValue("update_entry_page_hide_side_menu") != "" {
		v := r.FormValue("hide")
		hide, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "entry_page_hide_side_menu", hide)
		return nil, err
	}
	if r.FormValue("update_entry_page_selected_category") != "" {
		selCategory := r.FormValue("category")
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_selected_category", selCategory)
		return nil, err
	}
	if r.FormValue("update_entry_page_show_hidden_property") != "" {
		showHidden := r.FormValue("show_hidden")
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_show_hidden_property", showHidden)
		return nil, err
	}
	if r.FormValue("update_entry_page_expand_property") != "" {
		v := r.FormValue("expand")
		expand, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "entry_page_expand_property", expand)
		return nil, err
	}
	if r.FormValue("update_filter") != "" {
		entryType := r.FormValue("entry_page_entry_type")
		filter := r.FormValue("entry_page_property_filter")
		propertyFilter := map[string]string{
			entryType: filter,
		}
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_page_property_filter", propertyFilter)
		return nil, err
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
		return nil, err
	}
	if r.FormValue("update_picked_property") != "" {
		entryType := r.FormValue("entry_type")
		picked := r.FormValue("picked_property")
		pickedProperty := map[string]string{
			entryType: picked,
		}
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "picked_property", pickedProperty)
		return nil, err
	}
	if r.FormValue("update_picked_property_input_size") != "" {
		size := strings.TrimSpace(r.FormValue("size"))
		toks := strings.Split(size, "x")
		if len(toks) != 2 {
			return nil, fmt.Errorf("size should be a {width}x{height} string: got %v", size)
		}
		// validate user input
		sx := strings.TrimSpace(toks[0])
		sy := strings.TrimSpace(toks[1])
		x, err := strconv.Atoi(sx)
		if err != nil {
			return nil, fmt.Errorf("size should be a {width}x{height} string: got %v", size)
		}
		y, err := strconv.Atoi(sy)
		if err != nil {
			return nil, fmt.Errorf("size should be a {width}x{height} string: got %v", size)
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "picked_property_input_size", [2]int{x, y})
		return nil, err
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
				return nil, fmt.Errorf("need integer value for update_marker_lasts: %v", v)
			}
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "update_marker_lasts", last)
		return nil, err
	}
	if r.FormValue("update_quick_search") != "" {
		name := r.FormValue("quick_search_name")
		val := r.FormValue("quick_search_value")
		at := r.FormValue("quick_search_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return nil, fmt.Errorf("pinned_path_at cannot be converted to int: %v", at)
		}
		over := false
		if r.FormValue("quick_search_override") != "" {
			over = true
		}
		arr := forge.QuickSearchArranger{KV: forge.StringKV{K: name, V: val}, Index: n, Override: over}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "quick_searches", arr)
		return nil, err
	}
	if r.FormValue("update_pinned_path") != "" {
		path := strings.TrimSpace(r.FormValue("pinned_path"))
		if path == "" {
			return nil, fmt.Errorf("pinned_path not provided")
		}
		at := r.FormValue("pinned_path_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return nil, fmt.Errorf("pinned_path_at cannot be converted to int: %v", at)
		}
		pinnedPath := forge.StringSliceArranger{
			Value: path,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "pinned_paths", pinnedPath)
		return nil, err
	}
	if r.FormValue("update_recent_paths") != "" {
		path := strings.TrimSpace(r.FormValue("path"))
		if path == "" {
			return nil, fmt.Errorf("path not provided")
		}
		at := r.FormValue("path_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return nil, fmt.Errorf("path_at cannot be converted to int: %v", at)
		}
		recentPath := forge.StringSliceArranger{
			Value: path,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "recent_paths", recentPath)
		return nil, err
	}
	if r.FormValue("update_programs_in_use") != "" {
		prog := strings.TrimSpace(r.FormValue("program"))
		if prog == "" {
			return nil, fmt.Errorf("program not provided")
		}
		at := r.FormValue("program_at")
		n, err := strconv.Atoi(at)
		if err != nil {
			return nil, fmt.Errorf("program_at cannot be converted to int: %v", at)
		}
		progsInUse := forge.StringSliceArranger{
			Value: prog,
			Index: n,
		}
		user := forge.UserNameFromContext(ctx)
		err = h.server.UpdateUserSetting(ctx, user, "programs_in_use", progsInUse)
		return nil, err
	}
	if r.FormValue("update_search_view") != "" {
		view := strings.TrimSpace(r.FormValue("view"))
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "search_view", view)
		return nil, err
	}
	if r.FormValue("update_entry_group_by") != "" {
		groupBy := strings.TrimSpace(r.FormValue("group_by"))
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "entry_group_by", groupBy)
		return nil, err
	}
	if r.FormValue("update_copy_path_remap") != "" {
		from := strings.TrimSpace(r.FormValue("from"))
		if strings.Contains(from, ";") {
			return nil, fmt.Errorf("remap from path cannot have semicolon(;) in it")
		}
		to := strings.TrimSpace(r.FormValue("to"))
		if strings.Contains(to, ";") {
			return nil, fmt.Errorf("remap to path cannot have semicolon(;) in it")
		}
		remap := from + ";" + to
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "copy_path_remap", remap)
		return nil, err
	}
	if r.FormValue("update_show_archived") != "" {
		val := strings.TrimSpace(r.FormValue("show"))
		show, _ := strconv.ParseBool(val)
		user := forge.UserNameFromContext(ctx)
		err := h.server.UpdateUserSetting(ctx, user, "show_archived", show)
		return nil, err
	}
	return nil, fmt.Errorf("no valid user-setting value found")
}

func (h *apiHandler) handleEnsureUserDataSection(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	_, err := h.server.GetUserDataSection(ctx, user, section)
	if err == nil {
		return nil, nil
	}
	e := &forge.NotFoundError{}
	if !errors.As(err, &e) {
		return nil, err
	}
	err = h.server.AddUserDataSection(ctx, user, section)
	return nil, err
}

func (h *apiHandler) handleGetUserDataSection(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	return h.server.GetUserDataSection(ctx, user, section)
}

func (h *apiHandler) handleDeleteUserDataSection(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	err := h.server.DeleteUserDataSection(ctx, user, section)
	return nil, err
}

func (h *apiHandler) handleSetUserData(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	key := r.FormValue("key")
	value := r.FormValue("value")
	err := h.server.SetUserData(ctx, user, section, key, value)
	return nil, err
}

func (h *apiHandler) handleDeleteUserData(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	user := forge.UserNameFromContext(ctx)
	section := r.FormValue("section")
	key := r.FormValue("key")
	err := h.server.DeleteUserData(ctx, user, section, key)
	return nil, err
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
func (h *apiHandler) handleDryRunBulkUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	// TODO
	return nil, nil
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
func (h *apiHandler) handleBulkUpdate(ctx context.Context, w http.ResponseWriter, r *http.Request) (any, error) {
	KiB := int64(1 << 10)
	r.ParseMultipartForm(100 * KiB) // 100KiB buffer size
	file, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	addMode := false
	if r.FormValue("add") != "" {
		addMode = true
	}
	xlr, err := excelize.OpenReader(file)
	if err != nil {
		return nil, err
	}
	// TODO: I'd like to get the currently opened sheet, but ActiveSheetIndex doesn't return it.
	sheet := ""
	for _, sh := range xlr.GetSheetList() {
		vis, err := xlr.GetSheetVisible(sh)
		if err != nil {
			return nil, err
		}
		if vis {
			sheet = sh
			break
		}
	}
	if sheet == "" {
		return nil, fmt.Errorf("no sheet available")
	}
	rows, err := xlr.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no data to update in sheet %q", sheet)
	}
	propFor := make(map[int]string)
	labelRow := rows[0]
	nameIdx := -1
	idIdx := -1
	idProp := ""
	parentIdx := -1
	subIdx := -1
	thumbnailIdx := -1
	for i, p := range labelRow {
		if p == "name" {
			nameIdx = i
			continue
		}
		if p == "id" {
			return nil, fmt.Errorf("invalid 'id' field formatting in %q sheet: need id={property}, eg) id=shotcode", sheet)
		}
		if strings.HasPrefix(p, "id=") {
			idIdx = i
			idProp = strings.TrimPrefix(p, "id=")
			continue
		}
		if p == "parent" {
			parentIdx = i
			continue
		}
		if p == "sub" {
			subIdx = i
			continue
		}
		if p == "thumbnail" {
			thumbnailIdx = i
			continue
		}
		propFor[i] = p
	}
	if nameIdx == -1 && idIdx == -1 {
		return nil, fmt.Errorf("neither 'name' nor 'id' field exists in %q sheet", sheet)
	}
	if nameIdx != -1 && idIdx != -1 {
		return nil, fmt.Errorf("should not use both 'name' and 'id' field at a same time in %q sheet", sheet)
	}
	if parentIdx == -1 {
		return nil, fmt.Errorf("'parent' field not found in %q sheet", sheet)
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
			return nil, err
		}
		if !vis {
			continue
		}
		// Create the entry.
		parent := cols[parentIdx]
		if parent == "" {
			return nil, fmt.Errorf("row %s of %q field empty in %q sheet", strconv.Itoa(row+1), "parent", sheet)
		}
		if !path.IsAbs(parent) {
			return nil, fmt.Errorf("row %s of %q field should be abs path in %q sheet: got %v", strconv.Itoa(row+1), "parent", sheet, parent)
		}
		parent = path.Clean(parent)
		if parent == "/" {
			// It's uncommon to create a child of root from bulk update.
			// When it happens it will be very hard to restore.
			// Maybe it wouldn't sufficient to prevent disaster, but better than nothing.
			return nil, fmt.Errorf("cannot create a direct child of root from bulk update")
		}
		entPath := ""
		if nameIdx != -1 {
			name := cols[nameIdx]
			if name == "" {
				return nil, fmt.Errorf("row %s of 'name' field empty in %q sheet", strconv.Itoa(row+1), sheet)
			}
			entPath = path.Clean(path.Join(parent, name))
		} else {
			// idIdx should not be -1.
			id := cols[idIdx]
			if id == "" {
				return nil, fmt.Errorf("row %s of 'id' field empty in %q sheet", strconv.Itoa(row+1), sheet)
			}
			ents, err := h.server.SearchEntries(ctx, parent, idProp+"="+id)
			if err != nil {
				return nil, fmt.Errorf("searching %q from %q in %q sheet: %v", idProp+"="+id, parent, sheet, err)
			}
			if len(ents) == 0 {
				return nil, fmt.Errorf("searching %q from %q in %q sheet: not found entry", idProp+"="+id, parent, sheet)
			}
			if len(ents) > 1 {
				return nil, fmt.Errorf("searching %q from %q in %q sheet: found multiple entries", idProp+"="+id, parent, sheet)
			}
			entPath = ents[0].Path
		}
		if subIdx != -1 {
			sub := cols[subIdx]
			entPath = path.Join(entPath, sub)
		}
		ent, err := h.server.GetEntry(ctx, entPath)
		if err != nil {
			e := &forge.NotFoundError{}
			if !errors.As(err, &e) {
				return nil, err
			}
			if addMode {
				err := h.server.AddEntry(ctx, entPath, "")
				if err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("add a new entry is not allowed in update mode: %s", entPath)
			}
			// To check entry type, need to get the entry.
			ent, err = h.server.GetEntry(ctx, entPath)
			if err != nil {
				return nil, err
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
					log.Printf("failed to decode image on %v: %v\n", entPath, err)
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
				return nil, err
			}
		}
		// Update the entry's properties.
		defs, err := h.server.Defaults(ctx, ent.Type)
		if err != nil {
			return nil, err
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
						return nil, err
					}
					if old.ValueError != nil {
						return nil, fmt.Errorf("cannot evaluate the previous value: %v.%v", entPath, p)
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
					return nil, fmt.Errorf("got label %q more than once, use %q if you want to combine them", p, p+"+")
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
			return nil, err
		}
	}
	return nil, nil
}
