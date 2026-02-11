package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_ValidKey_XAPIKey(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-API-Key", "test-secret-key")
	w := httptest.NewRecorder()

	// Register a test handler on the apiMux
	srv.apiMux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_ValidKey_BearerToken(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	w := httptest.NewRecorder()

	srv.apiMux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["success"])
	assert.Nil(t, body["data"])
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestAuthMiddleware_MissingKey(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["success"])
	assert.Nil(t, body["data"])
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestAuthMiddleware_XAPIKeyTakesPrecedence(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	// X-API-Key is valid, Authorization is invalid — should succeed
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-API-Key", "test-secret-key")
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()

	srv.apiMux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_HealthzNoAuth(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	// /healthz should work without any auth header
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_ReadyzNoAuth(t *testing.T) {
	srv := NewServer(Config{APIKey: "test-secret-key"}, nil)

	// /readyz should work without any auth header
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	srv.mux.ServeHTTP(w, req)

	// 503 because not authenticated/syncing, but NOT 401
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
