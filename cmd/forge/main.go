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
	"github.com/imagvfx/forge/service"
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
		ctx := service.ContextWithUserName(context.Background(), "system")
		for _, s := range cfg.Structs {
			if s.Type == "root" {
				// root entry type should be already created.
				continue
			}
			err = server.AddEntryType(ctx, s.Type)
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, s := range cfg.Structs {
			for _, p := range s.SubEntries {
				err := server.AddDefault(ctx, s.Type, "sub_entry", p.Key, p.Type, p.Value)
				if err != nil {
					log.Fatal(err)
				}
			}
			for _, p := range s.Properties {
				err := server.AddDefault(ctx, s.Type, "property", p.Key, p.Type, p.Value)
				if err != nil {
					log.Fatal(err)
				}
			}
			for _, p := range s.Environs {
				err := server.AddDefault(ctx, s.Type, "environ", p.Key, p.Type, p.Value)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	login := &loginHandler{
		server: server,
		oidc: &forge.OIDC{
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			RedirectURI:  "https://" + host + "/login/callback/google",
			HostDomain:   host,
		},
	}
	path := &pathHandler{
		server: server,
		cfg:    cfg,
	}
	user := &userHandler{
		server: server,
	}
	group := &groupHandler{
		server: server,
	}
	typ := &entryTypeHandler{
		server: server,
	}
	api := &apiHandler{
		server: server,
	}
	Tmpl = template.New("").Funcs(pathHandlerFuncs)
	Tmpl = template.Must(bml.ToHTMLTemplate(Tmpl, "tmpl/*"))
	mux := http.NewServeMux()
	mux.HandleFunc("/", path.Handle)
	mux.HandleFunc("/logs", path.HandleEntryLogs)
	mux.HandleFunc("/thumbnail/", path.HandleThumbnail)
	mux.HandleFunc("/login", login.Handle)
	mux.HandleFunc("/login/callback/google", login.HandleCallback)
	mux.HandleFunc("/logout", login.HandleLogout)
	mux.HandleFunc("/users", user.Handle)
	mux.HandleFunc("/groups", group.Handle)
	mux.HandleFunc("/types", typ.Handle)
	mux.HandleFunc("/api/add-entry-type", api.HandleAddEntryType)
	mux.HandleFunc("/api/rename-entry-type", api.HandleRenameEntryType)
	mux.HandleFunc("/api/delete-entry-type", api.HandleDeleteEntryType)
	mux.HandleFunc("/api/add-default", api.HandleAddDefault)
	mux.HandleFunc("/api/set-default", api.HandleSetDefault)
	mux.HandleFunc("/api/delete-default", api.HandleDeleteDefault)
	mux.HandleFunc("/api/add-entry", api.HandleAddEntry)
	mux.HandleFunc("/api/rename-entry", api.HandleRenameEntry)
	mux.HandleFunc("/api/delete-entry", api.HandleDeleteEntry)
	mux.HandleFunc("/api/add-property", api.HandleAddProperty)
	mux.HandleFunc("/api/set-property", api.HandleSetProperty)
	mux.HandleFunc("/api/get-property", api.HandleGetProperty)
	mux.HandleFunc("/api/delete-property", api.HandleDeleteProperty)
	mux.HandleFunc("/api/add-environ", api.HandleAddEnviron)
	mux.HandleFunc("/api/set-environ", api.HandleSetEnviron)
	mux.HandleFunc("/api/get-environ", api.HandleGetEnviron)
	mux.HandleFunc("/api/delete-environ", api.HandleDeleteEnviron)
	mux.HandleFunc("/api/add-thumbnail", api.HandleAddThumbnail)
	mux.HandleFunc("/api/set-thumbnail", api.HandleUpdateThumbnail)
	mux.HandleFunc("/api/delete-thumbnail", api.HandleDeleteThumbnail)
	mux.HandleFunc("/api/add-access", api.HandleAddAccess)
	mux.HandleFunc("/api/set-access", api.HandleSetAccess)
	mux.HandleFunc("/api/get-access", api.HandleGetAccess)
	mux.HandleFunc("/api/delete-access", api.HandleDeleteAccess)
	mux.HandleFunc("/api/add-group", api.HandleAddGroup)
	mux.HandleFunc("/api/rename-group", api.HandleRenameGroup)
	mux.HandleFunc("/api/add-group-member", api.HandleAddGroupMember)
	mux.HandleFunc("/api/delete-group-member", api.HandleDeleteGroupMember)
	mux.HandleFunc("/api/set-user-setting", api.HandleSetUserSetting)
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
