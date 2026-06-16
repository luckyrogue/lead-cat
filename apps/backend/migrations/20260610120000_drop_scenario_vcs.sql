
DROP TABLE IF EXISTS scenario_run_steps;
DROP TABLE IF EXISTS scenario_runs;
DROP TABLE IF EXISTS scenarios;
DROP TABLE IF EXISTS developer_vcs_map;
ALTER TABLE workspaces
  DROP COLUMN IF EXISTS vcs_provider,
  DROP COLUMN IF EXISTS vcs_token_enc,
  DROP COLUMN IF EXISTS vcs_namespace,
  DROP COLUMN IF EXISTS vcs_base_url;

ALTER TABLE workspaces
  ADD COLUMN IF NOT EXISTS vcs_provider TEXT NOT NULL DEFAULT 'github',
  ADD COLUMN IF NOT EXISTS vcs_token_enc BYTEA,
  ADD COLUMN IF NOT EXISTS vcs_namespace TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS vcs_base_url TEXT;

CREATE TABLE IF NOT EXISTS developer_vcs_map (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    telegram_username TEXT NOT NULL,
    vcs_login TEXT NOT NULL,
    UNIQUE (workspace_id, telegram_username)
);

CREATE TABLE IF NOT EXISTS scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    definition JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scenario_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL REFERENCES scenarios(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'running',
    trigger TEXT NOT NULL,
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS scenario_run_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES scenario_runs(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    status TEXT NOT NULL,
    output TEXT NOT NULL DEFAULT '',
    duration_ms INT NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
