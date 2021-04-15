package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/imagvfx/forge"
)

type pathHandler struct {
	server *forge.Server
	cfg    *forge.Config
	oidc   *forge.OIDC
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
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (h *pathHandler) Handle(w http.ResponseWriter, r *http.Request) {
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
		path := r.URL.Path
		ent, err := h.server.GetEntry(user, path)
		if err != nil {
			return err
		}
		subEnts, err := h.server.SubEntries(user, path)
		if err != nil {
			return err
		}
		props, err := h.server.EntryProperties(user, path)
		if err != nil {
			return err
		}
		envs, err := h.server.EntryEnvirons(user, path)
		if err != nil {
			return err
		}
		acs, err := h.server.EntryAccessControls(user, path)
		if err != nil {
			return err
		}
		logs, err := h.server.EntryLogs(user, path)
		if err != nil {
			return err
		}
		subtyps := h.cfg.Struct[ent.Type()].SubEntryTypes
		recipe := struct {
			User           string
			Entry          *forge.Entry
			SubEntries     []*forge.Entry
			Properties     []*forge.Property
			Environs       []*forge.Property
			SubEntryTypes  []string
			AccessControls []*forge.AccessControl
			Logs           []*forge.Log
		}{
			User:           user,
			Entry:          ent,
			SubEntries:     subEnts,
			Properties:     props,
			Environs:       envs,
			SubEntryTypes:  subtyps,
			AccessControls: acs,
			Logs:           logs,
		}
		err = Tmpl.ExecuteTemplate(w, "path.bml", recipe)
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
		groups, err := h.server.FindAllGroups()
		if err != nil {
			return err
		}
		members := make(map[string][]*forge.Member)
		for _, g := range groups {
			mems, err := h.server.FindGroupMembers(g.Name)
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
		_, err = h.server.GetUser(user)
		if err != nil {
			if !errors.As(err, &forge.NotFoundError{}) {
				return err
			}
			err := h.server.AddUser(user)
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
		// parent, if suggested, will be used as prefix of the path.
		parent := r.FormValue("parent")
		path := r.FormValue("path")
		path = filepath.Join(parent, path)
		typ := r.FormValue("type")
		err = h.server.AddEntry(user, path, typ)
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
		path := r.FormValue("path")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		err = h.server.AddProperty(user, path, name, typ, value)
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
		path := r.FormValue("path")
		name := r.FormValue("name")
		value := r.FormValue("value")
		err = h.server.SetProperty(user, path, name, value)
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
		path := r.FormValue("path")
		name := r.FormValue("name")
		typ := r.FormValue("type")
		value := r.FormValue("value")
		err = h.server.AddEnviron(user, path, name, typ, value)
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
		path := r.FormValue("path")
		name := r.FormValue("name")
		value := r.FormValue("value")
		err = h.server.SetEnviron(user, path, name, value)
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
		path := r.FormValue("path")
		accessor := r.FormValue("accessor")
		accessor_type := r.FormValue("accessor_type")
		mode := r.FormValue("mode")
		err = h.server.AddAccessControl(user, path, accessor, accessor_type, mode)
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
		id := r.FormValue("id")
		mode := r.FormValue("mode")
		err = h.server.SetAccessControl(user, id, mode)
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
		group := r.FormValue("group")
		err = h.server.AddGroup(user, group)
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
		id := r.FormValue("id")
		group := r.FormValue("group")
		err = h.server.SetGroup(user, id, group)
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
		group := r.FormValue("group")
		member := r.FormValue("member")
		err = h.server.AddGroupMember(user, group, member)
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
		id := r.FormValue("id")
		err = h.server.DeleteGroupMember(user, id)
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
