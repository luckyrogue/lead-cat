package application

import (
	"context"
	"database/sql"
	"time"
)

type BookingEventView struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	DurationMins int    `json:"duration_mins"`
	OrgName      string `json:"org_name"`
	Timezone     string `json:"timezone"`
}

type BookingView struct {
	Event BookingEventView `json:"event"`
	Slots []BookingSlot    `json:"slots"`
}

func (s *Services) PublicBooking(ctx context.Context, slug string, now time.Time) (BookingView, error) {
	et, err := s.Store.GetBookingEventTypeBySlug(ctx, slug)
	if err != nil {
		return BookingView{}, err
	}
	if !et.Active {
		return BookingView{}, sql.ErrNoRows
	}
	slots, err := s.BookingAvailability(ctx, et, now, now.AddDate(0, 0, 14))
	if err != nil {
		return BookingView{}, err
	}
	orgName := ""
	if org, oerr := s.Store.GetOrganization(ctx, et.OrganizationID); oerr == nil {
		orgName = org.Name
	}
	return BookingView{
		Event: BookingEventView{Title: et.Title, Description: et.Description, DurationMins: et.DurationMins, OrgName: orgName, Timezone: et.Timezone},
		Slots: slots,
	}, nil
}
