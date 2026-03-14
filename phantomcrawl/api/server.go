package api

import (
	"fmt"
	"net/http"

	"github.com/var-raphael/phantomcrawl/config"
	"github.com/var-raphael/phantomcrawl/storage"
)

type Server struct {
	cfg *config.Config
	db  *storage.DB
}

func New(cfg *config.Config, db *storage.DB) *Server {
	return &Server{cfg: cfg, db: db}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", s.handleStream)
	mux.HandleFunc("/data", s.handleData)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.cfg.API.Port)
	fmt.Printf("API running at http://localhost%s\n", addr)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}