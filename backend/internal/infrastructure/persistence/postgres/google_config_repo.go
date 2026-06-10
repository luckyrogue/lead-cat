package postgres

import (
	"context"

	"github.com/google/uuid"
)

// SetGoogleConfig stores the (already-encrypted) service-account JSON plus the
// impersonation subject and target calendar for a workspace.
func (s *Store) SetGoogleConfig(ctx context.Context, id uuid.UUID, encJSON []byte, subject, calendarID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE organizations
		SET google_sa_json_enc = $2, google_subject = $3, google_calendar_id = $4, updated_at = now()
		WHERE id = $1`, id, encJSON, subject, calendarID)
	return err
}

// GetGoogleConfig returns the encrypted SA JSON, subject, and calendar id.
func (s *Store) GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT google_sa_json_enc, google_subject, google_calendar_id
		FROM organizations WHERE id = $1`, id).Scan(&encJSON, &subject, &calendarID)
	return encJSON, subject, calendarID, err
}
