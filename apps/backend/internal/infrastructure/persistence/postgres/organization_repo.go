package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]Organization, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id, o.slug, o.name, o.notify_chat_id, o.meet_link, o.tz, o.owner_user_id
		FROM organizations o
		JOIN organization_members m ON m.organization_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanOrganizations(rows)
}

// CreateOrganization inserts a new organization and adds ownerUserID as its
// owner member in a single transaction.
func (s *Store) CreateOrganization(ctx context.Context, name, slug string, ownerUserID uuid.UUID) (Organization, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Organization{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var w Organization
	err = tx.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, owner_user_id, tz)
		VALUES ($1, $2, $3, 'Asia/Almaty')
		RETURNING id, slug, name, notify_chat_id, meet_link, tz, owner_user_id`,
		name, slug, ownerUserID).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	if err != nil {
		return Organization{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')`,
		w.ID, ownerUserID)
	if err != nil {
		return Organization{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Organization{}, err
	}
	return w, nil
}

// GetOrgMember returns the member row for (orgID, userID). Returns (Member{}, false, nil) if not found.
func (s *Store) GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (Member, bool, error) {
	var m Member
	err := s.pool.QueryRow(ctx, `
		SELECT id, organization_id, user_id, telegram_username, role, invited_email
		FROM organization_members
		WHERE organization_id = $1 AND user_id = $2`,
		orgID, userID).Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.TelegramUsername, &m.Role, &m.InvitedEmail)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Member{}, false, nil
		}
		return Member{}, false, err
	}
	return m, true, nil
}

// ListOrgMembers returns all members of an organisation ordered by telegram_username.
func (s *Store) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT om.id, om.organization_id, om.user_id, om.telegram_username, om.role, om.invited_email,
		       COALESCE(pu.email, '')
		FROM organization_members om
		LEFT JOIN platform_users pu ON pu.id = om.user_id
		WHERE om.organization_id = $1
		ORDER BY om.telegram_username`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.TelegramUsername, &m.Role, &m.InvitedEmail, &m.Email); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// UpdateMemberRole sets the role for the member identified by (orgID, userID).
func (s *Store) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE organization_members SET role = $3
		WHERE organization_id = $1 AND user_id = $2`,
		orgID, userID, role)
	return err
}

// RemoveMember deletes the member identified by (orgID, userID).
func (s *Store) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM organization_members
		WHERE organization_id = $1 AND user_id = $2`,
		orgID, userID)
	return err
}

func (s *Store) GetOrganization(ctx context.Context, id uuid.UUID) (Organization, error) {
	var w Organization
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id
		FROM organizations WHERE id = $1`, id).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) GetOrganizationBySlug(ctx context.Context, slug string) (Organization, error) {
	var w Organization
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id
		FROM organizations WHERE slug = $1`, slug).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) GetOrganizationByChatID(ctx context.Context, chatID int64) (Organization, error) {
	var w Organization
	err := s.pool.QueryRow(ctx, `
		SELECT id, slug, name, notify_chat_id, meet_link, tz, owner_user_id
		FROM organizations WHERE notify_chat_id = $1`, chatID).Scan(
		&w.ID, &w.Slug, &w.Name, &w.NotifyChatID, &w.MeetLink, &w.TZ, &w.OwnerUserID)
	return w, err
}

func (s *Store) UpdateOrganization(ctx context.Context, id uuid.UUID, meetLink, tz string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE organizations SET meet_link = $2, tz = $3, updated_at = now() WHERE id = $1`,
		id, meetLink, tz)
	return err
}

func (s *Store) LinkChat(ctx context.Context, organizationID uuid.UUID, chatID int64) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE organizations SET notify_chat_id = $2, chat_linked_at = now(), updated_at = now()
		WHERE id = $1`, organizationID, chatID)
	return err
}

func (s *Store) ListMembers(ctx context.Context, organizationID uuid.UUID) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, user_id, telegram_username, role
		FROM organization_members WHERE organization_id = $1 ORDER BY telegram_username`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.TelegramUsername, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (Member, error) {
	username = normalizeUsername(username)
	var m Member
	err := s.pool.QueryRow(ctx, `
		INSERT INTO organization_members (organization_id, telegram_username, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, telegram_username) DO UPDATE SET role = EXCLUDED.role
		RETURNING id, organization_id, user_id, telegram_username, role`,
		organizationID, username, role).Scan(&m.ID, &m.OrganizationID, &m.UserID, &m.TelegramUsername, &m.Role)
	return m, err
}

func (s *Store) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM organization_members WHERE id = $1`, memberID)
	return err
}

func (s *Store) IsMemberDeveloper(ctx context.Context, organizationID uuid.UUID, username string) (bool, error) {
	username = normalizeUsername(username)
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM organization_members
		WHERE organization_id = $1 AND telegram_username = $2 AND role IN ('developer','admin','owner')`,
		organizationID, username).Scan(&n)
	return n > 0, err
}

func (s *Store) UpsertPendingChat(ctx context.Context, userID, chatID int64, title string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO pending_chat_links (telegram_user_id, chat_id, chat_title)
		VALUES ($1, $2, $3)`, userID, chatID, title)
	return err
}

func (s *Store) UpsertMemberFromChat(ctx context.Context, organizationID uuid.UUID, username, role string) error {
	username = normalizeUsername(username)
	if username == "" {
		return nil
	}
	if role == "" {
		role = "developer"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO organization_members (organization_id, telegram_username, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, telegram_username) DO NOTHING`,
		organizationID, username, role)
	return err
}

func scanOrganizations(rows pgx.Rows) ([]Organization, error) {
	var out []Organization
	for rows.Next() {
		var w Organization
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
