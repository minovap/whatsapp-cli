package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	cfg := Config{
		APIKey: "test-key",
		Port:   0,
	}
	srv := NewServer(cfg, nil)
	require.NotNil(t, srv)
	assert.NotNil(t, srv.mux)
	assert.Equal(t, cfg, srv.Config)
}

func TestServer_StartAndGracefulShutdown(t *testing.T) {
	cfg := Config{
		APIKey: "test-key",
		Port:   0, // will use a random port via listener
	}
	srv := NewServer(cfg, nil)

	// Use a random available port
	cfg.Port = 18923 + int(time.Now().UnixNano()%100)
	srv.Config = cfg

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	// Verify server is running
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", cfg.Port))
	if err == nil {
		resp.Body.Close()
	}

	// Cancel context to trigger graceful shutdown
	cancel()

	select {
	case err := <-errCh:
		// Shutdown should complete without error
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down within 5 seconds")
	}
}
