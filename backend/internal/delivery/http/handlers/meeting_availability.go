package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func almatyLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}

type conflictsRequest struct {
	Date             string   `json:"date"`  // YYYY-MM-DD
	Start            string   `json:"start"` // HH:MM
	End              string   `json:"end"`   // HH:MM
	Participants     []string `json:"participants"`
	ExcludeMeetingID string   `json:"exclude_meeting_id"`
}

type conflictItem struct {
	Email       string    `json:"email"`
	PersonName  string    `json:"person_name"`
	MeetingName string    `json:"meeting_name"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

// MeetingConflicts is the advisory §4.7 overlap check (read-only). Workspace authz
// is enforced by RequireWorkspaceAccess on the route.
func (a *API) MeetingConflicts(c *fiber.Ctx) error {
	var req conflictsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	start, err1 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
	if err1 != nil || err2 != nil || !end.After(start) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid date/time")
	}
	exclude := uuid.Nil
	if req.ExcludeMeetingID != "" {
		exclude, _ = uuid.Parse(req.ExcludeMeetingID)
	}
	conflicts, err := a.App.MeetingConflicts(c.Context(), req.Participants, start.UTC(), end.UTC(), exclude)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "conflict check failed")
	}
	out := make([]conflictItem, 0, len(conflicts))
	for _, cf := range conflicts {
		out = append(out, conflictItem{cf.Email, cf.PersonName, cf.MeetingName, cf.Start, cf.End})
	}
	return c.JSON(fiber.Map{"conflicts": out})
}

type freeSlotsRequest struct {
	From         string   `json:"from"` // YYYY-MM-DD (inclusive)
	To           string   `json:"to"`   // YYYY-MM-DD (inclusive)
	Participants []string `json:"participants"`
	DurationMins int      `json:"duration_mins"`
}

type freeSlotItem struct {
	Day   time.Time `json:"day"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Mins  int       `json:"mins"`
}

// FreeSlots is the §4.8 common-free-time finder (read-only).
func (a *API) FreeSlots(c *fiber.Ctx) error {
	var req freeSlotsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
	from, err1 := time.ParseInLocation("2006-01-02", req.From, loc)
	toIncl, err2 := time.ParseInLocation("2006-01-02", req.To, loc)
	if err1 != nil || err2 != nil || toIncl.Before(from) || req.DurationMins <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/duration")
	}
	slots, err := a.App.FreeSlots(c.Context(), req.Participants, from, toIncl.AddDate(0, 0, 1), req.DurationMins)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "free-slot search failed")
	}
	out := make([]freeSlotItem, 0, len(slots))
	for _, sl := range slots {
		out = append(out, freeSlotItem{sl.Day, sl.Start, sl.End, sl.Mins})
	}
	return c.JSON(fiber.Map{"slots": out})
}
