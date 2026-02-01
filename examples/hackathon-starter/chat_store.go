// SQLite-based conversation store for persistent chat history
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/becomeliminal/nim-go-sdk/store"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteConversations implements store.Conversations with SQLite persistence
type SQLiteConversations struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteConversations creates a new SQLite-based conversation store
func NewSQLiteConversations(dbPath string) (*SQLiteConversations, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteConversations{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *SQLiteConversations) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		title TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		blocks_json TEXT,
		tools_json TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Create creates a new conversation
func (s *SQLiteConversations) Create(ctx context.Context, userID string) (*store.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	now := time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO conversations (id, user_id, title, created_at, updated_at)
		VALUES (?, ?, '', ?, ?)
	`, id, userID, now, now)

	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return &store.Conversation{
		ID:        id,
		UserID:    userID,
		Title:     "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Get retrieves a conversation with all its messages
func (s *SQLiteConversations) Get(ctx context.Context, id string) (*store.ConversationWithMessages, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get conversation metadata
	var conv store.Conversation
	var createdAtStr, updatedAtStr string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, created_at, updated_at
		FROM conversations WHERE id = ?
	`, id).Scan(&conv.ID, &conv.UserID, &conv.Title, &createdAtStr, &updatedAtStr)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Parse timestamps
	conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

	// Get messages
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role, content, blocks_json, tools_json, created_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []store.StoredMessage
	for rows.Next() {
		var msg store.StoredMessage
		var blocksJSON, toolsJSON sql.NullString
		var createdAtStr string

		err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &blocksJSON, &toolsJSON, &createdAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Parse timestamp
		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)

		// Parse JSON fields if present
		if blocksJSON.Valid && blocksJSON.String != "" {
			json.Unmarshal([]byte(blocksJSON.String), &msg.Blocks)
		}
		if toolsJSON.Valid && toolsJSON.String != "" {
			json.Unmarshal([]byte(toolsJSON.String), &msg.Tools)
		}

		messages = append(messages, msg)
	}

	return &store.ConversationWithMessages{
		Conversation: conv,
		Messages:     messages,
	}, nil
}

// Append adds a message to a conversation
func (s *SQLiteConversations) Append(ctx context.Context, msg *store.AppendMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize JSON fields
	blocksJSON, _ := json.Marshal(msg.Blocks)
	toolsJSON, _ := json.Marshal(msg.Tools)

	msgID := uuid.New().String()
	createdAt := time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO messages (id, conversation_id, role, content, blocks_json, tools_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, msgID, msg.ConversationID, msg.Role, msg.Content, string(blocksJSON), string(toolsJSON), createdAt)

	if err != nil {
		return fmt.Errorf("failed to append message: %w", err)
	}

	// Update conversation updated_at
	_, err = s.db.ExecContext(ctx, `
		UPDATE conversations SET updated_at = ? WHERE id = ?
	`, time.Now(), msg.ConversationID)

	return err
}

// SetTitle updates the conversation title
func (s *SQLiteConversations) SetTitle(ctx context.Context, id, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		UPDATE conversations SET title = ?, updated_at = ? WHERE id = ?
	`, title, time.Now(), id)

	if err != nil {
		return fmt.Errorf("failed to set title: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}

	return nil
}

// List returns all conversations for a user
func (s *SQLiteConversations) List(ctx context.Context, userID string, limit int) ([]*store.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, title, created_at, updated_at
		FROM conversations
		WHERE user_id = ?
		ORDER BY updated_at DESC
		LIMIT ?
	`, userID, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*store.Conversation
	for rows.Next() {
		conv := &store.Conversation{}
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &createdAtStr, &updatedAtStr); err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// Delete removes a conversation and its messages
func (s *SQLiteConversations) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete messages first (foreign key)
	_, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE conversation_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete conversation
	result, err := s.db.ExecContext(ctx, `DELETE FROM conversations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found")
	}

	return nil
}

// Close closes the database connection
func (s *SQLiteConversations) Close() error {
	return s.db.Close()
}

// GetMessageCount returns the number of messages in a conversation
func (s *SQLiteConversations) GetMessageCount(ctx context.Context, conversationID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages WHERE conversation_id = ?
	`, conversationID).Scan(&count)

	return count, err
}

// GetRecentMessages returns the most recent messages from a conversation
func (s *SQLiteConversations) GetRecentMessages(ctx context.Context, conversationID string, limit int) ([]store.StoredMessage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role, content, blocks_json, tools_json, created_at
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, conversationID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []store.StoredMessage
	for rows.Next() {
		var msg store.StoredMessage
		var blocksJSON, toolsJSON sql.NullString
		var createdAtStr string

		if err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &blocksJSON, &toolsJSON, &createdAtStr); err != nil {
			return nil, err
		}

		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)

		if blocksJSON.Valid {
			json.Unmarshal([]byte(blocksJSON.String), &msg.Blocks)
		}
		if toolsJSON.Valid {
			json.Unmarshal([]byte(toolsJSON.String), &msg.Tools)
		}

		messages = append(messages, msg)
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
