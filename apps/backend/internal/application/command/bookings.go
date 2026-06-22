package command

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
	"github.com/luckyrogue/lead-cat/internal/platform/slug"
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
	SurveyID         *uuid.UUID
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

type bookingStore interface {
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	GetPlatformUserByID(ctx context.Context, id uuid.UUID) (model.PlatformUser, bool, error)
	CreateBookingEventType(ctx context.Context, et model.BookingEventType) (model.BookingEventType, error)
	GetBookingEventType(ctx context.Context, id uuid.UUID) (model.BookingEventType, error)
	UpdateBookingEventType(ctx context.Context, et model.BookingEventType) error
	DeleteBookingEventType(ctx context.Context, id uuid.UUID) error
}

type Bookings struct {
	Store bookingStore
}

func (c *Bookings) CreateEventType(ctx context.Context, hostUserID, orgID uuid.UUID, in EventTypeInput) (model.BookingEventType, error) {
	if err := validateEventType(in); err != nil {
		return model.BookingEventType{}, err
	}
	if _, ok, err := c.Store.GetOrgMember(ctx, orgID, hostUserID); err != nil {
		return model.BookingEventType{}, err
	} else if !ok {
		return model.BookingEventType{}, model.ErrForbidden
	}
	zone := strings.TrimSpace(in.Timezone)
	if zone == "" {
		if u, ok, err := c.Store.GetPlatformUserByID(ctx, hostUserID); err == nil && ok && u.Timezone != "" {
			zone = u.Timezone
		} else {
			zone = "Asia/Almaty"
		}
	}
	base := slug.Make(in.Title)
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
		Timezone:         zone,
		AvailWeekdays:    in.AvailWeekdays,
		AvailStartMinute: in.AvailStartMinute,
		AvailEndMinute:   in.AvailEndMinute,
	}
	return c.Store.CreateBookingEventType(ctx, et)
}

func (c *Bookings) UpdateEventType(ctx context.Context, hostUserID, id uuid.UUID, in EventTypeInput) error {
	if err := validateEventType(in); err != nil {
		return err
	}
	et, err := c.Store.GetBookingEventType(ctx, id)
	if err != nil {
		return err
	}
	if et.HostUserID != hostUserID {
		return model.ErrForbidden
	}
	zone := strings.TrimSpace(in.Timezone)
	if zone == "" {
		zone = et.Timezone
	}
	et.Title = in.Title
	et.Description = in.Description
	et.DurationMins = in.DurationMins
	et.Active = in.Active
	et.Timezone = zone
	et.AvailWeekdays = in.AvailWeekdays
	et.AvailStartMinute = in.AvailStartMinute
	et.AvailEndMinute = in.AvailEndMinute
	et.SurveyID = in.SurveyID
	return c.Store.UpdateBookingEventType(ctx, et)
}

func (c *Bookings) DeleteEventType(ctx context.Context, hostUserID, id uuid.UUID) error {
	et, err := c.Store.GetBookingEventType(ctx, id)
	if err != nil {
		return err
	}
	if et.HostUserID != hostUserID {
		return model.ErrForbidden
	}
	return c.Store.DeleteBookingEventType(ctx, id)
}
