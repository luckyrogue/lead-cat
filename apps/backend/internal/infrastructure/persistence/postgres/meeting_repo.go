package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

const meetingCols = `id, organization_id, organizer_user_id, dept, type, host,
	starts_at, ends_at, recurrence, name, description, google_event_id, meet_link, status,
	series_id, recurrence_until, recurrence_days`

const meetingColsM = `m.id, m.organization_id, m.organizer_user_id, m.dept, m.type, m.host,
	m.starts_at, m.ends_at, m.recurrence, m.name, m.description, m.google_event_id, m.meet_link, m.status,
	m.series_id, m.recurrence_until, m.recurrence_days`

func scanMeeting(row interface {
	Scan(dest ...any) error
}) (Meeting, error) {
	var m Meeting
	var daysRaw []byte
	err := row.Scan(&m.ID, &m.OrganizationID, &m.OrganizerUserID, &m.Dept, &m.Type, &m.Host,
		&m.StartsAt, &m.EndsAt, &m.Recurrence, &m.Name, &m.Description, &m.GoogleEventID, &m.MeetLink, &m.Status,
		&m.SeriesID, &m.RecurrenceUntil, &daysRaw)
	if err == nil && len(daysRaw) > 0 {
		err = json.Unmarshal(daysRaw, &m.RecurrenceDays)
	}
	return m, err
}

const insertMeetingSQL = `
	INSERT INTO meetings (organization_id, organizer_user_id, dept, type, host,
		starts_at, ends_at, recurrence, name, description, google_event_id, meet_link,
		series_id, recurrence_until, recurrence_days)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	RETURNING ` + meetingCols

func meetingInsertArgs(m Meeting) []any {
	var daysJSON any
	if len(m.RecurrenceDays) > 0 {
		b, _ := json.Marshal(m.RecurrenceDays)
		daysJSON = b
	}
	return []any{m.OrganizationID, m.OrganizerUserID, m.Dept, m.Type, m.Host,
		m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description, m.GoogleEventID, m.MeetLink,
		m.SeriesID, m.RecurrenceUntil, daysJSON}
}

const updateMeetingSQL = `
	UPDATE meetings SET dept=$3, type=$4, host=$5, starts_at=$6, ends_at=$7,
		recurrence=$8, name=$9, description=$10, updated_at=now()
	WHERE id=$1 AND organization_id=$2 AND status='scheduled'`

func updateMeetingArgs(organizationID, id uuid.UUID, m Meeting) []any {
	return []any{id, organizationID, m.Dept, m.Type, m.Host, m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description}
}

func (s *Store) CreateMeeting(ctx context.Context, m Meeting) (Meeting, error) {
	return scanMeeting(s.pool.QueryRow(ctx, insertMeetingSQL, meetingInsertArgs(m)...))
}

func (s *Store) CreateMeetingSeries(ctx context.Context, ms []Meeting, ps []MeetingParticipant) ([]Meeting, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	out := make([]Meeting, 0, len(ms))
	for _, m := range ms {
		created, err := scanMeeting(tx.QueryRow(ctx, insertMeetingSQL, meetingInsertArgs(m)...))
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			if _, err := tx.Exec(ctx, `
				INSERT INTO meeting_participants (meeting_id, employee_id, email)
				VALUES ($1, $2, $3) ON CONFLICT (meeting_id, email) DO NOTHING`, created.ID, p.EmployeeID, p.Email); err != nil {
				return nil, err
			}
		}
		created.Participants = ps
		out = append(out, created)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) AddParticipants(ctx context.Context, meetingID uuid.UUID, ps []MeetingParticipant) error {
	for _, p := range ps {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO meeting_participants (meeting_id, employee_id, email)
			VALUES ($1, $2, $3) ON CONFLICT (meeting_id, email) DO NOTHING`, meetingID, p.EmployeeID, p.Email); err != nil {
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

func (s *Store) RemoveParticipant(ctx context.Context, meetingID uuid.UUID, email string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM meeting_participants WHERE meeting_id = $1 AND email = $2`, meetingID, email)
	return err
}

func (s *Store) ListMeetings(ctx context.Context, organizationID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE organization_id = $1 ORDER BY starts_at DESC`, organizationID)
}

func (s *Store) ListMeetingsByOrganizer(ctx context.Context, organizationID, organizerID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE organization_id = $1 AND organizer_user_id = $2 ORDER BY starts_at DESC`, organizationID, organizerID)
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

func (s *Store) ListSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID, fromStart time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE series_id = $1 AND organization_id = $2 AND starts_at >= $3 AND status = 'scheduled'
		ORDER BY starts_at`, seriesID, organizationID, fromStart)
}

func (s *Store) ListSeriesAllOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE series_id = $1 AND organization_id = $2 AND status = 'scheduled'
		ORDER BY starts_at`, seriesID, organizationID)
}

