package postgres

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.organization_id, o.name, i.role
		FROM organization_invites i
		JOIN organizations o ON o.id = i.organization_id
		WHERE lower(i.email) = lower($1)
		  AND i.accepted_at IS NULL AND i.declined_at IS NULL AND i.expires_at > now()
		ORDER BY i.created_at`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.InviteView{}
	for rows.Next() {
		var v model.InviteView
		if err := rows.Scan(&v.InviteID, &v.OrganizationID, &v.OrgName, &v.Role); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orgID uuid.UUID
	var inviteEmail, role string
	err = tx.QueryRow(ctx, `
		SELECT organization_id, email, role
		FROM organization_invites
		WHERE id = $1 AND accepted_at IS NULL AND declined_at IS NULL AND expires_at > now()
		FOR UPDATE`, inviteID).Scan(&orgID, &inviteEmail, &role)
	if err != nil {
		return err
	}
	if !strings.EqualFold(inviteEmail, email) {
		return model.ErrInviteEmailMismatch
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3) ON CONFLICT (organization_id, user_id) DO NOTHING`,
		orgID, userID, role); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE organization_invites SET accepted_at = now() WHERE id = $1`, inviteID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	var inviteEmail string
	err := s.pool.QueryRow(ctx, `
		SELECT email FROM organization_invites
		WHERE id = $1 AND accepted_at IS NULL AND declined_at IS NULL AND expires_at > now()`,
		inviteID).Scan(&inviteEmail)
	if err != nil {
		return err
	}
	if !strings.EqualFold(inviteEmail, email) {
		return model.ErrInviteEmailMismatch
	}
	_, err = s.pool.Exec(ctx, `UPDATE organization_invites SET declined_at = now() WHERE id = $1`, inviteID)
	return err
}
