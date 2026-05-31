package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const meetingCols = `id, workspace_id, organizer_user_id, dept, type, host,
	starts_at, ends_at, recurrence, name, description, google_event_id, meet_link, status`

func scanMeeting(row interface {
	Scan(dest ...any) error
}) (Meeting, error) {
	var m Meeting
	err := row.Scan(&m.ID, &m.WorkspaceID, &m.OrganizerUserID, &m.Dept, &m.Type, &m.Host,
		&m.StartsAt, &m.EndsAt, &m.Recurrence, &m.Name, &m.Description, &m.GoogleEventID, &m.MeetLink, &m.Status)
	return m, err
}

func (s *Store) CreateMeeting(ctx context.Context, m Meeting) (Meeting, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO meetings (workspace_id, organizer_user_id, dept, type, host,
			starts_at, ends_at, recurrence, name, description, google_event_id, meet_link)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING `+meetingCols,
		m.WorkspaceID, m.OrganizerUserID, m.Dept, m.Type, m.Host,
		m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description, m.GoogleEventID, m.MeetLink)
	return scanMeeting(row)
}

func (s *Store) AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []MeetingParticipant) error {
	for _, p := range ps {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO meeting_participants (meeting_id, employee_id, email)
			VALUES ($1, $2, $3)`, meetingID, p.EmployeeID, p.Email); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]MeetingParticipant, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT employee_id, email FROM meeting_participants WHERE meeting_id = $1`, meetingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MeetingParticipant
	for rows.Next() {
		var p MeetingParticipant
		if err := rows.Scan(&p.EmployeeID, &p.Email); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListMeetings(ctx context.Context, workspaceID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE workspace_id = $1 ORDER BY starts_at DESC`, workspaceID)
}

func (s *Store) ListMeetingsByOrganizer(ctx context.Context, workspaceID, organizerID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE workspace_id = $1 AND organizer_user_id = $2 ORDER BY starts_at DESC`, workspaceID, organizerID)
}

func (s *Store) queryMeetings(ctx context.Context, sql string, args ...any) ([]Meeting, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Meeting
	for rows.Next() {
		m, err := scanMeeting(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMeeting(ctx context.Context, workspaceID, id uuid.UUID) (Meeting, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+meetingCols+` FROM meetings WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	return scanMeeting(row)
}

func (s *Store) CancelMeeting(ctx context.Context, workspaceID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now()
		WHERE id = $1 AND workspace_id = $2 AND status = 'scheduled'`, id, workspaceID)
	return err
}

func (s *Store) ListUpcomingMeetings(ctx context.Context, until time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE status = 'scheduled' AND starts_at > now() AND starts_at <= $1
		ORDER BY starts_at`, until)
}

// ReminderOffsetCreated is the sentinel offset_minutes value reserved in
// meeting_reminders for the "meeting created" notification dedup. It never
// collides with real reminder offsets (10/15/30/60/120/1440).
const ReminderOffsetCreated = -1

// TryClaimReminder atomically records that (meeting, telegram, offset) is being
// reminded. Returns true if this call claimed it (caller should send), false if
// it was already claimed.
func (s *Store) TryClaimReminder(ctx context.Context, meetingID uuid.UUID, telegramID int64, offset int) (bool, error) {
	ct, err := s.pool.Exec(ctx, `
		INSERT INTO meeting_reminders (meeting_id, telegram_id, offset_minutes)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, meetingID, telegramID, offset)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}
