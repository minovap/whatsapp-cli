package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

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
	lastIncludeJIDs    []string
	lastExcludeJIDs    []string
	lastAfter          *time.Time

	listChatsResult     string
	listChatsCalled     bool
	lastChatsQuery      *string
	lastChatsLimit      int
	lastChatsPage       int
	lastChatsIncludeJIDs []string
	lastChatsExcludeJIDs []string

	searchContactsResult     string
	searchContactsCalled     bool
	lastContactsQuery        string
	lastContactsIncludeJIDs  []string
	lastContactsExcludeJIDs  []string

	sendMessageResult string
	sendMessageCalled bool
	lastSendRecipient string
	lastSendMessage   string

	authenticated bool
	connected     bool

	syncResult string
	syncCalled bool
	syncCtx    context.Context
}

func (m *mockApp) ListMessages(chatJID *string, query *string, limit, page int, includeJIDs, excludeJIDs []string, after *time.Time) string {
	m.listMessagesCalled = true
	m.lastChatJID = chatJID
	m.lastQuery = query
	m.lastLimit = limit
	m.lastPage = page
	m.lastIncludeJIDs = includeJIDs
	m.lastExcludeJIDs = excludeJIDs
	m.lastAfter = after
	return m.listMessagesResult
}

func (m *mockApp) SearchContacts(query string, includeJIDs, excludeJIDs []string) string {
	m.searchContactsCalled = true
	m.lastContactsQuery = query
	m.lastContactsIncludeJIDs = includeJIDs
	m.lastContactsExcludeJIDs = excludeJIDs
	return m.searchContactsResult
}

func (m *mockApp) SendMessage(_ context.Context, recipient, message string) string {
	m.sendMessageCalled = true
	m.lastSendRecipient = recipient
	m.lastSendMessage = message
	return m.sendMessageResult
}

func (m *mockApp) Sync(ctx context.Context, onMessage func()) string {
	m.syncCalled = true
	m.syncCtx = ctx
	// Block until context is cancelled to mimic real Sync behavior
	<-ctx.Done()
	return m.syncResult
}

func (m *mockApp) IsAuthenticated() bool {
	return m.authenticated
}

func (m *mockApp) IsConnected() bool {
	return m.connected
}

func (m *mockApp) ListChats(query *string, limit, page int, includeJIDs, excludeJIDs []string) string {
	m.listChatsCalled = true
	m.lastChatsQuery = query
	m.lastChatsLimit = limit
	m.lastChatsPage = page
	m.lastChatsIncludeJIDs = includeJIDs
	m.lastChatsExcludeJIDs = excludeJIDs
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

// --- Auth Status Tests ---

func TestHandleAuthStatus_Authenticated(t *testing.T) {
	mock := &mockApp{authenticated: true, connected: true}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, true, data["authenticated"])
	assert.Equal(t, true, data["connected"])
}

func TestHandleAuthStatus_NotAuthenticated(t *testing.T) {
	mock := &mockApp{authenticated: false, connected: false}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, false, data["authenticated"])
	assert.Equal(t, false, data["connected"])
}

func TestHandleAuthStatus_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- QR Image Tests ---

func TestHandleQRImage_AlreadyAuthenticated(t *testing.T) {
	mock := &mockApp{authenticated: true, connected: true}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/qr/image", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, "already authenticated", data["message"])
}

func TestHandleQRImage_NoQRAvailable(t *testing.T) {
	mock := &mockApp{authenticated: false, connected: false}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/qr/image", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, false, body["success"])
	assert.Nil(t, body["data"])
	assert.Equal(t, "no QR code available, try again shortly", body["error"])
}

func TestHandleQRImage_ReturnsPNG(t *testing.T) {
	mock := &mockApp{authenticated: false, connected: false}
	srv := newTestServer(mock)
	srv.SetCurrentQR("test-qr-code-data")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/qr/image", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	// PNG magic bytes
	assert.True(t, len(w.Body.Bytes()) > 8)
	assert.Equal(t, byte(0x89), w.Body.Bytes()[0])
	assert.Equal(t, byte('P'), w.Body.Bytes()[1])
	assert.Equal(t, byte('N'), w.Body.Bytes()[2])
	assert.Equal(t, byte('G'), w.Body.Bytes()[3])
}

func TestHandleQRImage_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/qr/image", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSetCurrentQR_GetCurrentQR(t *testing.T) {
	srv := newTestServer(nil)

	// Initially empty
	assert.Equal(t, "", srv.GetCurrentQR())

	// Set a QR code
	srv.SetCurrentQR("some-qr-data")
	assert.Equal(t, "some-qr-data", srv.GetCurrentQR())

	// Update QR code
	srv.SetCurrentQR("new-qr-data")
	assert.Equal(t, "new-qr-data", srv.GetCurrentQR())

	// Clear QR code
	srv.SetCurrentQR("")
	assert.Equal(t, "", srv.GetCurrentQR())
}

