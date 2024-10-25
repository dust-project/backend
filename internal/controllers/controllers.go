package controllers

import (
	"dust/internal/server"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	//"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
)

func fileServer(hostpath, fspath string, root http.FileSystem, srv *server.Server) {

	fs := http.FileServer(root)

	srv.Mux.Get(hostpath, func(w http.ResponseWriter, r *http.Request) {

		// If the requested file exists then return if; otherwise return index.html (fileserver default page)
		if r.URL.Path != hostpath {
			fullPath := fspath + strings.TrimPrefix(path.Clean(r.URL.Path), "/")
			_, err := os.Stat(fullPath)
			if err != nil {
				if !os.IsNotExist(err) {
					panic(err)
				}
				// Requested file does not exist so we return the default (resolves to index.html)
				r.URL.Path = "/"
			}
		}
		fs.ServeHTTP(w, r)

	})

}

func HandleRoutes(srv *server.Server) {

	goth.UseProviders(
		github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), "http://127.0.0.1:8080/auth/github/callback"),
	)

	gothic.GetProviderName = func(req *http.Request) (string, error) {
		provider := chi.URLParam(req, "provider")
		if provider != "" {
			return provider, nil
		}
		return "", fmt.Errorf("no provider specified")
	}

	store := sessions.NewCookieStore([]byte(os.Getenv("GITHUB_SECRET")))
	store.Options.SameSite = http.SameSiteLaxMode
	store.Options.Secure = false // set to true in prod
	store.Options.HttpOnly = true
	store.Options.Path = "/"
	store.MaxAge(86400)

	gothic.Store = store

	// route handling begins
	fileServer("/", srv.Config.StaticDir, http.Dir(srv.Config.StaticDir), srv)
	HandleLoginCallback("/auth/{provider}/callback", srv)
	HandleLogin("/auth/{provider}", srv)
	HandleLogout("/auth/{provider}/logout", srv)

}

func HandleLoginCallback(pattern string, srv *server.Server) {

	srv.Mux.Get(pattern, func(w http.ResponseWriter, r *http.Request) {

		user, err := gothic.CompleteUserAuth(w, r)
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}

		if err := json.NewEncoder(w).Encode(&user); err != nil {

			http.Error(w, "Could not encode json error", http.StatusInternalServerError)

		}

	})

}

func HandleLogin(pattern string, srv *server.Server) {

	srv.Mux.Get(pattern, func(w http.ResponseWriter, r *http.Request) {

		gothic.BeginAuthHandler(w, r)

	})

}

func HandleLogout(pattern string, srv *server.Server) {

	srv.Mux.Get(pattern, func(w http.ResponseWriter, r *http.Request) {
		gothic.Logout(w, r)
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})

}
