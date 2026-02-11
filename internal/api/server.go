package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/mdp/qrterminal"
)

// AppService defines the interface for the application layer used by API handlers.
type AppService interface {
	ListMessages(chatJID *string, query *string, limit, page int, includeJIDs, excludeJIDs []string, after *time.Time) string
	ListChats(query *string, limit, page int, includeJIDs, excludeJIDs []string) string
	SearchContacts(query string, includeJIDs, excludeJIDs []string) string
	SendMessage(ctx context.Context, recipient, message string) string
	IsAuthenticated() bool
	IsConnected() bool
	Sync(ctx context.Context, onMessage func()) string
}

type Server struct {
	mux           *http.ServeMux
	apiMux        *http.ServeMux
	Config        Config
	app           AppService
	phoneFilter   *PhoneFilter
	authenticated atomic.Bool
	syncing       atomic.Bool
	currentQR     atomic.Value // stores string

	// Sync daemon fields
	syncRunning    atomic.Bool
	messagesSynced atomic.Int64
}

func NewServer(cfg Config, app AppService) *Server {
	s := &Server{
		mux:         http.NewServeMux(),
		Config:      cfg,
		app:         app,
		phoneFilter: NewPhoneFilter(cfg.PhoneWhitelist, cfg.PhoneBlacklist),
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
	apiMux.HandleFunc("GET /messages", s.handleListMessages)
	apiMux.HandleFunc("GET /messages/search", s.handleSearchMessages)
	apiMux.HandleFunc("GET /chats", s.handleListChats)
	apiMux.HandleFunc("GET /contacts", s.handleSearchContacts)
	apiMux.HandleFunc("POST /messages/send", s.handleSendMessage)
	apiMux.HandleFunc("GET /auth/status", s.handleAuthStatus)
	apiMux.HandleFunc("GET /auth/qr/image", s.handleQRImage)
	apiMux.HandleFunc("GET /sync/status", s.handleSyncStatus)
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

func (s *Server) handleSyncStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data": map[string]any{
			"running":         s.syncRunning.Load(),
			"messages_synced": s.messagesSynced.Load(),
		},
	})
}

// QRAuthProvider is implemented by types that can perform QR-based authentication.
// This is separate from AppService to avoid coupling the API package to whatsmeow types.
type QRAuthProvider interface {
	AuthWithQRCallback(ctx context.Context, onQR func(code string), onSuccess func()) error
}

// StartQRAuth launches a goroutine that performs QR authentication.
// QR codes are printed to stderr as ASCII art (for docker logs) and stored
// in Server.currentQR for the HTTP QR image endpoint.
func (s *Server) StartQRAuth(ctx context.Context, auth QRAuthProvider) {
	go func() {
		err := auth.AuthWithQRCallback(ctx,
			func(code string) {
				s.SetCurrentQR(code)
				fmt.Fprintln(os.Stderr, "\nScan this QR code with WhatsApp:")
				printQRToStderr(code)
			},
			func() {
				s.SetAuthenticated(true)
				s.SetCurrentQR("")
				fmt.Fprintln(os.Stderr, "\nAuthentication successful!")
			},
		)
		if err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "QR auth error: %v\n", err)
		}
	}()
}

// StartBackgroundSync launches the sync daemon in a background goroutine.
// It waits for authentication (polling Server.authenticated), then starts App.Sync.
// The goroutine is cancelled when ctx is cancelled.
func (s *Server) StartBackgroundSync(ctx context.Context) {
	go func() {
		// Wait for authentication before starting sync
		for !s.authenticated.Load() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
			}
		}

		fmt.Fprintln(os.Stderr, "Starting background sync...")
		s.syncRunning.Store(true)
		s.SetSyncing(true)
		defer func() {
			s.syncRunning.Store(false)
			s.SetSyncing(false)
		}()

		s.app.Sync(ctx, func() {
			s.messagesSynced.Add(1)
		})
	}()
}

func printQRToStderr(code string) {
	qrterminal.GenerateHalfBlock(code, qrterminal.M, os.Stderr)
}
