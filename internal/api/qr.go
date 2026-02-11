package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
)

func (s *Server) SetCurrentQR(qr string) {
	s.currentQR.Store(qr)
}

func (s *Server) GetCurrentQR() string {
	v := s.currentQR.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	authenticated := false
	connected := false
	if s.app != nil {
		authenticated = s.app.IsAuthenticated()
		connected = s.app.IsConnected()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data": map[string]any{
			"authenticated": authenticated,
			"connected":     connected,
		},
	})
}

func (s *Server) handleQRImage(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, return JSON message
	if s.app != nil && s.app.IsAuthenticated() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"message": "already authenticated",
			},
		})
		return
	}

	qr := s.GetCurrentQR()
	if qr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"data":    nil,
			"error":   "no QR code available, try again shortly",
		})
		return
	}

	png, err := qrcode.Encode(qr, qrcode.Medium, 256)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"data":    nil,
			"error":   "failed to generate QR code image",
		})
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(png)))
	w.WriteHeader(http.StatusOK)
	w.Write(png)
}
