-- +goose Up
ALTER TABLE developer_vcs_map
    ADD COLUMN IF NOT EXISTS github_login TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS gitlab_login TEXT NOT NULL DEFAULT '';

UPDATE developer_vcs_map
SET github_login = vcs_login
WHERE github_login = '' AND vcs_login <> '';

ALTER TABLE developer_vcs_map DROP COLUMN IF EXISTS vcs_login;

-- +goose Down
ALTER TABLE developer_vcs_map ADD COLUMN IF NOT EXISTS vcs_login TEXT NOT NULL DEFAULT '';

UPDATE developer_vcs_map
SET vcs_login = COALESCE(NULLIF(github_login, ''), gitlab_login)
WHERE vcs_login = '';

ALTER TABLE developer_vcs_map DROP COLUMN IF EXISTS github_login;
ALTER TABLE developer_vcs_map DROP COLUMN IF EXISTS gitlab_login;
