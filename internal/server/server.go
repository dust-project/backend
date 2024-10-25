package server

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	Mux    chi.Router
	Config *ServerConfig
	Logger *slog.Logger
}

func NewServer(config *ServerConfig) *Server {

	return &Server{Config: config, Logger: slog.Default(), Mux: chi.NewMux()}

}

func (s *Server) Run() {

	http.ListenAndServe(fmt.Sprintf(":%d", s.Config.Port), s.Mux)

}

type ServerConfig struct {
	Port      int
	StaticDir string
}
