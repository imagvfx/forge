package main

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service"
)

type apiHandler struct {
	server *forge.Server
}

func (h *apiHandler) WriteResponse(w http.ResponseWriter, m interface{}, e error) {
	w.WriteHeader(httpStatusFromError(e))
	resp, _ := json.Marshal(forge.APIResponse{Msg: m, Err: e})
	_, err := w.Write(resp)
	if err != nil {
		log.Print(err)
	}
}

func (h *apiHandler) HandleAddEntryType(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		name := r.FormValue("name")
		err = h.server.AddEntryType(ctx, name)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleRenameEntryType(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		name := r.FormValue("name")
		newName := r.FormValue("new_name")
		err = h.server.RenameEntryType(ctx, name, newName)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteEntryType(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		// parent, if suggested, will be used as prefix of the path.
		name := r.FormValue("name")
		err = h.server.DeleteEntryType(ctx, name)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddDefault(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		entType := r.FormValue("entry_type")
		ctg := r.FormValue("category")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		err = h.server.AddDefault(ctx, entType, ctg, name, typ, value)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetDefault(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		entType := r.FormValue("entry_type")
		ctg := r.FormValue("category")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		err = h.server.SetDefault(ctx, entType, ctg, name, typ, value)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteDefault(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		// parent, if suggested, will be used as prefix of the path.
		entType := r.FormValue("entry_type")
		ctg := r.FormValue("category")
		name := r.FormValue("name")
		err = h.server.DeleteDefault(ctx, entType, ctg, name)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddEntry(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		// parent, if suggested, will be used as prefix of the path.
		parent := r.FormValue("parent")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		for _, n := range strings.Fields(name) {
			// treat seperate field a child name
			path := filepath.Join(parent, n)
			err := h.server.AddEntry(ctx, path, typ)
			if err != nil {
				return err
			}
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleRenameEntry(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		// parent, if suggested, will be used as prefix of the path.
		path := r.FormValue("path")
		newName := r.FormValue("new-name")
		err = h.server.RenameEntry(ctx, path, newName)
		if err != nil {
			return err
		}
		newPath := filepath.Dir(path) + "/" + newName
		if r.FormValue("back_to_referer") != "" {
			referer := strings.Replace(r.Header.Get("Referer"), path, newPath, 1)
			http.Redirect(w, r, referer, http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteEntry(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		// parent, if suggested, will be used as prefix of the path.
		path := r.FormValue("path")
		err = h.server.DeleteEntry(ctx, path)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			referer := r.Header.Get("Referer")
			toks := strings.SplitN(referer, "?", 2)
			url := toks[0]
			parm := ""
			if len(toks) == 2 {
				parm = toks[1]
			}
			if strings.HasSuffix(url, path) {
				referer = filepath.Dir(path) + "?" + parm
			}
			http.Redirect(w, r, referer, http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddProperty(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		value = strings.TrimSpace(value)
		err = h.server.AddProperty(ctx, path, name, typ, value)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetProperty(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		value := r.FormValue("value")
		value = strings.TrimSpace(value)
		err = h.server.SetProperty(ctx, path, name, value)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleGetProperty(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		p, err := h.server.GetProperty(ctx, path, name)
		h.WriteResponse(w, p, err)
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteProperty(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		err = h.server.DeleteProperty(ctx, path, name)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddEnviron(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		value = strings.TrimSpace(value)
		err = h.server.AddEnviron(ctx, path, name, typ, value)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetEnviron(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		value := r.FormValue("value")
		value = strings.TrimSpace(value)
		err = h.server.SetEnviron(ctx, path, name, value)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleGetEnviron(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		env, err := h.server.GetEnviron(ctx, path, name)
		h.WriteResponse(w, env, err)
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteEnviron(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		err = h.server.DeleteEnviron(ctx, path, name)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddAccess(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		accessor := r.FormValue("name")
		accessor_type := r.FormValue("type")
		mode := r.FormValue("value")
		mode = strings.TrimSpace(mode)
		err = h.server.AddAccessControl(ctx, path, accessor, accessor_type, mode)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetAccess(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		accessor := r.FormValue("name")
		mode := r.FormValue("value")
		mode = strings.TrimSpace(mode)
		err = h.server.SetAccessControl(ctx, path, accessor, mode)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleGetAccess(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		acl, err := h.server.GetAccessControl(ctx, path, name)
		h.WriteResponse(w, acl, err)
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteAccess(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		name := r.FormValue("name")
		err = h.server.DeleteAccessControl(ctx, path, name)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddGroup(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		group := r.FormValue("group")
		g := &forge.Group{
			Name: group,
		}
		err = h.server.AddGroup(ctx, g)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleRenameGroup(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		group := r.FormValue("group")
		newName := r.FormValue("new-name")
		err = h.server.RenameGroup(ctx, group, newName)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddGroupMember(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		group := r.FormValue("group")
		member := r.FormValue("member")
		err = h.server.AddGroupMember(ctx, group, member)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteGroupMember(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		group := r.FormValue("group")
		member := r.FormValue("member")
		err = h.server.DeleteGroupMember(ctx, group, member)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleAddThumbnail(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
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
		err = h.server.AddThumbnail(ctx, path, img)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleUpdateThumbnail(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
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
		err = h.server.UpdateThumbnail(ctx, path, img)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleDeleteThumbnail(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.FormValue("path")
		err = h.server.DeleteThumbnail(ctx, path)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}

func (h *apiHandler) HandleSetUserSetting(w http.ResponseWriter, r *http.Request) {
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
		ctx := service.ContextWithUserName(r.Context(), user)
		// NOTE: don't use make, maps not for the update should be nil
		if r.FormValue("update_filter") != "" {
			entryType := r.FormValue("entry_page_entry_type")
			filter := r.FormValue("entry_page_property_filter")
			propertyFilter := map[string]string{
				entryType: filter,
			}
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
			err := h.server.UpdateUserSetting(ctx, user, "entry_page_sort_property", sortProperty)
			if err != nil {
				return err
			}
		}
		if r.FormValue("update_quick_search") != "" {
			name := r.FormValue("quick_search_name")
			val := r.FormValue("quick_search_value")
			quickSearch := map[string]string{
				name: val,
			}
			err := h.server.UpdateUserSetting(ctx, user, "entry_page_quick_search", quickSearch)
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
			pinnedPath := service.PinnedPathArranger{
				Path:  path,
				Index: n,
			}
			err = h.server.UpdateUserSetting(ctx, user, "pinned_paths", pinnedPath)
			if err != nil {
				return err
			}
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	handleError(w, err)
}
