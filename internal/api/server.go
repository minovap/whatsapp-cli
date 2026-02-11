package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/commands"
)

type Server struct {
	mux           *http.ServeMux
	apiMux        *http.ServeMux
	Config        Config
	app           *commands.App
	authenticated atomic.Bool
	syncing       atomic.Bool
}

func NewServer(cfg Config, app *commands.App) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		Config: cfg,
		app:    app,
	}
	s.registerRoutes()
	return s
}

func (s *Server) SetAuthenticated(v bool) {
	s.authenticated.Store(v)
}

func (s *Server) SetSyncing(v bool) {
	s.syncing.Store(v)
}

func (s *Server) registerRoutes() {
	// Health endpoints — no auth required
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)

	// API v1 routes — protected by auth middleware
	apiMux := http.NewServeMux()
	s.mux.Handle("/api/v1/", s.authMiddleware(http.StripPrefix("/api/v1", apiMux)))
	s.apiMux = apiMux
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authenticated := s.authenticated.Load()
	syncing := s.syncing.Load()

	if authenticated && syncing {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		return
	}

	reason := "not authenticated"
	if authenticated {
		reason = "not syncing"
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_ready",
		"reason": reason,
	})
}

func (s *Server) Start(ctx context.Context) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Config.Port),
		Handler: s.mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
