package application

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/emailtemplates"
)

type BookingRequest struct {
	Name     string
	Email    string
	Start    time.Time
	Language string
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

	s.sendBookingEmails(ctx, et, host, req, name, start, end, m.MeetLink)

	return BookingConfirmation{MeetLink: m.MeetLink, Start: start.UTC(), End: end.UTC()}, nil
}

// sendBookingEmails sends the booker confirmation and the host notification.
// Best-effort: a nil mailer is a no-op; render/send errors are logged and swallowed
// so a created booking never becomes an error response.
func (s *Services) sendBookingEmails(
	ctx context.Context,
	et model.BookingEventType,
	host model.PlatformUser,
	req BookingRequest,
	bookerName string,
	start, end time.Time,
	meetLink string,
) {
	if s.email == nil {
		return
	}

	// Booker email — times in the event-type timezone, page/browser language.
	bookerDate := start.Format("Mon, 02 Jan 2006")
	bookerTime := start.Format("15:04") + " – " + end.Format("15:04")
	if subject, text, htmlBody, rerr := emailtemplates.RenderBookingConfirmation(emailtemplates.BookingConfirmationData{
		Language:   req.Language,
		BookerName: bookerName,
		EventTitle: et.Title,
		Date:       bookerDate,
		Time:       bookerTime,
		Tz:         et.Timezone,
		MeetLink:   meetLink,
	}); rerr != nil {
		s.Log.Warn("booking_confirmation_render_failed", zap.Error(rerr))
	} else if serr := s.email.SendMultipart(ctx, req.Email, subject, text, htmlBody, ""); serr != nil {
		s.Log.Warn("booking_confirmation_send_failed", zap.Error(serr))
	}

	// Host email — times in the host timezone (fallback event-type tz), host language.
	hostTz := host.Timezone
	if hostTz == "" {
		hostTz = et.Timezone
	}
	hostLoc := loadLoc(hostTz)
	hStart := start.In(hostLoc)
	hEnd := end.In(hostLoc)
	if subject, text, htmlBody, rerr := emailtemplates.RenderBookingHostNotification(emailtemplates.BookingHostNotificationData{
		Language:    host.Language,
		EventTitle:  et.Title,
		BookerName:  bookerName,
		BookerEmail: req.Email,
		Date:        hStart.Format("Mon, 02 Jan 2006"),
		Time:        hStart.Format("15:04") + " – " + hEnd.Format("15:04"),
		Tz:          hostTz,
		MeetLink:    meetLink,
	}); rerr != nil {
		s.Log.Warn("booking_host_notification_render_failed", zap.Error(rerr))
	} else if serr := s.email.SendMultipart(ctx, host.Email, subject, text, htmlBody, ""); serr != nil {
		s.Log.Warn("booking_host_notification_send_failed", zap.Error(serr))
	}
}
