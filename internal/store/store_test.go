package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *MessageStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMessageStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	return store
}

func TestNewMessageStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewMessageStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestStoreChat(t *testing.T) {
	store := setupTestDB(t)

	err := store.StoreChat("1234@s.whatsapp.net", "John Doe", time.Now())
	assert.NoError(t, err)
}

func TestStoreChatDoesNotOverwriteFriendlyWithJID(t *testing.T) {
	store := setupTestDB(t)
	jid := "1234@s.whatsapp.net"

	require.NoError(t, store.StoreChat(jid, "John Doe", time.Now()))
	require.NoError(t, store.StoreChat(jid, jid, time.Now().Add(time.Minute)))

	chats, err := store.ListChats(ListChatsParams{Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, chats)
	assert.Equal(t, "John Doe", chats[0].Name)
}

func TestStoreChatUpgradesNameFromJID(t *testing.T) {
	store := setupTestDB(t)
	jid := "5678@s.whatsapp.net"

	require.NoError(t, store.StoreChat(jid, jid, time.Now()))
	require.NoError(t, store.StoreChat(jid, "Jane Smith", time.Now().Add(time.Minute)))

	chats, err := store.ListChats(ListChatsParams{Limit: 1})
	require.NoError(t, err)
	require.NotEmpty(t, chats)
	assert.Equal(t, "Jane Smith", chats[0].Name)
}

func TestStoreMessage(t *testing.T) {
	store := setupTestDB(t)

	// First store a chat
	chatJID := "1234@s.whatsapp.net"
	err := store.StoreChat(chatJID, "John Doe", time.Now())
	require.NoError(t, err)

	// Then store a message
	err = store.StoreMessage("msg1", chatJID, "1234", "Hello", time.Now(), false, "", "", "", "", "", nil, nil, nil, 0)
	assert.NoError(t, err)
}

func TestListMessages(t *testing.T) {
	store := setupTestDB(t)
	chatJID := "1234@s.whatsapp.net"

	// Setup test data
	store.StoreChat(chatJID, "John Doe", time.Now())
	now := time.Now()
	store.StoreMessage("msg1", chatJID, "1234", "Hello", now, false, "", "", "", "", "", nil, nil, nil, 0)
	store.StoreMessage("msg2", chatJID, "1234", "World", now.Add(time.Second), false, "", "", "", "", "", nil, nil, nil, 0)

	messages, err := store.ListMessages(ListMessagesParams{ChatJID: &chatJID, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "World", messages[0].Content) // Most recent first
	assert.Equal(t, "Hello", messages[1].Content)
}

func TestGetMessageForDownload(t *testing.T) {
	store := setupTestDB(t)
	chatJID := "1234@s.whatsapp.net"

	require.NoError(t, store.StoreChat(chatJID, "John Doe", time.Now()))

	now := time.Now().UTC().Truncate(time.Second)
	mediaKey := []byte{1, 2, 3}
	fileSHA := []byte{4, 5, 6}
	fileEncSHA := []byte{7, 8, 9}

	err := store.StoreMessage(
		"msg1",
		chatJID,
		"1234",
		"Sample caption",
		now,
		false,
		"image",
		"photo.jpg",
		"https://example.com/image",
		"/media/direct/path",
		"image/jpeg",
		mediaKey,
		fileSHA,
		fileEncSHA,
		1024,
	)
	require.NoError(t, err)

	info, err := store.GetMessageForDownload("msg1", nil)
	require.NoError(t, err)

	assert.Equal(t, "msg1", info.ID)
	assert.Equal(t, chatJID, info.ChatJID)
	assert.Equal(t, "image", info.MediaType)
	assert.Equal(t, "photo.jpg", info.Filename)
	assert.Equal(t, "/media/direct/path", info.DirectPath)
	assert.Equal(t, "image/jpeg", info.MimeType)
	assert.Equal(t, uint64(1024), info.FileLength)
	assert.Equal(t, mediaKey, info.MediaKey)
	assert.Equal(t, fileSHA, info.FileSHA256)
	assert.Equal(t, fileEncSHA, info.FileEncSHA256)
	assert.Nil(t, info.LocalPath)

	err = store.MarkMediaDownloaded("msg1", chatJID, "/tmp/photo.jpg", now.Add(time.Minute))
	require.NoError(t, err)

	infoAfter, err := store.GetMessageForDownload("msg1", nil)
	require.NoError(t, err)

	require.NotNil(t, infoAfter.LocalPath)
	assert.Equal(t, "/tmp/photo.jpg", *infoAfter.LocalPath)
	require.NotNil(t, infoAfter.DownloadedAt)
	assert.True(t, infoAfter.DownloadedAt.Equal(now.Add(time.Minute)))
}

func TestSearchContacts(t *testing.T) {
	store := setupTestDB(t)

	// Setup test data
	store.StoreChat("1234@s.whatsapp.net", "John Doe", time.Now())
	store.StoreChat("5678@s.whatsapp.net", "Jane Smith", time.Now())
	store.StoreChat("9999@g.us", "Group Chat", time.Now()) // Should be excluded

	contacts, err := store.SearchContacts(SearchContactsParams{Query: "John"})
	require.NoError(t, err)
	assert.Len(t, contacts, 1)
	assert.Equal(t, "John Doe", contacts[0].Name)
}

func TestListChats(t *testing.T) {
	store := setupTestDB(t)

	// Setup test data
	store.StoreChat("1234@s.whatsapp.net", "John Doe", time.Now())
	store.StoreChat("5678@s.whatsapp.net", "Jane Smith", time.Now().Add(-time.Hour))

	chats, err := store.ListChats(ListChatsParams{Limit: 10})
	require.NoError(t, err)
	assert.Len(t, chats, 2)
	assert.Equal(t, "John Doe", chats[0].Name) // Most recent first
}

// --- JID suffix filtering tests ---

func setupFilterTestDB(t *testing.T) *MessageStore {
	s := setupTestDB(t)
	now := time.Now()

	// Create chats with distinct phone suffixes
	require.NoError(t, s.StoreChat("11111234@s.whatsapp.net", "Alice", now))
	require.NoError(t, s.StoreChat("22225678@s.whatsapp.net", "Bob", now.Add(-time.Hour)))
	require.NoError(t, s.StoreChat("33339012@s.whatsapp.net", "Charlie", now.Add(-2*time.Hour)))
	require.NoError(t, s.StoreChat("99999999@g.us", "Group Chat", now.Add(-3*time.Hour)))

	// Create messages in each chat
	require.NoError(t, s.StoreMessage("m1", "11111234@s.whatsapp.net", "11111234", "Hello from Alice", now, false, "", "", "", "", "", nil, nil, nil, 0))
	require.NoError(t, s.StoreMessage("m2", "22225678@s.whatsapp.net", "22225678", "Hello from Bob", now.Add(-time.Hour), false, "", "", "", "", "", nil, nil, nil, 0))
	require.NoError(t, s.StoreMessage("m3", "33339012@s.whatsapp.net", "33339012", "Hello from Charlie", now.Add(-2*time.Hour), false, "", "", "", "", "", nil, nil, nil, 0))
	require.NoError(t, s.StoreMessage("m4", "99999999@g.us", "11111234", "Hello from group", now.Add(-3*time.Hour), false, "", "", "", "", "", nil, nil, nil, 0))

	return s
}

func TestListMessages_IncludeJIDs(t *testing.T) {
	s := setupFilterTestDB(t)

	// Include only Alice's suffix
	messages, err := s.ListMessages(ListMessagesParams{
		Limit:       100,
		IncludeJIDs: []string{"111234@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "Hello from Alice", messages[0].Content)
}

func TestListMessages_IncludeJIDs_Multiple(t *testing.T) {
	s := setupFilterTestDB(t)

	// Include Alice and Bob
	messages, err := s.ListMessages(ListMessagesParams{
		Limit:       100,
		IncludeJIDs: []string{"111234@s.whatsapp.net", "225678@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}

func TestListMessages_ExcludeJIDs(t *testing.T) {
	s := setupFilterTestDB(t)

	// Exclude Charlie
	messages, err := s.ListMessages(ListMessagesParams{
		Limit:       100,
		ExcludeJIDs: []string{"339012@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, messages, 3) // Alice, Bob, and Group
}

func TestListMessages_ExcludeJIDs_Multiple(t *testing.T) {
	s := setupFilterTestDB(t)

	// Exclude Alice and Bob
	messages, err := s.ListMessages(ListMessagesParams{
		Limit:       100,
		ExcludeJIDs: []string{"111234@s.whatsapp.net", "225678@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, messages, 2) // Charlie and Group
}

func TestListMessages_NoJIDFilter(t *testing.T) {
	s := setupFilterTestDB(t)

	// No filter — returns all
	messages, err := s.ListMessages(ListMessagesParams{Limit: 100})
	require.NoError(t, err)
	assert.Len(t, messages, 4)
}

func TestListChats_IncludeJIDs(t *testing.T) {
	s := setupFilterTestDB(t)

	chats, err := s.ListChats(ListChatsParams{
		Limit:       100,
		IncludeJIDs: []string{"111234@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, chats, 1)
	assert.Equal(t, "Alice", chats[0].Name)
}

func TestListChats_ExcludeJIDs(t *testing.T) {
	s := setupFilterTestDB(t)

	chats, err := s.ListChats(ListChatsParams{
		Limit:       100,
		ExcludeJIDs: []string{"111234@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, chats, 3) // Bob, Charlie, Group
}

func TestListChats_IncludeJIDs_Multiple(t *testing.T) {
	s := setupFilterTestDB(t)

	chats, err := s.ListChats(ListChatsParams{
		Limit:       100,
		IncludeJIDs: []string{"111234@s.whatsapp.net", "225678@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, chats, 2)
}

func TestListChats_NoJIDFilter(t *testing.T) {
	s := setupFilterTestDB(t)

	chats, err := s.ListChats(ListChatsParams{Limit: 100})
	require.NoError(t, err)
	assert.Len(t, chats, 4)
}

func TestSearchContacts_IncludeJIDs(t *testing.T) {
	s := setupFilterTestDB(t)

	// Search all contacts but include only Alice's suffix
	contacts, err := s.SearchContacts(SearchContactsParams{
		Query:       "",
		IncludeJIDs: []string{"111234@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, contacts, 1)
	assert.Equal(t, "Alice", contacts[0].Name)
}

func TestSearchContacts_ExcludeJIDs(t *testing.T) {
	s := setupFilterTestDB(t)

	// Search all contacts but exclude Alice
	contacts, err := s.SearchContacts(SearchContactsParams{
		Query:       "",
		ExcludeJIDs: []string{"111234@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, contacts, 2) // Bob and Charlie (group excluded by SearchContacts)
}

func TestSearchContacts_NoJIDFilter(t *testing.T) {
	s := setupFilterTestDB(t)

	contacts, err := s.SearchContacts(SearchContactsParams{Query: ""})
	require.NoError(t, err)
	assert.Len(t, contacts, 3) // Alice, Bob, Charlie (group excluded)
}

func TestSearchContacts_IncludeJIDs_Multiple(t *testing.T) {
	s := setupFilterTestDB(t)

	contacts, err := s.SearchContacts(SearchContactsParams{
		Query:       "",
		IncludeJIDs: []string{"111234@s.whatsapp.net", "225678@s.whatsapp.net"},
	})
	require.NoError(t, err)
	assert.Len(t, contacts, 2)
}
