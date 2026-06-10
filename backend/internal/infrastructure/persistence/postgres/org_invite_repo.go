package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// CreateInvite inserts a new pending invitation row and returns it.
func (s *Store) CreateInvite(
	ctx context.Context,
	orgID uuid.UUID,
	email, role string,
	tokenHash []byte,
	expiresAt time.Time,
	createdBy uuid.UUID,
) (OrganizationInvite, error) {
	var inv OrganizationInvite
	err := s.pool.QueryRow(ctx, `
		INSERT INTO organization_invites
			(organization_id, email, role, token_hash, expires_at, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, organization_id, email, role, token_hash, expires_at,
		          accepted_at, created_by_user_id, created_at`,
		orgID, email, role, tokenHash, expiresAt, createdBy,
	).Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.TokenHash,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedByUserID, &inv.CreatedAt,
	)
	return inv, err
}

// ListInvites returns pending (not yet accepted) invitations for an organisation.
func (s *Store) ListInvites(ctx context.Context, orgID uuid.UUID) ([]OrganizationInvite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, email, role, token_hash, expires_at,
		       accepted_at, created_by_user_id, created_at
		FROM organization_invites
		WHERE organization_id = $1 AND accepted_at IS NULL
		ORDER BY created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []OrganizationInvite
	for rows.Next() {
		var inv OrganizationInvite
		if err := rows.Scan(
			&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.TokenHash,
			&inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedByUserID, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// DeleteInvite removes an invitation by (orgID, inviteID).
func (s *Store) DeleteInvite(ctx context.Context, orgID, inviteID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM organization_invites
		WHERE organization_id = $1 AND id = $2`,
		orgID, inviteID)
	return err
}

// AcceptInvitesForEmail accepts all pending invitations whose email matches
// (case-insensitive), creates corresponding organization_members rows, and
// returns the number of invitations accepted.
func (s *Store) AcceptInvitesForEmail(ctx context.Context, email string, userID uuid.UUID) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Fetch all pending invites for this email.
	rows, err := tx.Query(ctx, `
		SELECT id, organization_id, role
		FROM organization_invites
		WHERE lower(email) = lower($1) AND accepted_at IS NULL
		FOR UPDATE`, email)
	if err != nil {
		return 0, err
	}

	type pendingInvite struct {
		id    uuid.UUID
		orgID uuid.UUID
		role  string
	}
	var invites []pendingInvite
	for rows.Next() {
		var inv pendingInvite
		if err := rows.Scan(&inv.id, &inv.orgID, &inv.role); err != nil {
			rows.Close()
			return 0, err
		}
		invites = append(invites, inv)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, inv := range invites {
		// Insert member; if they're already a member, skip silently.
		_, err := tx.Exec(ctx, `
			INSERT INTO organization_members (organization_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (organization_id, user_id) DO NOTHING`,
			inv.orgID, userID, inv.role)
		if err != nil {
			return 0, err
		}

		// Mark invite as accepted.
		_, err = tx.Exec(ctx, `
			UPDATE organization_invites SET accepted_at = now()
			WHERE id = $1`, inv.id)
		if err != nil {
			return 0, err
		}
		count++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return count, nil
}
