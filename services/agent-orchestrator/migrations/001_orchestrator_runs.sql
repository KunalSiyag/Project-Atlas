CREATE TABLE IF NOT EXISTS orchestrator_runs (
    run_id TEXT PRIMARY KEY,
    goal TEXT NOT NULL,
    status TEXT NOT NULL,
    result TEXT,
    error_message TEXT,
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,
    steps_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS orchestrator_runs_status_idx ON orchestrator_runs (status);
CREATE INDEX IF NOT EXISTS orchestrator_runs_updated_idx ON orchestrator_runs (updated_at DESC);

CREATE TABLE IF NOT EXISTS orchestrator_events (
    event_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES orchestrator_runs(run_id) ON DELETE CASCADE,
    step_id TEXT,
    event_type TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS orchestrator_events_run_idx ON orchestrator_events (run_id, created_at);
