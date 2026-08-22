package orchestrator

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/002_orchestrator_prompts.sql
var promptSchema string

// PromptStore is the durable publish path for workspace-scoped prompts.
// Published entries resolve by the same id/version keys as file prompts within a workspace.
type PromptStore interface {
	Publish(ctx context.Context, workspaceID string, entry PromptEntry) error
	List(ctx context.Context, workspaceID string) ([]PromptEntry, error)
	Get(ctx context.Context, workspaceID string, id, version string) (PromptEntry, error)
}

// NormalizeWorkspace returns a non-empty workspace id (default when blank).
func NormalizeWorkspace(workspaceID string) string {
	ws := strings.TrimSpace(workspaceID)
	if ws == "" {
		return "default"
	}
	return ws
}

// MemoryPromptStore is an in-process multi-tenant PromptStore.
type MemoryPromptStore struct {
	mu    sync.RWMutex
	byKey map[string]PromptEntry // workspace\0id\0version
}

// NewMemoryPromptStore returns an empty durable-in-memory prompt store.
func NewMemoryPromptStore() *MemoryPromptStore {
	return &MemoryPromptStore{byKey: make(map[string]PromptEntry)}
}

func wsPromptKey(workspaceID, id, version string) string {
	return NormalizeWorkspace(workspaceID) + "\x00" + id + "\x00" + version
}

func (s *MemoryPromptStore) Publish(_ context.Context, workspaceID string, entry PromptEntry) error {
	if s == nil {
		return fmt.Errorf("prompt store is nil")
	}
	if entry.ID == "" {
		return fmt.Errorf("prompt id is required")
	}
	if entry.Version == "" {
		entry.Version = "0"
	}
	if entry.Status == "" {
		entry.Status = "active"
	}
	ws := NormalizeWorkspace(workspaceID)
	entry.WorkspaceID = ws
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byKey == nil {
		s.byKey = make(map[string]PromptEntry)
	}
	s.byKey[wsPromptKey(ws, entry.ID, entry.Version)] = entry
	return nil
}

func (s *MemoryPromptStore) List(_ context.Context, workspaceID string) ([]PromptEntry, error) {
	if s == nil {
		return nil, fmt.Errorf("prompt store is nil")
	}
	ws := NormalizeWorkspace(workspaceID)
	prefix := ws + "\x00"
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PromptEntry, 0)
	for k, e := range s.byKey {
		if strings.HasPrefix(k, prefix) {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *MemoryPromptStore) Get(_ context.Context, workspaceID string, id, version string) (PromptEntry, error) {
	if s == nil {
		return PromptEntry{}, fmt.Errorf("prompt store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.byKey[wsPromptKey(workspaceID, id, version)]
	if !ok {
		return PromptEntry{}, fmt.Errorf("prompt %q version %q not found in workspace", id, version)
	}
	return e, nil
}

// PostgresPromptStore publishes workspace-scoped prompts to PostgreSQL.
type PostgresPromptStore struct {
	DB *sql.DB
}

// OpenPostgresPromptStore opens databaseURL, pings, and applies the prompts schema.
func OpenPostgresPromptStore(ctx context.Context, databaseURL string) (*PostgresPromptStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres prompts: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres prompts: %w", err)
	}
	if _, err := db.ExecContext(ctx, promptSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply prompts schema: %w", err)
	}
	return &PostgresPromptStore{DB: db}, nil
}

// Close closes the underlying DB.
func (s *PostgresPromptStore) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// Publish upserts a prompt row by (workspace_id, id, version).
func (s *PostgresPromptStore) Publish(ctx context.Context, workspaceID string, entry PromptEntry) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("postgres prompt store is nil")
	}
	if entry.ID == "" {
		return fmt.Errorf("prompt id is required")
	}
	if entry.Version == "" {
		entry.Version = "0"
	}
	if entry.Status == "" {
		entry.Status = "active"
	}
	ws := NormalizeWorkspace(workspaceID)
	entry.WorkspaceID = ws
	varsJSON, err := json.Marshal(entry.Variables)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO orchestrator_prompts
		(workspace_id, id, version, status, text, variables_json, model_class, approx_tokens, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (workspace_id, id, version) DO UPDATE SET
			status = EXCLUDED.status,
			text = EXCLUDED.text,
			variables_json = EXCLUDED.variables_json,
			model_class = EXCLUDED.model_class,
			approx_tokens = EXCLUDED.approx_tokens,
			updated_at = EXCLUDED.updated_at
	`, ws, entry.ID, entry.Version, entry.Status, entry.Text, varsJSON,
		nullStr(entry.ModelClass), entry.ApproxTokens, now)
	if err != nil {
		return fmt.Errorf("publish prompt: %w", err)
	}
	return nil
}

// List returns published prompts for one workspace only.
func (s *PostgresPromptStore) List(ctx context.Context, workspaceID string) ([]PromptEntry, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("postgres prompt store is nil")
	}
	ws := NormalizeWorkspace(workspaceID)
	rows, err := s.DB.QueryContext(ctx, `
		SELECT workspace_id, id, version, status, text, variables_json, model_class, approx_tokens
		FROM orchestrator_prompts WHERE workspace_id = $1 ORDER BY id, version`, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PromptEntry
	for rows.Next() {
		e, err := scanPromptRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Get returns one published prompt by workspace, id, and version.
func (s *PostgresPromptStore) Get(ctx context.Context, workspaceID string, id, version string) (PromptEntry, error) {
	if s == nil || s.DB == nil {
		return PromptEntry{}, fmt.Errorf("postgres prompt store is nil")
	}
	ws := NormalizeWorkspace(workspaceID)
	row := s.DB.QueryRowContext(ctx, `
		SELECT workspace_id, id, version, status, text, variables_json, model_class, approx_tokens
		FROM orchestrator_prompts WHERE workspace_id = $1 AND id = $2 AND version = $3`,
		ws, id, version)
	e, err := scanPromptRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return PromptEntry{}, fmt.Errorf("prompt %q version %q not found in workspace", id, version)
	}
	return e, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPromptRow(row rowScanner) (PromptEntry, error) {
	var e PromptEntry
	var vars []byte
	var modelClass sql.NullString
	if err := row.Scan(&e.WorkspaceID, &e.ID, &e.Version, &e.Status, &e.Text, &vars, &modelClass, &e.ApproxTokens); err != nil {
		return PromptEntry{}, err
	}
	e.ModelClass = modelClass.String
	if len(vars) > 0 && string(vars) != "null" {
		if err := json.Unmarshal(vars, &e.Variables); err != nil {
			return PromptEntry{}, fmt.Errorf("decode variables: %w", err)
		}
	}
	return e, nil
}
