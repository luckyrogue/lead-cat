package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/application/query"
)

type miniappMeetingDTO struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`
	Dept            string   `json:"dept"`
	Host            string   `json:"host"`
	Date            string   `json:"date"`
	Start           string   `json:"start"`
	End             string   `json:"end"`
	Rec             string   `json:"rec"`
	SeriesID        string   `json:"series_id"`
	RecurrenceUntil string   `json:"recurrence_until"`
	Organizer       string   `json:"organizer"`
	Participants    []string `json:"participants"`
	Desc            string   `json:"desc"`
	MeetLink        string   `json:"meet_link"`
	Status          string   `json:"status"`
}

type miniappEmployeeDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Dept  string `json:"dept"`
	Tg    bool   `json:"tg"`
}

type miniappFreeSlotDTO struct {
	ISO   string `json:"iso"`
	Start string `json:"start"`
	End   string `json:"end"`
	Mins  int    `json:"mins"`
}

func splitMeetingTime(startsAt, endsAt time.Time, loc *time.Location) (date, start, end string) {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	return s.Format("2006-01-02"), s.Format("15:04"), e.Format("15:04")
}

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
		SeriesID: d.SeriesID, RecurrenceUntil: d.RecurrenceUntil,
		Organizer: d.Organizer, Participants: d.Participants, Desc: d.Desc,
		MeetLink: d.MeetLink, Status: d.Status,
	}
}

func (a *API) toMeetingDTO(ctx context.Context, m model.Meeting) miniappMeetingDTO {
	return miniappMeetingFromQuery(a.App.MiniAppMeetingDTO(ctx, m, almatyLoc()))
}

func (a *API) toMeetingDTOs(ctx context.Context, ms []model.Meeting) []miniappMeetingDTO {
	out := make([]miniappMeetingDTO, 0, len(ms))
	for _, m := range ms {
		out = append(out, a.toMeetingDTO(ctx, m))
	}
	return out
}

func botUserEmail(c *fiber.Ctx) (string, bool) {
	bu, ok := botUser(c)
	if !ok {
		return "", false
	}
	return bu.Email, true
}

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
	From         string   `json:"from"`
	To           string   `json:"to"`
	DurationMins int      `json:"duration_mins"`
}

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
