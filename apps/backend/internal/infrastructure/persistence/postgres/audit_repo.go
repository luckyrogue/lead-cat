package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const auditCols = `id, actor_user_id, actor_telegram_id, actor_email, actor_kind, action, target_kind, target_id, details, created_at`

func (s *Store) InsertAuditEntry(ctx context.Context, e AuditEntry) error {
	if len(e.Details) == 0 {
		e.Details = json.RawMessage(`{}`)
	}
	kind := e.ActorKind
	if kind == "" {
		kind = "bot"
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO admin_audit_log (actor_user_id, actor_telegram_id, actor_email, actor_kind, action, target_kind, target_id, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ActorUserID, e.ActorTelegramID, e.ActorEmail, kind, e.Action, e.TargetKind, e.TargetID, []byte(e.Details))
	return err
}

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
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorTelegramID, &e.ActorEmail, &e.ActorKind, &e.Action, &e.TargetKind, &e.TargetID, &detailsRaw, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Details = detailsRaw
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) EnsureDefaultOrganizationID(ctx context.Context, defaultTZ, defaultMeetLink string, ownerUserID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM organizations WHERE name = 'Lead Cat' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}

	var ownerArg any
	if ownerUserID != uuid.Nil {
		ownerArg = ownerUserID
	} else {
		ownerArg = nil
	}

	if err := s.pool.QueryRow(ctx, `
		INSERT INTO organizations (slug, name, owner_user_id, tz, meet_link)
		VALUES ('lead-cat', 'Lead Cat', $1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id`, ownerArg, defaultTZ, defaultMeetLink).Scan(&id); err == nil {
		return id, nil
	}

	if err := s.pool.QueryRow(ctx, `SELECT id FROM organizations WHERE name = 'Lead Cat' LIMIT 1`).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Store) DefaultOrganizationWithGoogle(ctx context.Context) (uuid.UUID, bool, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM organizations
		WHERE name = 'Lead Cat' AND google_sa_json_enc IS NOT NULL AND google_subject <> ''
		LIMIT 1`).Scan(&id)
	if IsNotFound(err) {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, err
	}
	return id, true, nil
}
