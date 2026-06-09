package postgres

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListWorkspacesForUser(ctx context.Context, userID uuid.UUID) ([]Workspace, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT w.id, w.slug, w.name, w.notify_chat_id, w.meet_link, w.tz, w.owner_user_id
		FROM workspaces w
		WHERE w.owner_user_id = $1
			OR EXISTS (
				SELECT 1 FROM workspace_members m
				WHERE m.workspace_id = w.id AND m.user_id = $1
			)
		ORDER BY w.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkspaces(rows)
}

func (s *Store) CreateWorkspace(ctx context.Context, slug, name string, ownerID uuid.UUID) (Workspace, error) {
	var w Workspace
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspaces (slug, name, owner_user_id, tz)
		VALUES ($1, $2, $3, 'Asia/Almaty')
		RETURNING id, slug, name, notify_chat_id, meet_link, tz, owner_user_id`,
		slug, name, ownerID).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) GetWorkspace(ctx context.Context, id uuid.UUID) (Workspace, error) {
	var w Workspace
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id
		FROM workspaces WHERE id = $1`, id).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) GetWorkspaceBySlug(ctx context.Context, slug string) (Workspace, error) {
	var w Workspace
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id
		FROM workspaces WHERE slug = $1`, slug).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) GetWorkspaceByChatID(ctx context.Context, chatID int64) (Workspace, error) {
	var w Workspace
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id
		FROM workspaces WHERE notify_chat_id = $1`, chatID).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) UpdateWorkspace(ctx context.Context, id uuid.UUID, meetLink, tz string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE workspaces SET meet_link = $2, tz = $3, updated_at = now() WHERE id = $1`,
		id, meetLink, tz)
	return err
}

func (s *Store) LinkChat(ctx context.Context, workspaceID uuid.UUID, chatID int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE workspaces SET notify_chat_id = $2, chat_linked_at = now(), updated_at = now()
		WHERE id = $1`, workspaceID, chatID)
	return err
}

func (s *Store) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, user_id, telegram_username, role
		FROM workspace_members WHERE workspace_id = $1 ORDER BY telegram_username`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.TelegramUsername, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) AddMember(ctx context.Context, workspaceID uuid.UUID, username, role string) (Member, error) {
	username = normalizeUsername(username)
	var m Member
	err := s.pool.QueryRow(ctx, `
		INSERT INTO workspace_members (workspace_id, telegram_username, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, telegram_username) DO UPDATE SET role = EXCLUDED.role
		RETURNING id, workspace_id, user_id, telegram_username, role`,
		workspaceID, username, role).Scan(&m.ID, &m.WorkspaceID, &m.UserID, &m.TelegramUsername, &m.Role)
	return m, err
}

func (s *Store) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM workspace_members WHERE id = $1`, memberID)
	return err
}

func (s *Store) IsMemberDeveloper(ctx context.Context, workspaceID uuid.UUID, username string) (bool, error) {
	username = normalizeUsername(username)
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workspace_members
		WHERE workspace_id = $1 AND telegram_username = $2 AND role IN ('developer','admin','owner')`,
		workspaceID, username).Scan(&n)
	return n > 0, err
}

func (s *Store) UpsertPendingChat(ctx context.Context, userID, chatID int64, title string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO pending_chat_links (telegram_user_id, chat_id, chat_title)
		VALUES ($1, $2, $3)`, userID, chatID, title)
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

func scanWorkspaces(rows pgx.Rows) ([]Workspace, error) {
	var out []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func normalizeUsername(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "@")
	return strings.ToLower(s)
}
