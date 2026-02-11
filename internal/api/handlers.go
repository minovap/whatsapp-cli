package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 20)
	page := parseIntParam(r, "page", 0)

	if limit > s.Config.MaxMessages {
		limit = s.Config.MaxMessages
	}

	var chatJID *string
	if v := r.URL.Query().Get("chat_jid"); v != "" {
		chatJID = &v
	}

	includeJIDs, excludeJIDs := s.phoneFilter.JIDSuffixes()
	after := s.computeAfter()

	result := s.app.ListMessages(chatJID, nil, limit, page, includeJIDs, excludeJIDs, after)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
}

func (s *Server) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"data":null,"error":"query parameter required"}`))
		return
	}

	limit := parseIntParam(r, "limit", 20)
	page := parseIntParam(r, "page", 0)

	if limit > s.Config.MaxMessages {
		limit = s.Config.MaxMessages
	}

	includeJIDs, excludeJIDs := s.phoneFilter.JIDSuffixes()
	after := s.computeAfter()

	result := s.app.ListMessages(nil, &query, limit, page, includeJIDs, excludeJIDs, after)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
}

func (s *Server) handleListChats(w http.ResponseWriter, r *http.Request) {
	limit := parseIntParam(r, "limit", 20)
	page := parseIntParam(r, "page", 0)

	if limit > s.Config.MaxMessages {
		limit = s.Config.MaxMessages
	}

	var query *string
	if v := r.URL.Query().Get("query"); v != "" {
		query = &v
	}

	includeJIDs, excludeJIDs := s.phoneFilter.JIDSuffixes()

	result := s.app.ListChats(query, limit, page, includeJIDs, excludeJIDs)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
}

func (s *Server) handleSearchContacts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"data":null,"error":"query parameter required"}`))
		return
	}

	includeJIDs, excludeJIDs := s.phoneFilter.JIDSuffixes()

	result := s.app.SearchContacts(query, includeJIDs, excludeJIDs)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
}

type sendRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req sendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"data":null,"error":"invalid JSON body"}`))
		return
	}

	if req.To == "" || req.Message == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"data":null,"error":"'to' and 'message' fields are required"}`))
		return
	}

	// Auto-append @s.whatsapp.net if no @ sign (matching CLI behavior)
	recipient := req.To
	if !strings.Contains(recipient, "@") {
		recipient = recipient + "@s.whatsapp.net"
	}

	// Check phone filter
	if !s.phoneFilter.IsAllowed(recipient) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success":false,"data":null,"error":"recipient not allowed"}`))
		return
	}

	result := s.app.SendMessage(r.Context(), req.To, req.Message)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
}

func (s *Server) handleMediaDownload(w http.ResponseWriter, r *http.Request) {
	messageID := r.PathValue("message_id")
	if messageID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"data":null,"error":"message_id required"}`))
		return
	}

	var chatJID *string
	if v := r.URL.Query().Get("chat_jid"); v != "" {
		chatJID = &v
	}

	filePath, mimeType, err := s.app.GetMediaFile(messageID, chatJID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"data":    nil,
			"error":   err.Error(),
		})
		return
	}

	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	http.ServeFile(w, r, filePath)
}

// computeAfter returns a *time.Time representing the earliest allowed message time
// based on Config.MaxHours. Returns nil if MaxHours is 0 (disabled).
func (s *Server) computeAfter() *time.Time {
	if s.Config.MaxHours <= 0 {
		return nil
	}
	t := time.Now().Add(-time.Duration(s.Config.MaxHours) * time.Hour)
	return &t
}

func parseIntParam(r *http.Request, name string, defaultVal int) int {
	v := r.URL.Query().Get(name)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}
