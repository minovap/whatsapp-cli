package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/vicentereig/whatsapp-cli/internal/commands"
)

type Server struct {
	mux    *http.ServeMux
	Config Config
	app    *commands.App
}

func NewServer(cfg Config, app *commands.App) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		Config: cfg,
		app:    app,
	}
	return s
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
