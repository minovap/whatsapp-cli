package api

import (
	"net/http"
	"strconv"
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

	result := s.app.ListMessages(chatJID, nil, limit, page)
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

	result := s.app.ListMessages(nil, &query, limit, page)
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

	result := s.app.ListChats(query, limit, page)
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

	result := s.app.SearchContacts(query)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(result))
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