// --- Phone Filter + MAX_HOURS Integration Tests ---

func TestHandleListMessages_PassesPhoneFilterSuffixes(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		MaxHours:       0, // disabled
		PhoneWhitelist: []string{"1234567890"},
	}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.listMessagesCalled)
	assert.Equal(t, []string{"567890@"}, mock.lastIncludeJIDs)
	assert.Nil(t, mock.lastExcludeJIDs)
	assert.Nil(t, mock.lastAfter) // MaxHours=0 => no time filter
}

func TestHandleListMessages_PassesMaxHoursAfter(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := NewServer(Config{
		APIKey:      "test-key",
		MaxMessages: 100,
		MaxHours:    24,
	}, mock)

	before := time.Now().Add(-24 * time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	after := time.Now().Add(-24 * time.Hour)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.listMessagesCalled)
	require.NotNil(t, mock.lastAfter)
	// The computed "after" should be between our before/after bounds
	assert.True(t, mock.lastAfter.After(before) || mock.lastAfter.Equal(before))
	assert.True(t, mock.lastAfter.Before(after) || mock.lastAfter.Equal(after))
}

func TestHandleListMessages_MaxHoursZeroDisablesTimeFilter(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := NewServer(Config{
		APIKey:      "test-key",
		MaxMessages: 100,
		MaxHours:    0,
	}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, mock.lastAfter)
}

func TestHandleListMessages_NoPhoneFilter(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock) // default: no whitelist/blacklist

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, mock.lastIncludeJIDs)
	assert.Nil(t, mock.lastExcludeJIDs)
}

func TestHandleSearchMessages_PassesFilters(t *testing.T) {
	mock := &mockApp{
		listMessagesResult: `{"success":true,"data":[]}`,
	}
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		MaxHours:       48,
		PhoneBlacklist: []string{"9876543210"},
	}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/search?query=hello", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.listMessagesCalled)
	assert.Nil(t, mock.lastIncludeJIDs)
	assert.Equal(t, []string{"543210@"}, mock.lastExcludeJIDs)
	require.NotNil(t, mock.lastAfter) // MaxHours=48 => after is set
}

func TestHandleListChats_PassesPhoneFilterSuffixes(t *testing.T) {
	mock := &mockApp{
		listChatsResult: `{"success":true,"data":[]}`,
	}
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		PhoneWhitelist: []string{"1234567890"},
	}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.listChatsCalled)
	assert.Equal(t, []string{"567890@"}, mock.lastChatsIncludeJIDs)
	assert.Nil(t, mock.lastChatsExcludeJIDs)
}

func TestHandleListChats_NoPhoneFilter(t *testing.T) {
	mock := &mockApp{
		listChatsResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chats", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, mock.lastChatsIncludeJIDs)
	assert.Nil(t, mock.lastChatsExcludeJIDs)
}

func TestHandleSearchContacts_PassesPhoneFilterSuffixes(t *testing.T) {
	mock := &mockApp{
		searchContactsResult: `{"success":true,"data":[]}`,
	}
	srv := NewServer(Config{
		APIKey:         "test-key",
		MaxMessages:    100,
		PhoneBlacklist: []string{"9876543210"},
	}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?query=john", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mock.searchContactsCalled)
	assert.Nil(t, mock.lastContactsIncludeJIDs)
	assert.Equal(t, []string{"543210@"}, mock.lastContactsExcludeJIDs)
}

func TestHandleSearchContacts_NoPhoneFilter(t *testing.T) {
	mock := &mockApp{
		searchContactsResult: `{"success":true,"data":[]}`,
	}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts?query=john", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Nil(t, mock.lastContactsIncludeJIDs)
	assert.Nil(t, mock.lastContactsExcludeJIDs)
}

// --- Sync Status Tests ---

func TestHandleSyncStatus_NotRunning(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, false, data["running"])
	assert.Equal(t, float64(0), data["messages_synced"])
}

func TestHandleSyncStatus_Running(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)
	srv.syncRunning.Store(true)
	srv.messagesSynced.Store(42)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, true, body["success"])
	data := body["data"].(map[string]any)
	assert.Equal(t, true, data["running"])
	assert.Equal(t, float64(42), data["messages_synced"])
}

