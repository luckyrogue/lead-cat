package postgres

import (
	"context"

	"github.com/google/uuid"
)

type VCSLogins struct {
	GitHub string
	GitLab string
}

type MemberWithVCS struct {
	Member
	GitHubLogin string `json:"github_login"`
	GitLabLogin string `json:"gitlab_login"`
}

func (s *Store) ListMembersWithVCS(ctx context.Context, workspaceID uuid.UUID) ([]MemberWithVCS, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.workspace_id, m.user_id, m.telegram_username, m.role,
			COALESCE(v.github_login, ''), COALESCE(v.gitlab_login, '')
		FROM workspace_members m
		LEFT JOIN developer_vcs_map v
			ON v.workspace_id = m.workspace_id AND v.telegram_username = m.telegram_username
		WHERE m.workspace_id = $1
		ORDER BY m.telegram_username`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberWithVCS
	for rows.Next() {
		var m MemberWithVCS
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.TelegramUsername, &m.Role,
			&m.GitHubLogin, &m.GitLabLogin); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListVCSMaps(ctx context.Context, workspaceID uuid.UUID) (map[string]VCSLogins, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT telegram_username, github_login, gitlab_login
		FROM developer_vcs_map WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]VCSLogins)
	for rows.Next() {
		var tg, gh, gl string
		if err := rows.Scan(&tg, &gh, &gl); err != nil {
			return nil, err
		}
		out[tg] = VCSLogins{GitHub: gh, GitLab: gl}
	}
	return out, rows.Err()
}

func (s *Store) SetMemberVCS(ctx context.Context, workspaceID uuid.UUID, username, githubLogin, gitlabLogin string) error {
	username = normalizeUsername(username)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO developer_vcs_map (workspace_id, telegram_username, github_login, gitlab_login)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (workspace_id, telegram_username) DO UPDATE SET
			github_login = EXCLUDED.github_login,
			gitlab_login = EXCLUDED.gitlab_login`,
		workspaceID, username, githubLogin, gitlabLogin)
	return err
}

func (s *Store) UpsertMemberFromChat(ctx context.Context, workspaceID uuid.UUID, username, role string) error {
	username = normalizeUsername(username)
	if username == "" {
		return nil
	}
	if role == "" {
		role = "developer"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, telegram_username, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, telegram_username) DO NOTHING`,
		workspaceID, username, role)
	return err
}
