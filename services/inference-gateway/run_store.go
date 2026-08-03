package inference

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/001_inference_runs.sql
var inferenceRunsSchema string

// InferenceRun is the durable operational record for one gateway execution.
type InferenceRun struct {
	RunID          string
	RequestID      string
	TaskClass      string
	RequestedModel string
	ResolvedModel  string
	Provider       string
	Status         string
	Messages       []Message
	StartedAt      time.Time
	CompletedAt    *time.Time
}

// InferenceRunView is the safe read model returned by the run inspection API.
type InferenceRunView struct {
	RunID          string        `json:"run_id"`
	RequestID      string        `json:"request_id"`
	TaskClass      string        `json:"task_class"`
	RequestedModel string        `json:"requested_model,omitempty"`
	ResolvedModel  string        `json:"resolved_model"`
	Provider       string        `json:"provider,omitempty"`
	Status         string        `json:"status"`
	Result         *ChatResponse `json:"result,omitempty"`
	ErrorClass     string        `json:"error_class,omitempty"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    *time.Time    `json:"completed_at,omitempty"`
}

// RunStore is the persistence port used by the HTTP gateway.
type RunStore interface {
	StartRun(ctx context.Context, run InferenceRun) error
	CompleteRun(ctx context.Context, runID string, result ChatResponse, completedAt time.Time) error
	FailRun(ctx context.Context, runID, class, message string, completedAt time.Time) error
	GetRun(ctx context.Context, runID string) (InferenceRunView, error)
}

// PostgresRunStore persists inference run lifecycle and result metadata.
type PostgresRunStore struct {
	DB *sql.DB
}

// OpenPostgresRunStore opens PostgreSQL, verifies connectivity, and applies
// the small idempotent schema needed by this service.
func OpenPostgresRunStore(ctx context.Context, databaseURL string) (*PostgresRunStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}
	store := &PostgresRunStore{DB: db}
	if err := store.ensureSchema(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresRunStore) ensureSchema(ctx context.Context) error {
	if s == nil || s.DB == nil {
		return errors.New("PostgreSQL run store is not configured")
	}
	if _, err := s.DB.ExecContext(ctx, inferenceRunsSchema); err != nil {
		return fmt.Errorf("apply inference run schema: %w", err)
	}
	return nil
}

func (s *PostgresRunStore) StartRun(ctx context.Context, run InferenceRun) error {
	messages, err := json.Marshal(run.Messages)
	if err != nil {
		return fmt.Errorf("marshal run messages: %w", err)
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO inference_runs
		(run_id, request_id, task_class, requested_model, resolved_model, provider, status, input_messages, started_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		run.RunID, run.RequestID, run.TaskClass, run.RequestedModel, run.ResolvedModel,
		run.Provider, "running", messages, run.StartedAt)
	if err != nil {
		return fmt.Errorf("start inference run: %w", err)
	}
	return nil
}

func (s *PostgresRunStore) CompleteRun(ctx context.Context, runID string, result ChatResponse, completedAt time.Time) error {
	message, err := json.Marshal(result.Message)
	if err != nil {
		return fmt.Errorf("marshal result message: %w", err)
	}
	execResult, err := s.DB.ExecContext(ctx, `
		UPDATE inference_runs
		SET provider = $2, resolved_model = $3, status = 'succeeded', output_message = $4,
		    input_tokens = $5, output_tokens = $6, finish_reason = $7, completed_at = $8
		WHERE run_id = $1`, runID, result.Provider, result.Model, message,
		result.Usage.InputTokens, result.Usage.OutputTokens, result.FinishReason, completedAt)
	if err != nil {
		return fmt.Errorf("complete inference run: %w", err)
	}
	if rows, _ := execResult.RowsAffected(); rows != 1 {
		return fmt.Errorf("complete inference run: run %q not found", runID)
	}
	return nil
}

func (s *PostgresRunStore) FailRun(ctx context.Context, runID, class, message string, completedAt time.Time) error {
	execResult, err := s.DB.ExecContext(ctx, `
		UPDATE inference_runs
		SET status = 'failed', error_class = $2, error_message = $3, completed_at = $4
		WHERE run_id = $1`, runID, class, message, completedAt)
	if err != nil {
		return fmt.Errorf("fail inference run: %w", err)
	}
	if rows, _ := execResult.RowsAffected(); rows != 1 {
		return fmt.Errorf("fail inference run: run %q not found", runID)
	}
	return nil
}

func (s *PostgresRunStore) GetRun(ctx context.Context, runID string) (InferenceRunView, error) {
	var view InferenceRunView
	var requestedModel, provider, errorClassValue, errorMessage sql.NullString
	var finishReason sql.NullString
	var inputTokens, outputTokens sql.NullInt64
	var outputMessage []byte
	var completedAt sql.NullTime
	err := s.DB.QueryRowContext(ctx, `
		SELECT run_id, request_id, task_class, requested_model, resolved_model, provider,
		       status, output_message, input_tokens, output_tokens, finish_reason,
		       error_class, error_message, started_at, completed_at
		FROM inference_runs WHERE run_id = $1`, runID).Scan(
		&view.RunID, &view.RequestID, &view.TaskClass, &requestedModel, &view.ResolvedModel,
		&provider, &view.Status, &outputMessage, &inputTokens, &outputTokens, &finishReason,
		&errorClassValue, &errorMessage,
		&view.StartedAt, &completedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return InferenceRunView{}, fmt.Errorf("run %q not found", runID)
	}
	if err != nil {
		return InferenceRunView{}, fmt.Errorf("get inference run: %w", err)
	}
	view.RequestedModel = requestedModel.String
	view.Provider = provider.String
	view.ErrorClass = errorClassValue.String
	view.ErrorMessage = errorMessage.String
	if completedAt.Valid {
		view.CompletedAt = &completedAt.Time
	}
	if len(outputMessage) > 0 {
		var message Message
		if err := json.Unmarshal(outputMessage, &message); err != nil {
			return InferenceRunView{}, fmt.Errorf("decode stored run result: %w", err)
		}
		view.Result = &ChatResponse{
			Message:      message,
			Usage:        Usage{InputTokens: int(inputTokens.Int64), OutputTokens: int(outputTokens.Int64)},
			Provider:     view.Provider,
			Model:        view.ResolvedModel,
			FinishReason: finishReason.String,
		}
	}
	return view, nil
}

func errorClass(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr.Class != "" {
		return providerErr.Class
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	return "provider_error"
}
