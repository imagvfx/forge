package main

import (
	"context"
	"errors"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/securecookie"
	"github.com/imagvfx/forge"
	"github.com/imagvfx/forge/service/sqlite"
	"github.com/kybin/bml"
)

var Tmpl *template.Template

// secureCookie is used to save or clear sessions.
var secureCookie *securecookie.SecureCookie

func portForward(httpsPort string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		to := "https://" + strings.Split(r.Host, ":")[0] + ":" + httpsPort + r.URL.Path
		if r.URL.RawQuery != "" {
			to += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, to, http.StatusTemporaryRedirect)
	}
}

func main() {
	var (
		addr        string
		oidcHost    string
		host        string
		insecure    bool
		cert        string
		key         string
		cookieHash  string
		cookieBlock string
		dbpath      string
	)
	flag.StringVar(&addr, "addr", "0.0.0.0:80:443", "address to bind. automatic port forwarding will be enabled, if two ports are specified")
	flag.StringVar(&host, "host", "", "host name of the site let users access this program")
	flag.StringVar(&oidcHost, "oidc-host", "", "oidc host name of the site let users access this program")
	flag.BoolVar(&insecure, "insecure", false, "use http instead of https for testing")
	flag.StringVar(&cert, "cert", "cert.pem", "https cert file")
	flag.StringVar(&key, "key", "key.pem", "https key file")
	flag.StringVar(&cookieHash, "cookie-hash", "cookie.hash", "hash for encrypting browser cookie. will be generated if not exists")
	flag.StringVar(&cookieBlock, "cookie-block", "cookie.block", "block for encrypting browser cookie. will be generated if not exists")
	flag.StringVar(&dbpath, "db", "forge.db", "db path to create or open")
	flag.Parse()

	var (
		httpPort  string
		httpsPort string
	)
	toks := strings.Split(addr, ":")
	n := len(toks)
	addr = toks[0]
	if host == "" {
		host = addr
	}
	if oidcHost == "" {
		oidcHost = host
	}
	if n == 2 {
		if insecure {
			httpPort = toks[1]
		} else {
			httpsPort = toks[1]
		}
	} else if n == 3 {
		httpPort = toks[1]
		httpsPort = toks[2]
	} else {
		log.Fatalf("invalid bind address: %v", addr)
	}

	_, err := os.Stat(cookieHash)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.WriteFile(cookieHash, securecookie.GenerateRandomKey(64), 0600)
		} else {
			log.Fatal(err)
		}
	}
	_, err = os.Stat(cookieBlock)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			os.WriteFile(cookieBlock, securecookie.GenerateRandomKey(32), 0600)
		} else {
			log.Fatal(err)
		}
	}
	hash, err := os.ReadFile(cookieHash)
	if err != nil {
		log.Fatal(err)
	}
	block, err := os.ReadFile(cookieBlock)
	if err != nil {
		log.Fatal(err)
	}
	secureCookie = securecookie.New(hash, block)

	cfg, err := forge.LoadConfig("config/")
	if err != nil {
		log.Fatal(err)
	}

	dbCreated := false
	_, err = os.Stat(dbpath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Fatal(err)
		}
		dbCreated = true
	}
	db, err := sqlite.Open(dbpath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = sqlite.Init(db)
	if err != nil {
		log.Fatal(err)
	}
	svc := sqlite.NewService(db)
	server := forge.NewServer(svc, cfg)
	if dbCreated {
		// TODO: when fatal raised from this block,
		// remaining code will not be called even if we re-run the program,
		// as the db has already created.
		// how to handle this better?
		ctx := forge.ContextWithUserName(context.Background(), "system")
		for _, t := range cfg.EntryType.Types {
			if t.Name == "root" {
				// root entry type should be already created.
				continue
			}
			err = server.AddEntryType(ctx, t.Name)
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, t := range cfg.EntryType.Types {
			for _, p := range t.SubEntries {
				err := server.AddDefault(ctx, t.Name, "sub_entry", p.Key, p.Type, p.Value)
				if err != nil {
					log.Fatal(err)
				}
			}
			for _, p := range t.Properties {
				err := server.AddDefault(ctx, t.Name, "property", p.Key, p.Type, p.Value)
				if err != nil {
					log.Fatal(err)
				}
			}
			for _, p := range t.Environs {
				err := server.AddDefault(ctx, t.Name, "environ", p.Key, p.Type, p.Value)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	appSessionMan := NewAppSessionManager()
	// appSessionMan.DebugStatus()
	login := &loginHandler{
		server: server,
		oidc: &forge.OIDC{
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			RedirectURI:  "https://" + host + "/login/callback/google",
			HostDomain:   oidcHost,
		},
		apps: appSessionMan,
	}
	page := &pageHandler{
		server: server,
		cfg:    cfg,
	}
	api := &apiHandler{
		server: server,
		apps:   appSessionMan,
	}
	Tmpl = template.New("").Funcs(pageHandlerFuncs)
	Tmpl = template.Must(bml.ToHTMLTemplate(Tmpl, "tmpl/*"))
	mux := http.NewServeMux()
	mux.HandleFunc("/login", login.Handle)
	mux.HandleFunc("/login/callback/google", login.HandleCallback)
	mux.HandleFunc("/app-login-completed", login.HandleAppLoginCompleted)
	mux.HandleFunc("/logout", login.HandleLogout)
	mux.HandleFunc("/", page.Handler(page.handleEntry))
	mux.HandleFunc("/logs", page.Handler(page.handleEntryLogs))
	mux.HandleFunc("/thumbnail/", page.Handler(page.handleThumbnail))
	mux.HandleFunc("/users", page.Handler(page.handleUsers))
	mux.HandleFunc("/groups", page.Handler(page.handleGroups))
	mux.HandleFunc("/types", page.Handler(page.handleEntryTypes))
	mux.HandleFunc("/types/", page.Handler(page.handleEachEntryType))
	mux.HandleFunc("/setting", page.Handler(page.handleSetting))
	mux.HandleFunc("/api/", api.Handler(api.handleNotFound))
	mux.HandleFunc("/api/app-login", api.Handler(api.handleAppLogin))
	mux.HandleFunc("/api/get-base-entry-types", api.Handler(api.handleGetBaseEntryTypes))
	mux.HandleFunc("/api/add-entry-type", api.Handler(api.handleAddEntryType))
	mux.HandleFunc("/api/rename-entry-type", api.Handler(api.handleRenameEntryType))
	mux.HandleFunc("/api/delete-entry-type", api.Handler(api.handleDeleteEntryType))
	mux.HandleFunc("/api/add-default", api.Handler(api.handleAddDefault))
	mux.HandleFunc("/api/update-default", api.Handler(api.handleUpdateDefault))
	mux.HandleFunc("/api/delete-default", api.Handler(api.handleDeleteDefault))
	mux.HandleFunc("/api/get-globals", api.Handler(api.handleGetGlobals))
	mux.HandleFunc("/api/add-global", api.Handler(api.handleAddGlobal))
	mux.HandleFunc("/api/update-global", api.Handler(api.handleUpdateGlobal))
	mux.HandleFunc("/api/delete-global", api.Handler(api.handleDeleteGlobal))
	mux.HandleFunc("/api/sub-entries", api.Handler(api.handleSubEntries))
	mux.HandleFunc("/api/parent-entries", api.Handler(api.handleParentEntries))
	mux.HandleFunc("/api/search-entries", api.Handler(api.handleSearchEntries))
	mux.HandleFunc("/api/add-entry", api.Handler(api.handleAddEntry))
	mux.HandleFunc("/api/get-entry", api.Handler(api.handleGetEntry))
	mux.HandleFunc("/api/rename-entry", api.Handler(api.handleRenameEntry))
	mux.HandleFunc("/api/archive-entry", api.Handler(api.handleArchiveEntry))
	mux.HandleFunc("/api/unarchive-entry", api.Handler(api.handleUnarchiveEntry))
	mux.HandleFunc("/api/delete-entry", api.Handler(api.handleDeleteEntry))
	mux.HandleFunc("/api/count-all-sub-entries", api.Handler(api.handleCountAllSubEntries))
	mux.HandleFunc("/api/update-property", api.Handler(api.handleUpdateProperty))
	mux.HandleFunc("/api/get-property", api.Handler(api.handleGetProperty))
	mux.HandleFunc("/api/add-environ", api.Handler(api.handleAddEnviron))
	mux.HandleFunc("/api/update-environ", api.Handler(api.handleUpdateEnviron))
	mux.HandleFunc("/api/get-environ", api.Handler(api.handleGetEnviron))
	mux.HandleFunc("/api/entry-environs", api.Handler(api.handleEntryEnvirons))
	mux.HandleFunc("/api/delete-environ", api.Handler(api.handleDeleteEnviron))
	mux.HandleFunc("/api/add-thumbnail", api.Handler(api.handleAddThumbnail))
	mux.HandleFunc("/api/update-thumbnail", api.Handler(api.handleUpdateThumbnail))
	mux.HandleFunc("/api/get-thumbnail", api.Handler(api.handleGetThumbnail))
	mux.HandleFunc("/api/delete-thumbnail", api.Handler(api.handleDeleteThumbnail))
	mux.HandleFunc("/api/add-access", api.Handler(api.handleAddAccess))
	mux.HandleFunc("/api/update-access", api.Handler(api.handleUpdateAccess))
	mux.HandleFunc("/api/add-or-update-access", api.Handler(api.handleAddOrUpdateAccess))
	mux.HandleFunc("/api/get-access", api.Handler(api.handleGetAccess))
	mux.HandleFunc("/api/delete-access", api.Handler(api.handleDeleteAccess))
	mux.HandleFunc("/api/add-group", api.Handler(api.handleAddGroup))
	mux.HandleFunc("/api/rename-group", api.Handler(api.handleRenameGroup))
	mux.HandleFunc("/api/add-group-member", api.Handler(api.handleAddGroupMember))
	mux.HandleFunc("/api/delete-group-member", api.Handler(api.handleDeleteGroupMember))
	mux.HandleFunc("/api/update-user-called", api.Handler(api.handleUpdateUserCalled))
	mux.HandleFunc("/api/get-user-setting", api.Handler(api.handleGetUserSetting))
	mux.HandleFunc("/api/update-user-setting", api.Handler(api.handleUpdateUserSetting))
	mux.HandleFunc("/api/get-user-data-section", api.Handler(api.handleGetUserDataSection))
	mux.HandleFunc("/api/set-user-data", api.Handler(api.handleSetUserData))
	mux.HandleFunc("/api/bulk-update", api.Handler(api.handleBulkUpdate))
	fs := http.FileServer(http.Dir("asset"))
	mux.Handle("/asset/", http.StripPrefix("/asset/", fs))

	if insecure {
		log.Printf("bind to %v:%v", addr, httpPort)
		err = http.ListenAndServe(addr+":"+httpPort, mux)
		log.Fatal(err)
	} else {
		if httpPort != "" {
			// port forward
			go func() {
				log.Printf("port forwarding enabled from %v to %v", httpPort, httpsPort)
				err := http.ListenAndServe(addr+":"+httpPort, http.HandlerFunc(portForward(httpsPort)))
				log.Fatal(err)
			}()
		}
		log.Printf("bind to %v:%v", addr, httpsPort)
		err := http.ListenAndServeTLS(addr+":"+httpsPort, cert, key, mux)
		log.Fatal(err)
	}
}
