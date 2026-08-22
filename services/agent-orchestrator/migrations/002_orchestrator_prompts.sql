-- Versioned, workspace-scoped prompt publish path (file hot-reload + Postgres).
CREATE TABLE IF NOT EXISTS orchestrator_prompts (
    workspace_id TEXT NOT NULL DEFAULT 'default',
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    text TEXT NOT NULL,
    variables_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    model_class TEXT,
    approx_tokens INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, id, version)
);

CREATE INDEX IF NOT EXISTS orchestrator_prompts_ws_status_idx
    ON orchestrator_prompts (workspace_id, id, status);
