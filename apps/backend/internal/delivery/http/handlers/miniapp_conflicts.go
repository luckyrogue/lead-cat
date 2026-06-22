package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

func mapAvailabilityError(err error) *fiber.Error {
	if errors.Is(err, application.ErrUnknownParticipant) {
		return fiber.NewError(fiber.StatusBadRequest, "unknown_participant")
	}
	return fiber.NewError(fiber.StatusInternalServerError, "internal")
}

type miniappConflictDTO struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Start string `json:"start"`
	End   string `json:"end"`
}

func toConflictDTO(c application.Conflict, loc *time.Location) miniappConflictDTO {
	return miniappConflictDTO{
		Email: c.Email,
		Name:  c.PersonName,
		Title: c.MeetingName,
		Start: c.Start.In(loc).Format("15:04"),
		End:   c.End.In(loc).Format("15:04"),
	}
}

type miniappOccurrenceConflictsDTO struct {
	Date      string               `json:"date"`
	Start     string               `json:"start"`
	End       string               `json:"end"`
	Conflicts []miniappConflictDTO `json:"conflicts"`
}

func toOccurrenceConflicts(oc application.OccurrenceConflicts, loc *time.Location) miniappOccurrenceConflictsDTO {
	startLocal := oc.Span.Start.In(loc)
	endLocal := oc.Span.End.In(loc)
	cs := make([]miniappConflictDTO, 0, len(oc.Conflicts))
	for _, c := range oc.Conflicts {
		cs = append(cs, toConflictDTO(c, loc))
	}
	return miniappOccurrenceConflictsDTO{
		Date:      startLocal.Format("2006-01-02"),
		Start:     startLocal.Format("15:04"),
		End:       endLocal.Format("15:04"),
		Conflicts: cs,
	}
}

type miniappConflictRequest struct {
	Participants    []string `json:"participants"`
	Date            string   `json:"date"`
	Start           string   `json:"start"`
	End             string   `json:"end"`
	ExcludeID       string   `json:"exclude_id"`
	Recurrence      *string  `json:"recurrence,omitempty"`
	RecurrenceUntil *string  `json:"recurrence_until,omitempty"`
	RecurrenceDays  *[]int   `json:"recurrence_days,omitempty"`
}

func (a *API) MiniAppConflicts(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req miniappConflictRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := resolveLoc(bu.Timezone)
	start, err1 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.Start, loc)
	end, err2 := time.ParseInLocation("2006-01-02 15:04", req.Date+" "+req.End, loc)
	if err1 != nil || err2 != nil || !end.After(start) || len(req.Participants) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid range/participants")
	}
	rec := ""
	if req.Recurrence != nil {
		rec = *req.Recurrence
	}
	if rec == "" || rec == string(meeting.Once) {
		exclude := uuid.Nil
		if s := strings.TrimSpace(req.ExcludeID); s != "" {
			if id, perr := uuid.Parse(s); perr == nil {
				exclude = id
			}
		}
		conflicts, err := a.App.MeetingConflicts(c.Context(), bu.Email, req.Participants, start, end, exclude)
		if err != nil {
			return mapAvailabilityError(err)
		}
		out := make([]miniappConflictDTO, 0, len(conflicts))
		for _, cf := range conflicts {
			out = append(out, toConflictDTO(cf, loc))
		}
		return c.JSON(fiber.Map{"occurrences": []miniappOccurrenceConflictsDTO{{
			Date:      req.Date,
			Start:     req.Start,
			End:       req.End,
			Conflicts: out,
		}}})
	}

	var until time.Time
	if req.RecurrenceUntil != nil && strings.TrimSpace(*req.RecurrenceUntil) != "" {
		u, uerr := time.ParseInLocation("2006-01-02", strings.TrimSpace(*req.RecurrenceUntil), loc)
		if uerr != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid recurrence_until")
		}
		until = u
	}
	var days []int
	if req.RecurrenceDays != nil {
		days = *req.RecurrenceDays
	}
	ocs, err := a.App.MeetingSeriesConflicts(c.Context(), bu.Email, req.Participants, start, end, meeting.Recurrence(rec), days, until)
	if err != nil {
		if fe := mapAvailabilityError(err); fe.Code == fiber.StatusBadRequest {
			return fe
		}
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	occurrences := make([]miniappOccurrenceConflictsDTO, 0, len(ocs))
	for _, oc := range ocs {
		occurrences = append(occurrences, toOccurrenceConflicts(oc, loc))
	}
	return c.JSON(fiber.Map{"occurrences": occurrences})
}
