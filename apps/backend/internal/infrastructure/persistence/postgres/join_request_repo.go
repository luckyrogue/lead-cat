package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateJoinRequest(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO organization_join_requests (organization_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (organization_id, user_id) WHERE status = 'pending' DO NOTHING`,
		orgID, userID)
	return err
}

func (s *Store) ListJoinRequestsForUser(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.organization_id, o.name, r.status
		FROM organization_join_requests r
		JOIN organizations o ON o.id = r.organization_id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.JoinRequestView{}
	for rows.Next() {
		var v model.JoinRequestView
		if err := rows.Scan(&v.OrganizationID, &v.OrgName, &v.Status); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) ListPendingJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.user_id, '', u.email, r.created_at
		FROM organization_join_requests r
		JOIN platform_users u ON u.id = r.user_id
		WHERE r.organization_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.JoinRequestAdminView{}
	for rows.Next() {
		var v model.JoinRequestAdminView
		if err := rows.Scan(&v.RequestID, &v.UserID, &v.Name, &v.Email, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT user_id FROM organization_join_requests
		WHERE id = $1 AND organization_id = $2 AND status = 'pending'
		FOR UPDATE`, requestID, orgID).Scan(&userID)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'member') ON CONFLICT (organization_id, user_id) DO NOTHING`,
		orgID, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE organization_join_requests
		SET status = 'accepted', decided_at = now(), decided_by_user_id = $2
		WHERE id = $1`, requestID, deciderID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE organization_join_requests
		SET status = 'declined', decided_at = now(), decided_by_user_id = $3
		WHERE id = $1 AND organization_id = $2 AND status = 'pending'`,
		requestID, orgID, deciderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
