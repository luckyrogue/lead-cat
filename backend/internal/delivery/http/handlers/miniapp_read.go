package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/query"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type miniappMeetingDTO struct {
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

type miniappEmployeeDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Dept  string `json:"dept"`
	Tg    bool   `json:"tg"`
}

type miniappFreeSlotDTO struct {
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

// miniappScopeWindow maps a scope to a [from,to) window around now (ListScheduleForEmail
// filters starts_at in [from,to)). Unknown scope → ok=false.
func miniappScopeWindow(scope string, now time.Time) (from, to time.Time, ok bool) {
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

func miniappMeetingFromQuery(d query.MiniAppMeeting) miniappMeetingDTO {
	return miniappMeetingDTO{
		ID: d.ID, Type: d.Type, Dept: d.Dept, Host: d.Host,
		Date: d.Date, Start: d.Start, End: d.End, Rec: d.Rec,
		Organizer: d.Organizer, Participants: d.Participants, Desc: d.Desc,
		MeetLink: d.MeetLink, Status: d.Status,
	}
}

// toMeetingDTO maps a meeting to the UI-shaped DTO, resolving organizer email and
// participant emails (N+1 per meeting; fine for personal-scale lists).
func (a *API) toMeetingDTO(ctx context.Context, m postgres.Meeting) miniappMeetingDTO {
	return miniappMeetingFromQuery(a.App.MiniAppMeetingDTO(ctx, m, almatyLoc()))
}

func (a *API) toMeetingDTOs(ctx context.Context, ms []postgres.Meeting) []miniappMeetingDTO {
	out := make([]miniappMeetingDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, a.toMeetingDTO(ctx, m))
	}
	return out
}

// botUserEmail returns the authed TMA user's email, or "" if absent.
func botUserEmail(c *fiber.Ctx) (string, bool) {
	bu, ok := botUser(c)
	if !ok {
		return "", false
	}
	return bu.Email, true
}

// MiniAppMyMeetings lists the authed user's meetings for a scope window.
func (a *API) MiniAppMyMeetings(c *fiber.Ctx) error {
	email, ok := botUserEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	from, to, ok := miniappScopeWindow(c.Query("scope"), time.Now())
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scope")
	}
	ms, err := a.App.EmployeeSchedule(c.Context(), email, from, to)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meetings": a.toMeetingDTOs(c.Context(), ms)})
}

// MiniAppSchedule lists a colleague's meetings (read-only directory feature, §4.6).
func (a *API) MiniAppSchedule(c *fiber.Ctx) error {
	if _, ok := botUserEmail(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email required")
	}
	from, to, ok := miniappScopeWindow(c.Query("scope"), time.Now())
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scope")
	}
	ms, err := a.App.EmployeeSchedule(c.Context(), email, from, to)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meetings": a.toMeetingDTOs(c.Context(), ms)})
}

// MiniAppEmployees searches the global directory (empty q → empty list).
func (a *API) MiniAppEmployees(c *fiber.Ctx) error {
	if _, ok := botUserEmail(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	q := strings.TrimSpace(c.Query("q"))
	out := []miniappEmployeeDTO{}
	if q != "" {
		emps, err := a.App.SearchEmployeesGlobal(c.Context(), q)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
		for _, e := range emps {
			out = append(out, miniappEmployeeDTO{ID: e.ID.String(), Name: e.FullName, Email: e.Email, Dept: e.Dept, Tg: e.HasTelegram})
		}
	}
	return c.JSON(fiber.Map{"employees": out})
}

type miniappFreeSlotsRequest struct {
	Participants []string `json:"participants"`
	From         string   `json:"from"` // YYYY-MM-DD (inclusive)
	To           string   `json:"to"`   // YYYY-MM-DD (inclusive)
	DurationMins int      `json:"duration_mins"`
}

// MiniAppFreeSlots finds common free time across participants (§4.8).
func (a *API) MiniAppFreeSlots(c *fiber.Ctx) error {
	if _, ok := botUserEmail(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req miniappFreeSlotsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	from, err1 := time.ParseInLocation("2006-01-02", req.From, loc)
	toIncl, err2 := time.ParseInLocation("2006-01-02", req.To, loc)
	if err1 != nil || err2 != nil || toIncl.Before(from) || req.DurationMins <= 0 || len(req.Participants) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/participants/duration")
	}
	slots, err := a.App.FreeSlots(c.Context(), req.Participants, from, toIncl.AddDate(0, 0, 1), req.DurationMins)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	out := make([]miniappFreeSlotDTO, 0, len(slots))
	for _, sl := range slots {
		out = append(out, miniappFreeSlotDTO{
			ISO:   sl.Day.In(loc).Format("2006-01-02"),
			Start: sl.Start.In(loc).Format("15:04"),
			End:   sl.End.In(loc).Format("15:04"),
			Mins:  sl.Mins,
		})
	}
	return c.JSON(fiber.Map{"slots": out})
}
