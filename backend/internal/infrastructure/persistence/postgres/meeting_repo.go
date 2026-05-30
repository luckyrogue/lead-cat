package postgres

import (
	"context"

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

func (s *Store) GetMeeting(ctx context.Context, id uuid.UUID) (Meeting, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+meetingCols+` FROM meetings WHERE id = $1`, id)
	return scanMeeting(row)
}

func (s *Store) CancelMeeting(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now() WHERE id = $1`, id)
	return err
}
