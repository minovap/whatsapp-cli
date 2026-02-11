package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockApp implements AppService for testing.
type mockApp struct {
	listMessagesResult string
	listMessagesCalled bool
	lastChatJID        *string
	lastQuery          *string
	lastLimit          int
	lastPage           int

	listChatsResult string
	listChatsCalled bool
	lastChatsQuery  *string
	lastChatsLimit  int
	lastChatsPage   int

	searchContactsResult string
	searchContactsCalled bool
	lastContactsQuery    string

	sendMessageResult string
	sendMessageCalled bool
	lastSendRecipient string
	lastSendMessage   string
}

func (m *mockApp) ListMessages(chatJID *string, query *string, limit, page int) string {
	m.listMessagesCalled = true
	m.lastChatJID = chatJID
	m.lastQuery = query
	m.lastLimit = limit
	m.lastPage = page
	return m.listMessagesResult
}

func (m *mockApp) SearchContacts(query string) string {
	m.searchContactsCalled = true
	m.lastContactsQuery = query
	return m.searchContactsResult
}

func (m *mockApp) SendMessage(_ context.Context, recipient, message string) string {
	m.sendMessageCalled = true
	m.lastSendRecipient = recipient
	m.lastSendMessage = message
	return m.sendMessageResult
}

func (m *mockApp) ListChats(query *string, limit, page int) string {
	m.listChatsCalled = true
	m.lastChatsQuery = query
	m.lastChatsLimit = limit
	m.lastChatsPage = page
	return m.listChatsResult
}

func newTestServer(app AppService) *Server {
	return NewServer(Config{APIKey: "test-key", MaxMessages: 100}, app)
}

func TestHandleListMessages_Defaults(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":true,"data":[]}`, w.Body.String())
	assert.True(t, mock.listMessagesCalled)
	assert.Nil(t, mock.lastChatJID)
	assert.Nil(t, mock.lastQuery)
	assert.Equal(t, 20, mock.lastLimit)
	assert.Equal(t, 0, mock.lastPage)
}

func TestHandleListMessages_WithChatJID(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?chat_jid=123@s.whatsapp.net", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, mock.lastChatJID)
	assert.Equal(t, "123@s.whatsapp.net", *mock.lastChatJID)
}

func TestHandleListMessages_WithLimitAndPage(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=50&page=2", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 50, mock.lastLimit)
	assert.Equal(t, 2, mock.lastPage)
}

func TestHandleListMessages_LimitCappedToMaxMessages(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=500", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 100, mock.lastLimit) // capped to MaxMessages
}

func TestHandleListMessages_InvalidLimitUsesDefault(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages?limit=abc", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 20, mock.lastLimit) // default
}

func TestHandleSearchMessages_Success(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[{"id":"msg1"}]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/search?query=hello", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":true,"data":[{"id":"msg1"}]}`, w.Body.String())
	assert.True(t, mock.listMessagesCalled)
	assert.Nil(t, mock.lastChatJID)
	require.NotNil(t, mock.lastQuery)
	assert.Equal(t, "hello", *mock.lastQuery)
	assert.Equal(t, 20, mock.lastLimit)
	assert.Equal(t, 0, mock.lastPage)
}

func TestHandleSearchMessages_MissingQuery(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/search", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["success"])
	assert.Nil(t, body["data"])
	assert.Equal(t, "query parameter required", body["error"])
	assert.False(t, mock.listMessagesCalled)
}

func TestHandleSearchMessages_WithLimitAndPage(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/search?query=test&limit=10&page=3", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 10, mock.lastLimit)
	assert.Equal(t, 3, mock.lastPage)
}

func TestHandleSearchMessages_LimitCappedToMaxMessages(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/search?query=test&limit=999", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 100, mock.lastLimit) // capped to MaxMessages
}

func TestHandleListMessages_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, mock.listMessagesCalled)
}

func TestHandleSearchMessages_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/search?query=hello", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, mock.listMessagesCalled)
}

func TestHandleListMessages_WritesAppResponseDirectly(t *testing.T) {
	// Verifies the handler writes the App JSON response string directly (no re-marshaling)
	appJSON := `{"success":true,"data":{"messages":[{"id":"1","content":"hello"}],"total":1}}`
	mock := &mockApp{
		listMessagesResult: appJSON,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, appJSON, w.Body.String())
}

func TestHandleListChats_Defaults(t *testing.T) {
	mock := &mockApp{
		listChatsResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":true,"data":[]}`, w.Body.String())
	assert.True(t, mock.listChatsCalled)
	assert.Nil(t, mock.lastChatsQuery)
	assert.Equal(t, 20, mock.lastChatsLimit)
	assert.Equal(t, 0, mock.lastChatsPage)
}

