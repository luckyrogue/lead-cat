package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const auditCols = `id, actor_user_id, actor_telegram_id, actor_email, action, target_kind, target_id, details, created_at`

// InsertAuditEntry writes a new row. details may be empty/nil; we coerce to '{}'.
func (s *Store) InsertAuditEntry(ctx context.Context, e AuditEntry) error {
	if len(e.Details) == 0 {
		e.Details = json.RawMessage(`{}`)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO admin_audit_log (actor_user_id, actor_telegram_id, actor_email, action, target_kind, target_id, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ActorUserID, e.ActorTelegramID, e.ActorEmail, e.Action, e.TargetKind, e.TargetID, []byte(e.Details))
	return err
}

// ListAuditEntries returns entries by created_at DESC. Filters are AND-combined.
func (s *Store) ListAuditEntries(ctx context.Context, f AuditFilter) ([]AuditEntry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	q := `SELECT ` + auditCols + ` FROM admin_audit_log WHERE 1=1`
	args := []any{}
	if f.Action != "" {
		args = append(args, f.Action)
		q += fmt.Sprintf(" AND action = $%d", len(args))
	}
	if f.ActorEmail != "" {
		args = append(args, f.ActorEmail)
		q += fmt.Sprintf(" AND actor_email = $%d", len(args))
	}
	args = append(args, limit)
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var detailsRaw []byte
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorTelegramID, &e.ActorEmail, &e.Action, &e.TargetKind, &e.TargetID, &detailsRaw, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Details = detailsRaw
		out = append(out, e)
	}
	return out, rows.Err()
}

// EnsureDefaultOrganizationID returns the single Lead Cat organization id, creating
// it on first call. Idempotent — safe under concurrency thanks to the slug
// uniqueness constraint on the organizations table.
//
// ownerUserID may be uuid.Nil — in that case the organization is created with
// owner_user_id = NULL (which is permitted by the FK definition).
func (s *Store) EnsureDefaultOrganizationID(ctx context.Context, defaultTZ, defaultMeetLink string, ownerUserID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM organizations WHERE name = 'Lead Cat' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}

	// Build owner arg as nil-when-zero so the FK accepts a missing owner.
	var ownerArg any
	if ownerUserID != uuid.Nil {
		ownerArg = ownerUserID
	} else {
		ownerArg = nil
	}

	// Try INSERT. ON CONFLICT DO NOTHING swallows slug uniqueness races;
	// re-SELECT to pick up the winner's row.
	if err := s.pool.QueryRow(ctx, `
		INSERT INTO organizations (slug, name, owner_user_id, tz, meet_link)
		VALUES ('lead-cat', 'Lead Cat', $1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id`, ownerArg, defaultTZ, defaultMeetLink).Scan(&id); err == nil {
		return id, nil
	}
	// Race: re-select.
	if err := s.pool.QueryRow(ctx, `SELECT id FROM organizations WHERE name = 'Lead Cat' LIMIT 1`).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
