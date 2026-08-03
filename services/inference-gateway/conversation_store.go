package inference

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"
)

//go:embed migrations/002_conversations.sql
var conversationsSchema string

//go:embed migrations/003_message_idempotency.sql
var conversationIdempotencySchema string

type Conversation struct {
	ConversationID string                `json:"conversation_id"`
	Title          string                `json:"title"`
	Status         string                `json:"status"`
	Messages       []ConversationMessage `json:"messages"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type ConversationMessage struct {
	MessageID      string    `json:"message_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	RunID          string    `json:"run_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	idempotencyKey string
}

type ConversationStore interface {
	CreateConversation(ctx context.Context, title string) (Conversation, error)
	GetConversation(ctx context.Context, conversationID string) (Conversation, error)
	AppendMessage(ctx context.Context, conversationID, role, content, runID, idempotencyKey string) (ConversationMessage, error)
}

func (s *PostgresConversationStore) AppendMessage(ctx context.Context, conversationID, role, content, runID, idempotencyKey string) (ConversationMessage, error) {
	if s == nil || s.DB == nil {
		return ConversationMessage{}, errors.New("PostgreSQL conversation store is not configured")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return ConversationMessage{}, fmt.Errorf("begin append conversation message: %w", err)
	}
	defer tx.Rollback()

	var status string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM conversations WHERE conversation_id = $1 FOR UPDATE`, conversationID).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		return ConversationMessage{}, fmt.Errorf("conversation %q not found", conversationID)
	} else if err != nil {
		return ConversationMessage{}, fmt.Errorf("lock conversation: %w", err)
	}
	if status != "open" {
		return ConversationMessage{}, fmt.Errorf("conversation %q is not open", conversationID)
	}
	if idempotencyKey != "" {
		var existing ConversationMessage
		err := tx.QueryRowContext(ctx, `
			SELECT message_id, role, content, run_id, created_at
			FROM conversation_messages
			WHERE conversation_id = $1 AND idempotency_key = $2`, conversationID, idempotencyKey).Scan(
			&existing.MessageID, &existing.Role, &existing.Content, &existing.RunID, &existing.CreatedAt)
		if err == nil {
			if existing.Role != role || existing.Content != content {
				return ConversationMessage{}, fmt.Errorf("idempotency key %q was already used with different content", idempotencyKey)
			}
			existing.idempotencyKey = idempotencyKey
			return existing, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ConversationMessage{}, fmt.Errorf("check message idempotency: %w", err)
		}
	}

	now := time.Now().UTC()
	message := ConversationMessage{MessageID: newRequestID(), Role: role, Content: content, RunID: runID, CreatedAt: now, idempotencyKey: idempotencyKey}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO conversation_messages (message_id, conversation_id, role, content, run_id, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7)`, message.MessageID, conversationID, role, content, runID, idempotencyKey, now); err != nil {
		return ConversationMessage{}, fmt.Errorf("append conversation message: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE conversations SET updated_at = $1 WHERE conversation_id = $2`, now, conversationID); err != nil {
		return ConversationMessage{}, fmt.Errorf("update conversation timestamp: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ConversationMessage{}, fmt.Errorf("commit conversation message: %w", err)
	}
	return message, nil
}

type PostgresConversationStore struct {
	DB *sql.DB
}

func NewPostgresConversationStore(ctx context.Context, db *sql.DB) (*PostgresConversationStore, error) {
	store := &PostgresConversationStore{DB: db}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *PostgresConversationStore) ensureSchema(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("PostgreSQL conversation store is not configured")
	}
	if _, err := s.DB.ExecContext(ctx, conversationsSchema); err != nil {
		return fmt.Errorf("apply conversation schema: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, conversationIdempotencySchema); err != nil {
		return fmt.Errorf("apply conversation idempotency schema: %w", err)
	}
	return nil
}

func (s *PostgresConversationStore) CreateConversation(ctx context.Context, title string) (Conversation, error) {
	now := time.Now().UTC()
	conversation := Conversation{
		ConversationID: newRequestID(),
		Title:          title,
		Status:         "open",
		Messages:       []ConversationMessage{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO conversations (conversation_id, title, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`, conversation.ConversationID, conversation.Title,
		conversation.Status, conversation.CreatedAt, conversation.UpdatedAt)
	if err != nil {
		return Conversation{}, fmt.Errorf("create conversation: %w", err)
	}
	return conversation, nil
}

func (s *PostgresConversationStore) GetConversation(ctx context.Context, conversationID string) (Conversation, error) {
	var conversation Conversation
	err := s.DB.QueryRowContext(ctx, `
		SELECT conversation_id, title, status, created_at, updated_at
		FROM conversations WHERE conversation_id = $1`, conversationID).Scan(
		&conversation.ConversationID, &conversation.Title, &conversation.Status,
		&conversation.CreatedAt, &conversation.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, fmt.Errorf("conversation %q not found", conversationID)
	}
	if err != nil {
		return Conversation{}, fmt.Errorf("get conversation: %w", err)
	}

	rows, err := s.DB.QueryContext(ctx, `
		SELECT message_id, role, content, run_id, created_at
		FROM conversation_messages
		WHERE conversation_id = $1 ORDER BY created_at, message_id`, conversationID)
	if err != nil {
		return Conversation{}, fmt.Errorf("get conversation messages: %w", err)
	}
	defer rows.Close()
	conversation.Messages = []ConversationMessage{}
	for rows.Next() {
		var message ConversationMessage
		var runID sql.NullString
		if err := rows.Scan(&message.MessageID, &message.Role, &message.Content, &runID, &message.CreatedAt); err != nil {
			return Conversation{}, fmt.Errorf("scan conversation message: %w", err)
		}
		message.RunID = runID.String
		conversation.Messages = append(conversation.Messages, message)
	}
	if err := rows.Err(); err != nil {
		return Conversation{}, fmt.Errorf("read conversation messages: %w", err)
	}
	return conversation, nil
}