func TestHandleListChats_WithQuery(t *testing.T) {
	mock := &mockApp{
		listChatsResult: `{"success":true,"data":[{"jid":"123@s.whatsapp.net"}]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats?query=john", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, mock.lastChatsQuery)
	assert.Equal(t, "john", *mock.lastChatsQuery)
}

func TestHandleListChats_WithLimitAndPage(t *testing.T) {
	mock := &mockApp{
		listChatsResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats?limit=50&page=2", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 50, mock.lastChatsLimit)
	assert.Equal(t, 2, mock.lastChatsPage)
}

func TestHandleListChats_LimitCappedToMaxMessages(t *testing.T) {
	mock := &mockApp{
		listChatsResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats?limit=500", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 100, mock.lastChatsLimit) // capped to MaxMessages
}

func TestHandleListChats_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, mock.listChatsCalled)
}

func TestHandleListChats_WritesAppResponseDirectly(t *testing.T) {
	appJSON := `{"success":true,"data":{"chats":[{"jid":"123@s.whatsapp.net","name":"John"}],"total":1}}`
	mock := &mockApp{
		listChatsResult: appJSON,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, appJSON, w.Body.String())
}

func TestHandleSearchContacts_Success(t *testing.T) {
	mock := &mockApp{
		searchContactsResult: `{"success":true,"data":[{"jid":"123@s.whatsapp.net","name":"John"}]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?query=john", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":true,"data":[{"jid":"123@s.whatsapp.net","name":"John"}]}`, w.Body.String())
	assert.True(t, mock.searchContactsCalled)
	assert.Equal(t, "john", mock.lastContactsQuery)
}

func TestHandleSearchContacts_MissingQuery(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["success"])
	assert.Nil(t, body["data"])
	assert.Equal(t, "query parameter required", body["error"])
	assert.False(t, mock.searchContactsCalled)
}

func TestHandleSearchContacts_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?query=john", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, mock.searchContactsCalled)
}

func TestHandleSearchContacts_WritesAppResponseDirectly(t *testing.T) {
	appJSON := `{"success":true,"data":[{"jid":"456@s.whatsapp.net","name":"Jane","phone":"+1234567890"}]}`
	mock := &mockApp{
		searchContactsResult: appJSON,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?query=jane", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, appJSON, w.Body.String())
}

func TestHandleSendMessage_Success(t *testing.T) {
	mock := &mockApp{
		sendMessageResult: `{"success":true,"data":{"id":"msg123"}}`,
	}
	srv := newTestServer(mock)

	body := `{"to":"1234567890","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"success":true,"data":{"id":"msg123"}}`, w.Body.String())
	assert.True(t, mock.sendMessageCalled)
	assert.Equal(t, "1234567890", mock.lastSendRecipient)
	assert.Equal(t, "Hello!", mock.lastSendMessage)
}

func TestHandleSendMessage_MissingTo(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	body := `{"message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["success"])
	assert.Nil(t, resp["data"])
	assert.Equal(t, "'to' and 'message' fields are required", resp["error"])
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_MissingMessage(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	body := `{"to":"1234567890"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["success"])
	assert.Equal(t, "'to' and 'message' fields are required", resp["error"])
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_EmptyBody(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_InvalidJSON(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader("not json"))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["success"])
	assert.Equal(t, "invalid JSON body", resp["error"])
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_BlockedByFilter(t *testing.T) {
	mock := &mockApp{}
	// Create server with a whitelist that only allows 567890
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		PhoneWhitelist: []string{"567890"},
	}, mock)

	// Send to a number NOT in the whitelist
	body := `{"to":"9999999999","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, false, resp["success"])
	assert.Nil(t, resp["data"])
	assert.Equal(t, "recipient not allowed", resp["error"])
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_AllowedByWhitelist(t *testing.T) {
	mock := &mockApp{
		sendMessageResult: `{"success":true,"data":{"id":"msg1"}}`,
	}
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		PhoneWhitelist: []string{"567890"},
	}, mock)

	// Send to a number matching the whitelist suffix
	body := `{"to":"1234567890","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.sendMessageCalled)
	assert.Equal(t, "1234567890", mock.lastSendRecipient)
}

func TestHandleSendMessage_BlockedByBlacklist(t *testing.T) {
	mock := &mockApp{}
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		PhoneBlacklist: []string{"567890"},
	}, mock)

	body := `{"to":"1234567890","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_GroupJIDPassesFilter(t *testing.T) {
	mock := &mockApp{
		sendMessageResult: `{"success":true,"data":{"id":"msg1"}}`,
	}
	// Whitelist only allows 567890, but group JIDs should always pass
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		PhoneWhitelist: []string{"567890"},
	}, mock)

	body := `{"to":"12345678@g.us","message":"Hello group!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	body := `{"to":"1234567890","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, mock.sendMessageCalled)
}

func TestHandleSendMessage_WithFullJID(t *testing.T) {
	mock := &mockApp{
		sendMessageResult: `{"success":true,"data":{"id":"msg1"}}`,
	}
	srv := newTestServer(mock)

	// Provide full JID with @s.whatsapp.net
	body := `{"to":"1234567890@s.whatsapp.net","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages/send", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.sendMessageCalled)
	// Original "to" value is passed to App.SendMessage (not the auto-suffixed version)
	assert.Equal(t, "1234567890@s.whatsapp.net", mock.lastSendRecipient)
}