func TestHandleSyncStatus_RequiresAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	// No API key
	w := httptest.NewRecorder()
	srv.mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestStartBackgroundSync_WaitsForAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv.StartBackgroundSync(ctx)

	// Sync should not have started yet (not authenticated)
	time.Sleep(50 * time.Millisecond)
	assert.False(t, srv.syncRunning.Load())
	assert.False(t, mock.syncCalled)

	// Authenticate — sync should start
	srv.SetAuthenticated(true)
	// Give time for the polling loop (1s interval) + sync startup
	assert.Eventually(t, func() bool {
		return srv.syncRunning.Load() && mock.syncCalled
	}, 3*time.Second, 50*time.Millisecond)
	assert.True(t, srv.syncing.Load())

	// Cancel context — sync should stop
	cancel()
	assert.Eventually(t, func() bool {
		return !srv.syncRunning.Load()
	}, 3*time.Second, 50*time.Millisecond)
}

func TestStartBackgroundSync_CancelledBeforeAuth(t *testing.T) {
	mock := &mockApp{}
	srv := newTestServer(mock)

	ctx, cancel := context.WithCancel(context.Background())
	srv.StartBackgroundSync(ctx)

	// Cancel before authenticating
	cancel()
	time.Sleep(100 * time.Millisecond)

	assert.False(t, srv.syncRunning.Load())
	assert.False(t, mock.syncCalled)
}

func TestStartBackgroundSync_MessageCounter(t *testing.T) {
	customMock := &syncCallbackMock{}
	srv := NewServer(Config{APIKey: "test-key", MaxMessages: 100}, customMock)
	srv.SetAuthenticated(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv.StartBackgroundSync(ctx)

	// Wait for sync to start and get the callback
	assert.Eventually(t, func() bool {
		return customMock.getOnMessage() != nil
	}, 3*time.Second, 50*time.Millisecond)

	// Call the callback and check counter
	cb := customMock.getOnMessage()
	cb()
	cb()
	cb()
	assert.Equal(t, int64(3), srv.messagesSynced.Load())

	cancel()
}

// syncCallbackMock captures the onMessage callback from Sync
type syncCallbackMock struct {
	mockApp
	onMessageMu sync.Mutex
	onMessageCb func()
}

func (m *syncCallbackMock) Sync(ctx context.Context, onMessage func()) string {
	m.onMessageMu.Lock()
	m.onMessageCb = onMessage
	m.onMessageMu.Unlock()
	<-ctx.Done()
	return `{"success":true,"data":{"synced":true}}`
}

func (m *syncCallbackMock) getOnMessage() func() {
	m.onMessageMu.Lock()
	defer m.onMessageMu.Unlock()
	return m.onMessageCb
}

// --- StartQRAuth Tests ---

type mockQRAuth struct {
	qrCodes   []string
	onQRCalls []string
	succeed   bool
	err       error
}

func (m *mockQRAuth) AuthWithQRCallback(ctx context.Context, onQR func(code string), onSuccess func()) error {
	if m.err != nil {
		return m.err
	}
	for _, code := range m.qrCodes {
		if onQR != nil {
			onQR(code)
		}
		m.onQRCalls = append(m.onQRCalls, code)
	}
	if m.succeed && onSuccess != nil {
		onSuccess()
	}
	return nil
}

func TestStartQRAuth_UpdatesCurrentQRAndAuthenticated(t *testing.T) {
	srv := newTestServer(nil)

	qrAuth := &mockQRAuth{
		qrCodes: []string{"qr-code-1", "qr-code-2"},
		succeed: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv.StartQRAuth(ctx, qrAuth)

	// Wait for auth to complete
	assert.Eventually(t, func() bool {
		return srv.authenticated.Load()
	}, 3*time.Second, 50*time.Millisecond)

	// After success, currentQR should be cleared
	assert.Equal(t, "", srv.GetCurrentQR())
	assert.True(t, srv.authenticated.Load())
}

func TestStartQRAuth_SetsCurrentQRDuringAuth(t *testing.T) {
	srv := newTestServer(nil)

	// Use a blocking QR auth to observe intermediate state
	qrReady := make(chan struct{})
	done := make(chan struct{})
	qrAuth := &blockingQRAuth{
		code:    "test-qr",
		ready:   qrReady,
		done:    done,
		succeed: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv.StartQRAuth(ctx, qrAuth)

	// Wait for QR code to be set
	<-qrReady
	assert.Equal(t, "test-qr", srv.GetCurrentQR())

	// Signal success
	close(done)
	assert.Eventually(t, func() bool {
		return srv.authenticated.Load()
	}, 3*time.Second, 50*time.Millisecond)
}

type blockingQRAuth struct {
	code    string
	ready   chan struct{}
	done    chan struct{}
	succeed bool
}

func (m *blockingQRAuth) AuthWithQRCallback(ctx context.Context, onQR func(code string), onSuccess func()) error {
	if onQR != nil {
		onQR(m.code)
	}
	close(m.ready)
	<-m.done
	if m.succeed && onSuccess != nil {
		onSuccess()
	}
	return nil
}
