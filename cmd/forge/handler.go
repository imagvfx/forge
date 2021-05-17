package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image"
	_ "image/jpeg"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service"
)

type pathHandler struct {
	server *forge.Server
	cfg    *forge.Config
}

var pathHandlerFuncs = template.FuncMap{
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
}

func handleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var notFound *service.NotFoundError
	if errors.As(err, &notFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// Log unauthorized and undefined errors.
	log.Print(err)
	var unauthorized *service.UnauthorizedError
	if errors.As(err, &unauthorized) {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (h *pathHandler) Handle(w http.ResponseWriter, r *http.Request) {
	tab := r.FormValue("tab")
	switch tab {
	case "logs":
		h.HandleEntryLogs(w, r)
		return
	case "edit":
		h.HandleEntryEdit(w, r)
		return
	case "delete":
		h.HandleEntryDelete(w, r)
		return
	}
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		user := session["user"]
		if user == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.URL.Path
		ent, err := h.server.GetEntry(ctx, path)
		if err != nil {
			return err
		}
		subEnts, err := h.server.SubEntries(ctx, path)
		if err != nil {
			return err
		}
		sort.Slice(subEnts, func(i, j int) bool {
			return subEnts[i].Name() < subEnts[j].Name()
		})
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
		subtyps, err := h.server.SubEntryTypes(ctx, ent.Type)
		if err != nil {
			return err
		}
		recipe := struct {
			User           string
			Entry          *forge.Entry
			SubEntries     []*forge.Entry
			Properties     []*forge.Property
			Environs       []*forge.Property
			SubEntryTypes  []string
			AccessControls []*forge.AccessControl
		}{
			User:           user,
			Entry:          ent,
			SubEntries:     subEnts,
			Properties:     props,
			Environs:       envs,
			SubEntryTypes:  subtyps,
			AccessControls: acs,
		}
		err = Tmpl.ExecuteTemplate(w, "entry.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *pathHandler) HandleEntryEdit(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		user := session["user"]
		if user == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.URL.Path
		ent, err := h.server.GetEntry(ctx, path)
		if err != nil {
			return err
		}
		subEnts, err := h.server.SubEntries(ctx, path)
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
		subtyps, err := h.server.SubEntryTypes(ctx, ent.Type)
		if err != nil {
			return err
		}
		recipe := struct {
			User           string
			Entry          *forge.Entry
			SubEntries     []*forge.Entry
			Properties     []*forge.Property
			Environs       []*forge.Property
			SubEntryTypes  []string
			AccessControls []*forge.AccessControl
		}{
			User:           user,
			Entry:          ent,
			SubEntries:     subEnts,
			Properties:     props,
			Environs:       envs,
			SubEntryTypes:  subtyps,
			AccessControls: acs,
		}
		err = Tmpl.ExecuteTemplate(w, "entry-edit.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *pathHandler) HandleEntryDelete(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		user := session["user"]
		if user == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.URL.Path
		ent, err := h.server.GetEntry(ctx, path)
		if err != nil {
			return err
		}
		subEnts, err := h.server.SubEntries(ctx, path)
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
		subtyps, err := h.server.SubEntryTypes(ctx, ent.Type)
		if err != nil {
			return err
		}
		recipe := struct {
			User           string
			Entry          *forge.Entry
			SubEntries     []*forge.Entry
			Properties     []*forge.Property
			Environs       []*forge.Property
			SubEntryTypes  []string
			AccessControls []*forge.AccessControl
		}{
			User:           user,
			Entry:          ent,
			SubEntries:     subEnts,
			Properties:     props,
			Environs:       envs,
			SubEntryTypes:  subtyps,
			AccessControls: acs,
		}
		err = Tmpl.ExecuteTemplate(w, "entry-delete.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *pathHandler) HandleEntryLogs(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		user := session["user"]
		if user == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		ctx := service.ContextWithUserName(r.Context(), user)
		path := r.URL.Path
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
	}()
	handleError(w, err)
}

func (h *pathHandler) HandleThumbnail(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		if !strings.HasPrefix(r.URL.Path, "/thumbnail/") {
			return fmt.Errorf("invalid thumbnail path")
		}
		path := strings.TrimPrefix(r.URL.Path, "/thumbnail")
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
		ctx := service.ContextWithUserName(r.Context(), user)
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
	}()
	handleError(w, err)
}

type groupHandler struct {
	server *forge.Server
}

func (h *groupHandler) Handle(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		user := session["user"]
		if user == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		ctx := service.ContextWithUserName(r.Context(), user)
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
	}()
	handleError(w, err)
}

type entryTypeHandler struct {
	server *forge.Server
}

func (h *entryTypeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		user := session["user"]
		if user == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		ctx := service.ContextWithUserName(r.Context(), user)
		entTypes, err := h.server.EntryTypes(ctx)
		if err != nil {
			return err
		}
		subEntryTypes := make(map[string][]string)
		entDefaults := make(map[string][]*forge.Default)
		for _, t := range entTypes {
			subTypes, err := h.server.SubEntryTypes(ctx, t)
			if err != nil {
				return err
			}
			subEntryTypes[t] = subTypes
			items, err := h.server.Defaults(ctx, t)
			if err != nil {
				return err
			}
			entDefaults[t] = items
		}
		recipe := struct {
			User          string
			EntryTypes    []string
			SubEntryTypes map[string][]string
			Defaults      map[string][]*forge.Default
		}{
			User:          user,
			EntryTypes:    entTypes,
			SubEntryTypes: subEntryTypes,
			Defaults:      entDefaults,
		}
		err = Tmpl.ExecuteTemplate(w, "types.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

type loginHandler struct {
	server *forge.Server
	oidc   *forge.OIDC
}

func (h *loginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		// state prevents hijacking of communication
		seed := make([]byte, 1024)
		rand.Read(seed)
		hs := sha256.New()
		hs.Write(seed)
		state := fmt.Sprintf("%x", hs.Sum(nil))

		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		session["state"] = state
		setSession(w, session)

		// nonce prevents replay attack
		seed = make([]byte, 1024)
		rand.Read(seed)
		hs = sha256.New()
		hs.Write(seed)
		nonce := fmt.Sprintf("%x", hs.Sum(nil))

		recipe := struct {
			OIDC      *forge.OIDC
			OIDCState string
			OIDCNonce string
		}{
			OIDC:      h.oidc,
			OIDCState: state,
			OIDCNonce: nonce,
		}
		err = Tmpl.ExecuteTemplate(w, "login.bml", recipe)
		if err != nil {
			return err
		}
		return nil
	}()
	handleError(w, err)
}

func (h *loginHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		if r.FormValue("state") != session["state"] {
			return fmt.Errorf("send and recieved states are different")
		}
		// code is needed for backend communication
		code := r.FormValue("code")
		if code == "" {
			return fmt.Errorf("no code in oauth response")
		}
		resp, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
			"code":          {code},
			"client_id":     {h.oidc.ClientID},
			"client_secret": {h.oidc.ClientSecret},
			"redirect_uri":  {h.oidc.RedirectURI},
			"grant_type":    {"authorization_code"},
		})
		if err != nil {
			return err
		}
		oa := OIDCResponse{}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&oa)
		if err != nil {
			return err
		}
		part := strings.Split(oa.IDToken, ".")
		if len(part) != 3 {
			return fmt.Errorf("oauth id token should consist of 3 parts")
		}
		// usually we need to verify jwt token, but will skip this time as we just got from authorization server.
		payload, err := base64.RawURLEncoding.DecodeString(part[1])
		if err != nil {
			return err
		}
		op := OIDCPayload{}
		dec = json.NewDecoder(bytes.NewReader(payload))
		err = dec.Decode(&op)
		if err != nil {
			return err
		}
		user := op.Email
		ctx := service.ContextWithUserName(r.Context(), user)
		_, err = h.server.GetUser(ctx, user)
		if err != nil {
			var e *service.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			err := h.server.AddUser(ctx, user)
			if err != nil {
				return err
			}
		}
		session["user"] = user
		err = setSession(w, session)
		if err != nil {
			return err
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil

	}()
	handleError(w, err)
}

