package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func TestHealthz(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-key"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestReadyz_NotReady_NotAuthenticated(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-key"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])
	assert.Equal(t, "not authenticated", body["reason"])
}

func TestReadyz_NotReady_NotSyncing(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-key"}, nil)
	srv.SetAuthenticated(true)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "not_ready", body["status"])
	assert.Equal(t, "not syncing", body["reason"])
}

func TestReadyz_Ready(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-key"}, nil)
	srv.SetAuthenticated(true)
	srv.SetSyncing(true)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ready", body["status"])
}
