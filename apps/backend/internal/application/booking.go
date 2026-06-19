package application

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

type EventTypeInput struct {
	Title            string
	Description      string
	DurationMins     int
	Timezone         string
	AvailWeekdays    []int
	AvailStartMinute int
	AvailEndMinute   int
	Active           bool
}

var ErrInvalidEventType = errors.New("invalid event type")

func validateEventType(in EventTypeInput) error {
	if strings.TrimSpace(in.Title) == "" {
		return ErrInvalidEventType
	}
	if in.DurationMins < 5 || in.DurationMins > 480 {
		return ErrInvalidEventType
	}
	if len(in.AvailWeekdays) == 0 {
		return ErrInvalidEventType
	}
	for _, d := range in.AvailWeekdays {
		if d < 1 || d > 7 {
			return ErrInvalidEventType
		}
	}
	if in.AvailStartMinute < 0 || in.AvailEndMinute > 1440 || in.AvailStartMinute >= in.AvailEndMinute {
		return ErrInvalidEventType
	}
	return nil
}

func (s *Services) CreateEventType(ctx context.Context, hostUserID, orgID uuid.UUID, in EventTypeInput) (model.BookingEventType, error) {
	if err := validateEventType(in); err != nil {
		return model.BookingEventType{}, err
	}
	if _, ok, err := s.Store.GetOrgMember(ctx, orgID, hostUserID); err != nil {
		return model.BookingEventType{}, err
	} else if !ok {
		return model.BookingEventType{}, model.ErrForbidden
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		if u, ok, err := s.Store.GetPlatformUserByID(ctx, hostUserID); err == nil && ok && u.Timezone != "" {
			tz = u.Timezone
		} else {
			tz = "Asia/Almaty"
		}
	}
	base := slugify(in.Title)
	if base == "" {
		base = "event"
	}
	suffix, err := authweb.NewState(nil)
	if err != nil {
		return model.BookingEventType{}, err
	}
	et := model.BookingEventType{
		HostUserID:       hostUserID,
		OrganizationID:   orgID,
		Slug:             base + "-" + suffix[:6],
		Title:            in.Title,
		Description:      in.Description,
		DurationMins:     in.DurationMins,
		Active:           in.Active,
		Timezone:         tz,
		AvailWeekdays:    in.AvailWeekdays,
		AvailStartMinute: in.AvailStartMinute,
		AvailEndMinute:   in.AvailEndMinute,
	}
	return s.Store.CreateBookingEventType(ctx, et)
}

func (s *Services) ListMyEventTypes(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	return s.Store.ListBookingEventTypesForUser(ctx, hostUserID)
}

func (s *Services) UpdateEventType(ctx context.Context, hostUserID, id uuid.UUID, in EventTypeInput) error {
	if err := validateEventType(in); err != nil {
		return err
	}
	et, err := s.Store.GetBookingEventType(ctx, id)
	if err != nil {
		return err
	}
	if et.HostUserID != hostUserID {
		return model.ErrForbidden
	}
	tz := strings.TrimSpace(in.Timezone)
	if tz == "" {
		tz = et.Timezone
	}
	et.Title = in.Title
	et.Description = in.Description
	et.DurationMins = in.DurationMins
	et.Active = in.Active
	et.Timezone = tz
	et.AvailWeekdays = in.AvailWeekdays
	et.AvailStartMinute = in.AvailStartMinute
	et.AvailEndMinute = in.AvailEndMinute
	return s.Store.UpdateBookingEventType(ctx, et)
}

func (s *Services) DeleteEventType(ctx context.Context, hostUserID, id uuid.UUID) error {
	et, err := s.Store.GetBookingEventType(ctx, id)
	if err != nil {
		return err
	}
	if et.HostUserID != hostUserID {
		return model.ErrForbidden
	}
	return s.Store.DeleteBookingEventType(ctx, id)
}