func (s *Store) ListSeriesOccurrenceStarts(ctx context.Context, organizationID, seriesID uuid.UUID) ([]time.Time, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT starts_at FROM meetings
		WHERE series_id = $1 AND organization_id = $2`, seriesID, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []time.Time
	for rows.Next() {
		var t time.Time
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetMeeting(ctx context.Context, organizationID, id uuid.UUID) (Meeting, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+meetingCols+` FROM meetings WHERE id = $1 AND organization_id = $2`, id, organizationID)
	return scanMeeting(row)
}

func (s *Store) UpdateMeeting(ctx context.Context, organizationID, id uuid.UUID, m Meeting) error {
	ct, err := s.pool.Exec(ctx, updateMeetingSQL, updateMeetingArgs(organizationID, id, m)...)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrMeetingNotEditable
	}
	return nil
}

func (s *Store) UpdateMeetingsTx(ctx context.Context, organizationID uuid.UUID, ms []Meeting) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, m := range ms {
		ct, err := tx.Exec(ctx, updateMeetingSQL, updateMeetingArgs(organizationID, m.ID, m)...)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return ErrMeetingNotEditable
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) ListMeetingsByOrganizerTelegram(ctx context.Context, telegramID int64) ([]MeetingWithTZ, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+meetingColsM+`, w.tz
		FROM meetings m
		JOIN platform_users pu ON pu.id = m.organizer_user_id
		JOIN organizations w ON w.id = m.organization_id
		WHERE pu.telegram_id = $1 AND m.status = 'scheduled' AND m.starts_at > now()
		ORDER BY m.starts_at`, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MeetingWithTZ
	for rows.Next() {
		var mt MeetingWithTZ
		var daysRaw []byte
		if err := rows.Scan(&mt.ID, &mt.OrganizationID, &mt.OrganizerUserID, &mt.Dept, &mt.Type, &mt.Host,
			&mt.StartsAt, &mt.EndsAt, &mt.Recurrence, &mt.Name, &mt.Description, &mt.GoogleEventID, &mt.MeetLink, &mt.Status,
			&mt.SeriesID, &mt.RecurrenceUntil, &daysRaw,
			&mt.TZ); err != nil {
			return nil, err
		}
		if len(daysRaw) > 0 {
			if err := json.Unmarshal(daysRaw, &mt.RecurrenceDays); err != nil {
				return nil, err
			}
		}
		out = append(out, mt)
	}
	return out, rows.Err()
}

func (s *Store) ListScheduleForEmail(ctx context.Context, email string, from, to time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT DISTINCT `+meetingColsM+`
		FROM meetings m
		LEFT JOIN meeting_participants mp ON mp.meeting_id = m.id
		LEFT JOIN platform_users pu ON pu.id = m.organizer_user_id
		WHERE (mp.email = $1 OR pu.email = $1)
			AND m.status = 'scheduled' AND m.starts_at >= $2 AND m.starts_at < $3
		ORDER BY m.starts_at`, email, from, to)
}

func (s *Store) ListMeetingsOverlapping(ctx context.Context, emails []string, from, to time.Time) ([]Meeting, error) {
	if len(emails) == 0 {
		return nil, nil
	}
	return s.queryMeetings(ctx, `
		SELECT DISTINCT `+meetingColsM+`
		FROM meetings m
		LEFT JOIN meeting_participants mp ON mp.meeting_id = m.id
		LEFT JOIN platform_users pu ON pu.id = m.organizer_user_id
		WHERE (mp.email = ANY($1) OR pu.email = ANY($1))
			AND m.status = 'scheduled'
			AND m.starts_at < $3 AND m.ends_at > $2
		ORDER BY m.starts_at`, emails, from, to)
}

func (s *Store) CancelMeeting(ctx context.Context, organizationID, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now()
		WHERE id = $1 AND organization_id = $2 AND status = 'scheduled'`, id, organizationID)
	return err
}

func (s *Store) CancelSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID, fromStart time.Time) (int, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now()
		WHERE series_id = $1 AND organization_id = $2 AND starts_at >= $3 AND status = 'scheduled'`,
		seriesID, organizationID, fromStart)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

func (s *Store) CancelAllSeriesOccurrences(ctx context.Context, organizationID, seriesID uuid.UUID) (int, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now()
		WHERE series_id = $1 AND organization_id = $2 AND status = 'scheduled'`,
		seriesID, organizationID)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}

func (s *Store) SetSeriesRecurrenceUntil(ctx context.Context, organizationID, seriesID uuid.UUID, until time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET recurrence_until = $3, updated_at = now()
		WHERE series_id = $1 AND organization_id = $2 AND status = 'scheduled'`,
		seriesID, organizationID, until)
	return err
}

func meetingFilter(organizationID uuid.UUID, f model.MeetingFilter) (string, []any) {
	args := []any{organizationID}
	where := "organization_id = $1"
	add := func(expr string, val any) {
		args = append(args, val)
		where += fmt.Sprintf(" AND %s $%d", expr, len(args))
	}
	switch f.Status {
	case "scheduled", "cancelled":
		add("status =", f.Status)
	}
	if f.From != nil {
		add("starts_at >=", *f.From)
	}
	if f.To != nil {
		add("starts_at <", *f.To)
	}
	if f.Dept != "" {
		add("dept ILIKE", "%"+f.Dept+"%")
	}
	if f.Organizer != nil {
		add("organizer_user_id =", *f.Organizer)
	}
	return where, args
}

func (s *Store) ListMeetingsFiltered(ctx context.Context, organizationID uuid.UUID, f model.MeetingFilter) ([]Meeting, error) {
	where, args := meetingFilter(organizationID, f)
	return s.queryMeetings(ctx, `SELECT `+meetingCols+` FROM meetings WHERE `+where+` ORDER BY starts_at DESC`, args...)
}

func (s *Store) ListUpcomingMeetings(ctx context.Context, until time.Time) ([]Meeting, error) {
	return s.queryMeetings(ctx, `
		SELECT `+meetingCols+` FROM meetings
		WHERE status = 'scheduled' AND starts_at > now() AND starts_at <= $1
		ORDER BY starts_at`, until)
}

const ReminderOffsetCreated = model.ReminderOffsetCreated

func (s *Store) TryClaimReminder(ctx context.Context, meetingID uuid.UUID, telegramID int64, offset int) (bool, error) {
	ct, err := s.pool.Exec(ctx, `
		INSERT INTO meeting_reminders (meeting_id, telegram_id, offset_minutes)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, meetingID, telegramID, offset)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}
