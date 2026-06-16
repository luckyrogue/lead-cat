
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE platform_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_sub TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    telegram_id BIGINT UNIQUE,
    avatar_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    notify_chat_id BIGINT,
    meet_link TEXT NOT NULL DEFAULT '',
    tz TEXT NOT NULL DEFAULT 'Asia/Almaty',
    owner_user_id UUID REFERENCES platform_users(id),
    vcs_provider TEXT NOT NULL DEFAULT 'github',
    vcs_token_enc BYTEA,
    vcs_namespace TEXT NOT NULL DEFAULT '',
    vcs_base_url TEXT,
    chat_linked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX workspaces_notify_chat_id_unique ON workspaces (notify_chat_id) WHERE notify_chat_id IS NOT NULL;

CREATE TABLE workspace_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID REFERENCES platform_users(id) ON DELETE SET NULL,
    telegram_username TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'developer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, telegram_username)
);

CREATE TABLE developer_vcs_map (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    telegram_username TEXT NOT NULL,
    vcs_login TEXT NOT NULL,
    UNIQUE (workspace_id, telegram_username)
);

CREATE TABLE pending_chat_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    telegram_user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    chat_title TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    definition JSONB NOT NULL DEFAULT '{"nodes":[],"edges":[]}',
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scenario_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL REFERENCES scenarios(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'running',
    trigger TEXT NOT NULL DEFAULT 'manual',
    context JSONB NOT NULL DEFAULT '{}',
    error TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE scenario_run_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id UUID NOT NULL REFERENCES scenario_runs(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    node_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'running',
    output JSONB NOT NULL DEFAULT '{}',
    duration_ms INT NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX scenario_runs_scenario_id_idx ON scenario_runs (scenario_id, started_at DESC);
CREATE INDEX scenarios_workspace_id_idx ON scenarios (workspace_id);

DROP TABLE IF EXISTS scenario_run_steps;
DROP TABLE IF EXISTS scenario_runs;
DROP TABLE IF EXISTS scenarios;
DROP TABLE IF EXISTS pending_chat_links;
DROP TABLE IF EXISTS developer_vcs_map;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS platform_users;
