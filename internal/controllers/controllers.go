package controllers

import (
	"dust/internal/server"
	"dust/pkg/ondemand"
	"dust/pkg/pdf"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-chi/cors"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"

	"github.com/gorilla/websocket"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
)

type AudioOutput struct {
	Type            string `json:"type"`
	ID              string `json:"id"`
	Index           int    `json:"index"`
	Data            string `json:"data"`
	CustomSessionID string `json:"custom_session_id,omitempty"`
}

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

	srv.Mux.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"https://*", "http://*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

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
	HandleProcess("/api/pdf", srv)
	HandleProcessPodcast("/api/podcast", srv)

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

func HandleProcess(pattern string, srv *server.Server) {

	finish := make(chan struct{})

	srv.Mux.Post(pattern, func(w http.ResponseWriter, r *http.Request) {

		// Parse the multipart form
		err := r.ParseMultipartForm(20 << 20) // 20 MB
		if err != nil {
			http.Error(w, "Could not parse form", http.StatusInternalServerError)
			return
		}

		// Get the file from the form
		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Could not get file from form", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		reader, err := pdf.ReadPdfDSLIPAK(file, handler.Size)
		if err != nil {

			http.Error(w, "Could not read pdf", http.StatusInternalServerError)
			return

		}

		b, err := io.ReadAll(reader)
		if err != nil {

			http.Error(w, "Could not read pdf", http.StatusInternalServerError)
			return

		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		c, _, _ := websocket.DefaultDialer.Dial(fmt.Sprintf("wss://api.hume.ai/v0/evi/chat?api_key=%s", os.Getenv("HUME_KEY")), nil)

		go func(conn *websocket.Conn) {

			w.Header().Set("Content-Type", "audio/wav")

			for {

				var audioOutput AudioOutput
				mt, message, err := conn.ReadMessage()

				if mt == websocket.CloseMessage || mt == websocket.CloseNormalClosure || mt == websocket.CloseGoingAway {

					finish <- struct{}{}
					break

				}

				if err != nil {
					finish <- struct{}{}
					break
				}

				if err := json.Unmarshal(message, &audioOutput); err != nil {
					log.Println(err)
					finish <- struct{}{}
					break
				}

				bts, _ := base64.RawStdEncoding.DecodeString(audioOutput.Data)

				if _, err := w.Write(bts); err != nil {
					log.Println(err)
					finish <- struct{}{}
					break

				}

				flusher.Flush()

			}

		}(c)

		promptMap := map[string]string{

			"type":          "session_settings",
			"system_prompt": "Sanitize and format the text as it most likely pdf extracted plaintext. Do not mention this only narrate the text.",
		}
		inputMap := map[string]string{

			"type": "user_input",
			"text": string(b),
		}

		prompt, err := json.Marshal(promptMap)
		if err != nil {

			http.Error(w, "Could not marshal prompt", http.StatusInternalServerError)
			return

		}
		input, err := json.Marshal(inputMap)
		if err != nil {

			http.Error(w, "Could not marshal prompt", http.StatusInternalServerError)
			return

		}

		if err := c.WriteMessage(websocket.TextMessage, prompt); err != nil {

			http.Error(w, "Could not write message", http.StatusInternalServerError)
			return

		}
		if err := c.WriteMessage(websocket.TextMessage, input); err != nil {

			http.Error(w, "Could not write message", http.StatusInternalServerError)
			return

		}

		// Return success
		<-finish
	})
}

type UserInput struct {
	Topics []string `json:"topic"`
}

func HandleProcessPodcast(pattern string, srv *server.Server) {
	finish := make(chan struct{})

	srv.Mux.Post(pattern, func(w http.ResponseWriter, r *http.Request) {

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		c, _, _ := websocket.DefaultDialer.Dial(fmt.Sprintf("wss://api.hume.ai/v0/evi/chat?api_key=%s", os.Getenv("HUME_KEY")), nil)

		defer r.Body.Close()

		var userInput UserInput
		json.NewDecoder(r.Body).Decode(&userInput)

		go func(conn *websocket.Conn) {

			w.Header().Set("Content-Type", "audio/wav")

			for {

				var audioOutput AudioOutput
				mt, message, err := conn.ReadMessage()

				if mt == websocket.CloseMessage || mt == websocket.CloseNormalClosure || mt == websocket.CloseGoingAway {

					finish <- struct{}{}
					break

				}

				if err != nil {
					finish <- struct{}{}
					break
				}

				if err := json.Unmarshal(message, &audioOutput); err != nil {
					log.Println(err)
					finish <- struct{}{}
					break
				}

				bts, _ := base64.RawStdEncoding.DecodeString(audioOutput.Data)

				if _, err := w.Write(bts); err != nil {
					log.Println(err)
					finish <- struct{}{}
					break

				}

				flusher.Flush()

			}

		}(c)

		inputstring, err := ondemand.OnDemand(strings.Join(userInput.Topics, ","))
		if err != nil {

			http.Error(w, "Could not access ondemand rest agent", http.StatusInternalServerError)
			return

		}

		promptMap := map[string]string{

			"type":          "session_settings",
			"system_prompt": "You are a podcast generator. Generate a 30 minute podcast about the topics and content.",
		}
		inputMap := map[string]string{
			"type": "user_input",
			"text": inputstring,
		}

		prompt, err := json.Marshal(promptMap)
		if err != nil {

			http.Error(w, "Could not marshal prompt", http.StatusInternalServerError)
			return

		}
		input, err := json.Marshal(inputMap)
		if err != nil {

			http.Error(w, "Could not marshal prompt", http.StatusInternalServerError)
			return

		}

		if err := c.WriteMessage(websocket.TextMessage, prompt); err != nil {

			http.Error(w, "Could not write message", http.StatusInternalServerError)
			return

		}
		if err := c.WriteMessage(websocket.TextMessage, input); err != nil {

			http.Error(w, "Could not write message", http.StatusInternalServerError)
			return

		}

		// Return success
		<-finish
	})

}
