package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/imagvfx/forge"
)

type loginHandler struct {
	server *forge.Server
	oidc   *forge.OIDC
	apps   *AppSessionManager
}

func (h *loginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	err := func() error {

		session, err := getSession(r)
		if err != nil {
			clearSession(w)
			return err
		}
		// state prevents hijacking of communication
		//
		// Note: Google Chrome (and maybe other browsers too) secretly loads this page again from background.
		// Skip to create a new state in that case, as that will break this whole login process.
		if session["state"] == "" {
			seed := make([]byte, 1024)
			rand.Read(seed)
			hs := sha256.New()
			hs.Write(seed)
			state := fmt.Sprintf("%x", hs.Sum(nil))
			session["state"] = state
		}
		appKey := r.FormValue("app_session_key")
		if appKey != "" {
			// Need to store the authentication info so the app could retrive it with api.
			session["app_session_key"] = appKey
		}
		setSession(w, session)

		// nonce prevents replay attack
		seed := make([]byte, 1024)
		rand.Read(seed)
		hs := sha256.New()
		hs.Write(seed)
		nonce := fmt.Sprintf("%x", hs.Sum(nil))
		recipe := struct {
			OIDC      *forge.OIDC
			OIDCState string
			OIDCNonce string
		}{
			OIDC:      h.oidc,
			OIDCState: session["state"],
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
		called := op.Name
		ctx := forge.ContextWithUserName(r.Context(), user)
		_, err = h.server.GetUser(ctx, user)
		if err != nil {
			var e *forge.NotFoundError
			if !errors.As(err, &e) {
				return err
			}
			u := &forge.User{
				Name:   user,
				Called: called,
			}
			err := h.server.AddUser(ctx, u)
			if err != nil {
				return err
			}
		}
		session["user"] = user
		// clear session info that was created for the login process.
		session["state"] = ""
		appKey := session["app_session_key"]
		session["app_session_key"] = ""
		err = setSession(w, session)
		if err != nil {
			return err
		}
		if appKey != "" {
			encoded, err := secureCookie.Encode("session", session)
			if err != nil {
				return err
			}
			sess := AppSession{
				User:    user,
				Session: encoded,
			}
			h.apps.SendSession(appKey, sess)
			http.Redirect(w, r, "/app-login-completed", http.StatusSeeOther)
			return nil
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil

	}()
	handleError(w, err)
}

func (h *loginHandler) HandleAppLoginCompleted(w http.ResponseWriter, r *http.Request) {
	err := Tmpl.ExecuteTemplate(w, "app_login_completed.bml", nil)
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
	Name  string `json:"name"`
}
