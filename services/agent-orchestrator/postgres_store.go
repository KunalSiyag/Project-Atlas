package orchestrator

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

//go:embed migrations/001_orchestrator_runs.sql
var orchestratorSchema string

// PostgresStore persists runs and events in PostgreSQL.
type PostgresStore struct {
	DB *sql.DB
}

// OpenPostgresStore opens a database URL, pings, and applies schema.
func OpenPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	store := &PostgresStore{DB: db}
	if _, err := db.ExecContext(ctx, orchestratorSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return store, nil
}

func (s *PostgresStore) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *PostgresStore) SaveRun(ctx context.Context, run Run) error {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO orchestrator_runs
		(run_id, goal, status, result, error_message, cancel_requested, steps_json, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (run_id) DO UPDATE SET
			goal = EXCLUDED.goal,
			status = EXCLUDED.status,
			result = EXCLUDED.result,
			error_message = EXCLUDED.error_message,
			cancel_requested = EXCLUDED.cancel_requested,
			steps_json = EXCLUDED.steps_json,
			updated_at = EXCLUDED.updated_at
	`, run.RunID, run.Goal, run.Status, nullStr(run.Result), nullStr(run.Error),
		run.CancelRequested, steps, run.CreatedAt, run.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save run: %w", err)
	}
	return nil
}

func (s *PostgresStore) GetRun(ctx context.Context, runID string) (Run, error) {
	var run Run
	var result, errMsg sql.NullString
	var steps []byte
	err := s.DB.QueryRowContext(ctx, `
		SELECT run_id, goal, status, result, error_message, cancel_requested, steps_json, created_at, updated_at
		FROM orchestrator_runs WHERE run_id = $1`, runID).Scan(
		&run.RunID, &run.Goal, &run.Status, &result, &errMsg, &run.CancelRequested, &steps, &run.CreatedAt, &run.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, errors.New("run not found")
	}
	if err != nil {
		return Run{}, fmt.Errorf("get run: %w", err)
	}
	run.Result = result.String
	run.Error = errMsg.String
	if err := json.Unmarshal(steps, &run.Steps); err != nil {
		return Run{}, fmt.Errorf("decode steps: %w", err)
	}
	return run, nil
}

func (s *PostgresStore) AppendEvent(ctx context.Context, event Event) error {
	if event.EventID == "" {
		event.EventID = newID()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO orchestrator_events (event_id, run_id, step_id, event_type, message, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		event.EventID, event.RunID, nullStr(event.StepID), event.Type, event.Message, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListEvents(ctx context.Context, runID string) ([]Event, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT event_id, run_id, step_id, event_type, message, created_at
		FROM orchestrator_events WHERE run_id = $1 ORDER BY created_at ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var stepID sql.NullString
		if err := rows.Scan(&e.EventID, &e.RunID, &stepID, &e.Type, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.StepID = stepID.String
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
