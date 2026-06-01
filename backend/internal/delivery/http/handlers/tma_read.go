package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type tmaMeetingDTO struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Dept         string   `json:"dept"`
	Host         string   `json:"host"`
	Date         string   `json:"date"`  // YYYY-MM-DD, Almaty
	Start        string   `json:"start"` // HH:MM, Almaty
	End          string   `json:"end"`   // HH:MM, Almaty
	Rec          string   `json:"rec"`
	Organizer    string   `json:"organizer"`    // email
	Participants []string `json:"participants"` // emails
	Desc         string   `json:"desc"`
	MeetLink     string   `json:"meet_link"`
	Status       string   `json:"status"`
}

type tmaEmployeeDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Dept  string `json:"dept"`
	Tg    bool   `json:"tg"`
}

type tmaFreeSlotDTO struct {
	ISO   string `json:"iso"`   // YYYY-MM-DD, Almaty
	Start string `json:"start"` // HH:MM, Almaty
	End   string `json:"end"`   // HH:MM, Almaty
	Mins  int    `json:"mins"`
}

// splitMeetingTime renders a meeting's UTC start/end into Almaty-local date + times.
func splitMeetingTime(startsAt, endsAt time.Time, loc *time.Location) (date, start, end string) {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	return s.Format("2006-01-02"), s.Format("15:04"), e.Format("15:04")
}

// tmaScopeWindow maps a scope to a [from,to) window around now (ListScheduleForEmail
// filters starts_at in [from,to)). Unknown scope → ok=false.
func tmaScopeWindow(scope string, now time.Time) (from, to time.Time, ok bool) {
	const horizon = 365
	switch scope {
	case "upcoming":
		return now, now.AddDate(0, 0, horizon), true
	case "past":
		return now.AddDate(0, 0, -horizon), now, true
	case "all":
		return now.AddDate(0, 0, -horizon), now.AddDate(0, 0, horizon), true
	default:
		return time.Time{}, time.Time{}, false
	}
}

// toMeetingDTO maps a meeting to the UI-shaped DTO, resolving organizer email and
// participant emails (N+1 per meeting; fine for personal-scale lists).
func (a *API) toMeetingDTO(ctx context.Context, m postgres.Meeting) tmaMeetingDTO {
	loc := almatyLoc()
	date, start, end := splitMeetingTime(m.StartsAt, m.EndsAt, loc)
	organizer := ""
	if m.OrganizerUserID != nil {
		if u, err := a.App.Store.GetUserByID(ctx, *m.OrganizerUserID); err == nil {
			organizer = u.Email
		}
	}
	emails := []string{}
	if parts, err := a.App.Store.ListParticipants(ctx, m.ID); err == nil {
		for _, p := range parts {
			if p.Email != "" {
				emails = append(emails, p.Email)
			}
		}
	}
	return tmaMeetingDTO{
		ID: m.ID.String(), Type: m.Type, Dept: m.Dept, Host: m.Host,
		Date: date, Start: start, End: end, Rec: m.Recurrence,
		Organizer: organizer, Participants: emails, Desc: m.Description,
		MeetLink: m.MeetLink, Status: m.Status,
	}
}

func (a *API) toMeetingDTOs(ctx context.Context, ms []postgres.Meeting) []tmaMeetingDTO {
	out := make([]tmaMeetingDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, a.toMeetingDTO(ctx, m))
	}
	return out
}

// botUserEmail returns the authed TMA user's email, or "" if absent.
func botUserEmail(c *fiber.Ctx) (string, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok {
		return "", false
	}
	return bu.Email, true
}
