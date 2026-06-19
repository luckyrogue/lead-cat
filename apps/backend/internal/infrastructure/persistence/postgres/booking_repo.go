package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

const bookingCols = `id, host_user_id, organization_id, slug, title, description, duration_mins,
	active, timezone, avail_weekdays, avail_start_minute, avail_end_minute, created_at, updated_at`

func scanBookingRow(row rowScanner) (model.BookingEventType, error) {
	var et model.BookingEventType
	var weekdays []int32
	if err := row.Scan(&et.ID, &et.HostUserID, &et.OrganizationID, &et.Slug, &et.Title,
		&et.Description, &et.DurationMins, &et.Active, &et.Timezone, &weekdays,
		&et.AvailStartMinute, &et.AvailEndMinute, &et.CreatedAt, &et.UpdatedAt); err != nil {
		return model.BookingEventType{}, err
	}
	et.AvailWeekdays = make([]int, len(weekdays))
	for i, w := range weekdays {
		et.AvailWeekdays[i] = int(w)
	}
	return et, nil
}

func (s *Store) CreateBookingEventType(ctx context.Context, et model.BookingEventType) (model.BookingEventType, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO booking_event_types
			(host_user_id, organization_id, slug, title, description, duration_mins, active,
			 timezone, avail_weekdays, avail_start_minute, avail_end_minute)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at, updated_at`,
		et.HostUserID, et.OrganizationID, et.Slug, et.Title, et.Description, et.DurationMins,
		et.Active, et.Timezone, et.AvailWeekdays, et.AvailStartMinute, et.AvailEndMinute).
		Scan(&et.ID, &et.CreatedAt, &et.UpdatedAt)
	return et, err
}

func (s *Store) GetBookingEventType(ctx context.Context, id uuid.UUID) (model.BookingEventType, error) {
	return scanBookingRow(s.pool.QueryRow(ctx, `SELECT `+bookingCols+` FROM booking_event_types WHERE id = $1`, id))
}

func (s *Store) GetBookingEventTypeBySlug(ctx context.Context, slug string) (model.BookingEventType, error) {
	return scanBookingRow(s.pool.QueryRow(ctx, `SELECT `+bookingCols+` FROM booking_event_types WHERE slug = $1`, slug))
}

func (s *Store) ListBookingEventTypesForUser(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+bookingCols+` FROM booking_event_types WHERE host_user_id = $1 ORDER BY created_at DESC`, hostUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.BookingEventType{}
	for rows.Next() {
		et, err := scanBookingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, et)
	}
	return out, rows.Err()
}

func (s *Store) UpdateBookingEventType(ctx context.Context, et model.BookingEventType) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE booking_event_types SET
			title=$2, description=$3, duration_mins=$4, active=$5, timezone=$6,
			avail_weekdays=$7, avail_start_minute=$8, avail_end_minute=$9, updated_at=now()
		WHERE id=$1`,
		et.ID, et.Title, et.Description, et.DurationMins, et.Active, et.Timezone,
		et.AvailWeekdays, et.AvailStartMinute, et.AvailEndMinute)
	return err
}

func (s *Store) DeleteBookingEventType(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM booking_event_types WHERE id = $1`, id)
	return err
}
