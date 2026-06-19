package application

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type BookingRequest struct {
	Name  string
	Email string
	Start time.Time
}

type BookingConfirmation struct {
	MeetLink string    `json:"meet_link"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
}

func (s *Services) SubmitBooking(ctx context.Context, slug string, req BookingRequest) (BookingConfirmation, error) {
	et, err := s.Store.GetBookingEventTypeBySlug(ctx, slug)
	if err != nil {
		return BookingConfirmation{}, err
	}
	if !et.Active {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	if _, perr := mail.ParseAddress(req.Email); perr != nil {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	loc := loadLoc(et.Timezone)
	start := req.Start.In(loc)
	dur := time.Duration(et.DurationMins) * time.Minute
	end := start.Add(dur)
	if !start.After(time.Now()) {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	iso := int(start.Weekday())
	if iso == 0 {
		iso = 7
	}
	if !weekdaySet(et.AvailWeekdays)[iso] {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	minute := start.Hour()*60 + start.Minute()
	if minute < et.AvailStartMinute || minute+et.DurationMins > et.AvailEndMinute {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	host, ok, err := s.Store.GetPlatformUserByID(ctx, et.HostUserID)
	if err != nil || !ok || host.Email == "" {
		return BookingConfirmation{}, model.ErrInvalidBooking
	}
	conflicts, err := s.MeetingConflicts(ctx, host.Email, []string{host.Email}, start.UTC(), end.UTC(), uuid.Nil)
	if err != nil {
		return BookingConfirmation{}, err
	}
	if len(conflicts) > 0 {
		return BookingConfirmation{}, model.ErrSlotTaken
	}
	m, err := s.CreateMeeting(ctx, et.OrganizationID, et.HostUserID, CreateMeetingInput{
		Dept:         et.Title,
		Type:         "Booking",
		Host:         name,
		Title:        et.Title + " — " + name,
		Description:  "Booked via " + et.Title + " by " + name + " <" + req.Email + ">",
		Date:         start.Format("2006-01-02"),
		Start:        start.Format("15:04"),
		End:          end.Format("15:04"),
		Timezone:     et.Timezone,
		Recurrence:   "once",
		Participants: []model.MeetingParticipant{{Email: req.Email}},
	})
	if err != nil {
		return BookingConfirmation{}, err
	}
	return BookingConfirmation{MeetLink: m.MeetLink, Start: start.UTC(), End: end.UTC()}, nil
}