func (h *loginHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	err := func() error {
		clearSession(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}()
	handleError(w, err)
}

type OIDCResponse struct {
	IDToken string `json:"id_token"`
}

type OIDCPayload struct {
	Email string `json:"email"`
}

type apiHandler struct {
	server *forge.Server
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
	if err != nil {
		handleError(w, err)
		return
	}
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
	if err != nil {
		handleError(w, err)
	}
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
	if err != nil {
		handleError(w, err)
	}
}

func (h *apiHandler) HandleAddSubEntryType(w http.ResponseWriter, r *http.Request) {
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
		parentType := r.FormValue("parent_type")
		subType := r.FormValue("sub_type")
		err = h.server.AddSubEntryType(ctx, parentType, subType)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
}

func (h *apiHandler) HandleDeleteSubEntryType(w http.ResponseWriter, r *http.Request) {
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
		parentType := r.FormValue("parent_type")
		subType := r.FormValue("sub_type")
		err = h.server.DeleteSubEntryType(ctx, parentType, subType)
		if err != nil {
			return err
		}
		if r.FormValue("back_to_referer") != "" {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
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
	if err != nil {
		handleError(w, err)
		return
	}
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
	if err != nil {
		handleError(w, err)
	}
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
	if err != nil {
		handleError(w, err)
	}
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
		path := r.FormValue("path")
		path = filepath.Join(parent, path)
		typ := r.FormValue("type")
		err = h.server.AddEntry(ctx, path, typ)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
	if err != nil {
		handleError(w, err)
	}
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
			parm := toks[1]
			if strings.HasSuffix(url, path) {
				referer = filepath.Dir(path) + "?" + parm
			}
			http.Redirect(w, r, referer, http.StatusSeeOther)
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
	}
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
		err = h.server.AddProperty(ctx, path, name, typ, value)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		err = h.server.SetProperty(ctx, path, name, value)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		err = h.server.AddEnviron(ctx, path, name, typ, value)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		err = h.server.SetEnviron(ctx, path, name, value)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func (h *apiHandler) HandleAddAccessControl(w http.ResponseWriter, r *http.Request) {
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
		accessor := r.FormValue("accessor")
		accessor_type := r.FormValue("accessor_type")
		mode := r.FormValue("mode")
		err = h.server.AddAccessControl(ctx, path, accessor, accessor_type, mode)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func (h *apiHandler) HandleSetAccessControl(w http.ResponseWriter, r *http.Request) {
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
		accessor := r.FormValue("accessor")
		mode := r.FormValue("mode")
		err = h.server.SetAccessControl(ctx, path, accessor, mode)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func (h *apiHandler) HandleDeleteAccessControl(w http.ResponseWriter, r *http.Request) {
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		err = h.server.AddGroup(ctx, group)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func (h *apiHandler) HandleSetGroup(w http.ResponseWriter, r *http.Request) {
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
		id := r.FormValue("id")
		group := r.FormValue("group")
		err = h.server.SetGroup(ctx, id, group)
		if err != nil {
			return err
		}
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
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
		return nil
	}()
	if err != nil {
		handleError(w, err)
		return
	}
	if r.FormValue("back_to_referer") != "" {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}
