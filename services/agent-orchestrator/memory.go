package orchestrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// MemoryFact is a scoped piece of knowledge for context assembly.
type MemoryFact struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id,omitempty"`
	Scope       string    `json:"scope"` // short | long
	Key         string    `json:"key,omitempty"`
	Content     string    `json:"content"`
	RunID       string    `json:"run_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// MemoryStore is the thin memory port.
type MemoryStore interface {
	PutShort(ctx context.Context, runID, content string) error
	ListShort(ctx context.Context, runID string) ([]MemoryFact, error)
	PutLong(ctx context.Context, fact MemoryFact) error
	ListLong(ctx context.Context, workspaceID, userID string, limit int) ([]MemoryFact, error)
}

// InMemoryMemory is the default short+long store (also short-term Redis substitute).
type InMemoryMemory struct {
	mu    sync.RWMutex
	short map[string][]MemoryFact // runID
	long  []MemoryFact
}

func NewInMemoryMemory() *InMemoryMemory {
	return &InMemoryMemory{
		short: make(map[string][]MemoryFact),
	}
}

func (m *InMemoryMemory) PutShort(_ context.Context, runID, content string) error {
	if runID == "" || content == "" {
		return errors.New("run_id and content required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.short[runID] = append(m.short[runID], MemoryFact{
		ID: newID(), Scope: "short", RunID: runID, Content: content, CreatedAt: time.Now().UTC(),
	})
	return nil
}

func (m *InMemoryMemory) ListShort(_ context.Context, runID string) ([]MemoryFact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src := m.short[runID]
	out := make([]MemoryFact, len(src))
	copy(out, src)
	return out, nil
}

func (m *InMemoryMemory) PutLong(_ context.Context, fact MemoryFact) error {
	if fact.Content == "" {
		return errors.New("content required")
	}
	if fact.ID == "" {
		fact.ID = newID()
	}
	if fact.WorkspaceID == "" {
		fact.WorkspaceID = "default"
	}
	if fact.Scope == "" {
		fact.Scope = "long"
	}
	fact.CreatedAt = time.Now().UTC()
	m.mu.Lock()
	m.long = append(m.long, fact)
	m.mu.Unlock()
	return nil
}

func (m *InMemoryMemory) ListLong(_ context.Context, workspaceID, userID string, limit int) ([]MemoryFact, error) {
	if workspaceID == "" {
		workspaceID = "default"
	}
	if limit <= 0 {
		limit = 20
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []MemoryFact
	for i := len(m.long) - 1; i >= 0 && len(out) < limit; i-- {
		f := m.long[i]
		if f.WorkspaceID != workspaceID {
			continue
		}
		if userID != "" && f.UserID != "" && f.UserID != userID {
			continue
		}
		out = append(out, f)
	}
	return out, nil
}

// PostgresMemory stores long-term facts in Postgres; short-term stays in-process.
type PostgresMemory struct {
	DB    *sql.DB
	short *InMemoryMemory
}

const memorySchema = `
CREATE TABLE IF NOT EXISTS memory_facts (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    scope TEXT NOT NULL DEFAULT 'long',
    key TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    run_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS memory_facts_ws_idx ON memory_facts (workspace_id, created_at DESC);
`

func OpenPostgresMemory(ctx context.Context, databaseURL string) (*PostgresMemory, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, memorySchema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &PostgresMemory{DB: db, short: NewInMemoryMemory()}, nil
}

func (m *PostgresMemory) PutShort(ctx context.Context, runID, content string) error {
	return m.short.PutShort(ctx, runID, content)
}
func (m *PostgresMemory) ListShort(ctx context.Context, runID string) ([]MemoryFact, error) {
	return m.short.ListShort(ctx, runID)
}

func (m *PostgresMemory) PutLong(ctx context.Context, fact MemoryFact) error {
	if fact.ID == "" {
		fact.ID = newID()
	}
	if fact.WorkspaceID == "" {
		fact.WorkspaceID = "default"
	}
	if fact.Scope == "" {
		fact.Scope = "long"
	}
	if fact.CreatedAt.IsZero() {
		fact.CreatedAt = time.Now().UTC()
	}
	_, err := m.DB.ExecContext(ctx, `
		INSERT INTO memory_facts (id, workspace_id, user_id, scope, key, content, run_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		fact.ID, fact.WorkspaceID, fact.UserID, fact.Scope, fact.Key, fact.Content, fact.RunID, fact.CreatedAt)
	return err
}

func (m *PostgresMemory) ListLong(ctx context.Context, workspaceID, userID string, limit int) ([]MemoryFact, error) {
	if workspaceID == "" {
		workspaceID = "default"
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := m.DB.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, scope, key, content, run_id, created_at
		FROM memory_facts WHERE workspace_id = $1
		ORDER BY created_at DESC LIMIT $2`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemoryFact
	for rows.Next() {
		var f MemoryFact
		if err := rows.Scan(&f.ID, &f.WorkspaceID, &f.UserID, &f.Scope, &f.Key, &f.Content, &f.RunID, &f.CreatedAt); err != nil {
			return nil, err
		}
		if userID != "" && f.UserID != "" && f.UserID != userID {
			continue
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (m *PostgresMemory) Close() error {
	if m == nil || m.DB == nil {
		return nil
	}
	return m.DB.Close()
}

// CollectMemoryNotes loads short + long facts for context assembly.
func CollectMemoryNotes(ctx context.Context, mem MemoryStore, runID, workspaceID, userID string) []string {
	if mem == nil {
		return nil
	}
	var notes []string
	if short, err := mem.ListShort(ctx, runID); err == nil {
		for _, f := range short {
			notes = append(notes, f.Content)
		}
	}
	if long, err := mem.ListLong(ctx, workspaceID, userID, 10); err == nil {
		for _, f := range long {
			notes = append(notes, f.Content)
		}
	}
	return notes
}

// HybridMemory uses Redis (or any MemoryStore) for short-term and another store for long-term.
type HybridMemory struct {
	Short MemoryStore
	Long  MemoryStore
}

func (m *HybridMemory) PutShort(ctx context.Context, runID, content string) error {
	if m.Short == nil {
		return errors.New("short memory not configured")
	}
	return m.Short.PutShort(ctx, runID, content)
}
func (m *HybridMemory) ListShort(ctx context.Context, runID string) ([]MemoryFact, error) {
	if m.Short == nil {
		return nil, nil
	}
	return m.Short.ListShort(ctx, runID)
}
func (m *HybridMemory) PutLong(ctx context.Context, fact MemoryFact) error {
	if m.Long == nil {
		return errors.New("long memory not configured")
	}
	return m.Long.PutLong(ctx, fact)
}
func (m *HybridMemory) ListLong(ctx context.Context, workspaceID, userID string, limit int) ([]MemoryFact, error) {
	if m.Long == nil {
		return nil, nil
	}
	return m.Long.ListLong(ctx, workspaceID, userID, limit)
}

// Ensure fmt stays imported for any remaining uses.
var _ = fmt.Sprintf
