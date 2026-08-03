CREATE TABLE IF NOT EXISTS inference_runs (
    run_id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,
    task_class TEXT NOT NULL,
    requested_model TEXT,
    resolved_model TEXT NOT NULL,
    provider TEXT,
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    input_messages JSONB NOT NULL,
    output_message JSONB,
    input_tokens INTEGER,
    output_tokens INTEGER,
    finish_reason TEXT,
    error_class TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS inference_runs_request_id_idx ON inference_runs (request_id);
CREATE INDEX IF NOT EXISTS inference_runs_started_at_idx ON inference_runs (started_at DESC);
