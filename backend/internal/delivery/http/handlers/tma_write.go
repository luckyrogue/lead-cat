package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// parseScope reads ?scope=this|whole; default "this". Returns error for any other value.
func parseScope(c *fiber.Ctx) (string, error) {
	switch s := c.Query("scope", "this"); s {
	case "this", "whole":
		return s, nil
	default:
		return "", fmt.Errorf("invalid scope %q", s)
	}
}

// mapToSeriesUpdateInput converts the wire request into a SeriesUpdateInput.
// Date is intentionally NOT carried — whole-series edits don't change the date
// of each occurrence (that's locked when the series is created).
func mapToSeriesUpdateInput(req tmaUpdateRequest) application.SeriesUpdateInput {
	return application.SeriesUpdateInput{
		Dept:        req.Dept,
		Type:        req.Type,
		Host:        req.Host,
		Description: req.Desc,
		Start:       req.Start,
		End:         req.End,
	}
}

// tmaCreateRequest is the create-meeting payload. Recurrence fields are optional:
// the domain Validate() enforces the per-recurrence rules (until required for
// non-once, days required for custom, etc.).
type tmaCreateRequest struct {
	Dept            string   `json:"dept"`
	Type            string   `json:"type"`
	Host            string   `json:"host"`
	Date            string   `json:"date"`  // YYYY-MM-DD
	Start           string   `json:"start"` // HH:MM
	End             string   `json:"end"`   // HH:MM
	Recurrence      string   `json:"recurrence"`
	Desc            string   `json:"desc"`
	Participants    []string `json:"participants"`               // emails
	RecurrenceUntil *string  `json:"recurrence_until,omitempty"` // YYYY-MM-DD; required when recurrence != once
	RecurrenceDays  *[]int   `json:"recurrence_days,omitempty"`  // 1..7 (Mon..Sun); required when recurrence == custom
}

// toCreateMeetingInput maps the TMA request to the application input. Pure: host
// falls back to the bot user's name when blank; blank participant emails are dropped.
func toCreateMeetingInput(req tmaCreateRequest, hostFallback string) application.CreateMeetingInput {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = hostFallback
	}
	parts := make([]postgres.MeetingParticipant, 0, len(req.Participants))
	for _, e := range req.Participants {
		if e = strings.TrimSpace(e); e != "" {
			parts = append(parts, postgres.MeetingParticipant{Email: e})
		}
	}
	in := application.CreateMeetingInput{
		Dept: req.Dept, Type: req.Type, Host: host,
		Date: req.Date, Start: req.Start, End: req.End,
		Recurrence: req.Recurrence, Description: req.Desc,
		Participants: parts,
	}
	if req.RecurrenceUntil != nil {
		in.RecurrenceUntil = *req.RecurrenceUntil
	}
	if req.RecurrenceDays != nil {
		in.RecurrenceDays = *req.RecurrenceDays
	}
	return in
}

// botUser returns the authed TMA bot_user from locals.
func botUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok
}

// TMACreateMeeting creates a meeting (once or recurring series) for the authed TMA user.
// Recurrence rules are enforced by the domain Validate() inside CreateMeeting and
// surface as 400 validation_failed via ErrInvalidInput.
func (a *API) TMACreateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	wsIDs, err := a.App.Store.ListWorkspacesWithGoogle(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if len(wsIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "meetings_not_configured")
	}
	// Use the first Google-configured workspace; multi-workspace targeting is deferred.
	workspaceID := wsIDs[0]
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	m, err := a.App.CreateMeeting(c.Context(), workspaceID, organizerID, toCreateMeetingInput(req, bu.FullName))
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) || errors.Is(err, application.ErrGoogleNotConfigured) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.App.Log.Info("tma_meeting_created",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", m.ID.String()),
		zap.String("workspace_id", workspaceID.String()))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}

type tmaConflictDTO struct {
	Email string `json:"email"`
	Name  string `json:"name"`  // application.Conflict.PersonName
	Title string `json:"title"` // application.Conflict.MeetingName
	Start string `json:"start"` // HH:MM Almaty
	End   string `json:"end"`
}

// toConflictDTO renders a conflict's UTC times into Almaty HH:MM. Pure.
func toConflictDTO(c application.Conflict, loc *time.Location) tmaConflictDTO {
	return tmaConflictDTO{
		Email: c.Email,
		Name:  c.PersonName,
		Title: c.MeetingName,
		Start: c.Start.In(loc).Format("15:04"),
		End:   c.End.In(loc).Format("15:04"),
	}
}

// tmaOccurrenceConflictsDTO is one occurrence's date/time + its conflicts list,
// rendered in Almaty TZ. Always wrapped in {"occurrences":[...]} on the wire.
type tmaOccurrenceConflictsDTO struct {
	Date      string           `json:"date"`  // YYYY-MM-DD
	Start     string           `json:"start"` // HH:MM Almaty
	End       string           `json:"end"`   // HH:MM Almaty
	Conflicts []tmaConflictDTO `json:"conflicts"`
}

// toOccurrenceConflicts maps an application OccurrenceConflicts + locale into
// the wire DTO. Pure.
func toOccurrenceConflicts(oc application.OccurrenceConflicts, loc *time.Location) tmaOccurrenceConflictsDTO {
	startLocal := oc.Span.Start.In(loc)
	endLocal := oc.Span.End.In(loc)
	cs := make([]tmaConflictDTO, 0, len(oc.Conflicts))
	for _, c := range oc.Conflicts {
		cs = append(cs, toConflictDTO(c, loc))
	}
	return tmaOccurrenceConflictsDTO{
		Date:      startLocal.Format("2006-01-02"),
		Start:     startLocal.Format("15:04"),
		End:       endLocal.Format("15:04"),
		Conflicts: cs,
	}
}

type tmaConflictRequest struct {
	Participants    []string `json:"participants"`
	Date            string   `json:"date"`  // YYYY-MM-DD
	Start           string   `json:"start"` // HH:MM
	End             string   `json:"end"`   // HH:MM
	ExcludeID       string   `json:"exclude_id"`
	Recurrence      *string  `json:"recurrence,omitempty"`
	RecurrenceUntil *string  `json:"recurrence_until,omitempty"` // YYYY-MM-DD
	RecurrenceDays  *[]int   `json:"recurrence_days,omitempty"`  // 1..7 (Mon..Sun)
}

// TMAConflicts reports cross-participant conflicts for a pending meeting (§4.7).
// Response is always occurrence-grouped: {"occurrences":[{date,start,end,conflicts:[]}]}.
// For once (or absent recurrence) it's a one-element array with the single check;
// for daily/weekly/custom/monthly the series is expanded and only occurrences with
// ≥1 conflict are returned.
func (a *API) TMAConflicts(c *fiber.Ctx) error {
	if _, ok := botUser(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req tmaConflictRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	loc := almatyLoc()
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
		conflicts, err := a.App.MeetingConflicts(c.Context(), req.Participants, start, end, exclude)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
		out := make([]tmaConflictDTO, 0, len(conflicts))
		for _, cf := range conflicts {
			out = append(out, toConflictDTO(cf, loc))
		}
		return c.JSON(fiber.Map{"occurrences": []tmaOccurrenceConflictsDTO{{
			Date:      req.Date,
			Start:     req.Start,
			End:       req.End,
			Conflicts: out,
		}}})
	}
	// Recurring series: expand and per-occurrence check.
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
	ocs, err := a.App.MeetingSeriesConflicts(c.Context(), req.Participants, start, end, meeting.Recurrence(rec), days, until)
	if err != nil {
		// Domain errors from meeting.Occurrences (ErrRecurrenceDays,
		// ErrRecurrenceWindow, ErrTooManyOccurrences, …) surface as 400.
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	occurrences := make([]tmaOccurrenceConflictsDTO, 0, len(ocs))
	for _, oc := range ocs {
		occurrences = append(occurrences, toOccurrenceConflicts(oc, loc))
	}
	return c.JSON(fiber.Map{"occurrences": occurrences})
}

// editableWorkspace returns the workspace of a meeting the TMA user may edit,
// or false if the meeting is not in their editable set (not theirs / not
// scheduled / past). Used by edit + delete.
func (a *API) editableWorkspace(c *fiber.Ctx, telegramID int64, meetingID uuid.UUID) (uuid.UUID, bool, error) {
	ms, err := a.App.ListEditableMeetings(c.Context(), telegramID)
	if err != nil {
		return uuid.Nil, false, err
	}
	for _, m := range ms {
		if m.ID == meetingID {
			return m.WorkspaceID, true, nil
		}
	}
	return uuid.Nil, false, nil
}

type tmaUpdateRequest struct {
	Dept  *string `json:"dept"`
	Type  *string `json:"type"`
	Host  *string `json:"host"`
	Date  *string `json:"date"`
	Start *string `json:"start"`
	End   *string `json:"end"`
	Desc  *string `json:"desc"`
}

// TMAUpdateMeeting edits a single meeting the authed TMA user organizes (§4.4).
func (a *API) TMAUpdateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	var req tmaUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	workspaceID, found, err := a.editableWorkspace(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if scope == "this" {
		m, err := a.App.UpdateMeeting(c.Context(), workspaceID, organizerID, meetingID, application.UpdateMeetingInput{
			Dept: req.Dept, Type: req.Type, Host: req.Host,
			Date: req.Date, Start: req.Start, End: req.End, Description: req.Desc,
		})
		if err != nil {
			switch {
			case errors.Is(err, application.ErrForbidden):
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			case errors.Is(err, application.ErrInvalidInput):
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			default:
				return fiber.NewError(fiber.StatusInternalServerError, "internal")
			}
		}
		a.App.Log.Info("tma_meeting_updated",
			zap.Int64("telegram_id", bu.TelegramID),
			zap.String("meeting_id", meetingID.String()),
			zap.String("workspace_id", workspaceID.String()))
		return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
	}
	// scope == "whole"
	if req.Date != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date cannot be changed for whole-series edit")
	}
	n, err := a.App.UpdateWholeSeries(c.Context(), workspaceID, organizerID, meetingID, mapToSeriesUpdateInput(req))
	if err != nil {
		switch {
		case errors.Is(err, application.ErrForbidden):
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		case errors.Is(err, application.ErrInvalidInput):
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
	}
	a.App.Log.Info("tma_meeting_whole_series_updated",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("count", n))
	// Refetch the picked occurrence so the response shape mirrors scope=this.
	m, err := a.App.GetMeeting(c.Context(), workspaceID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}

// TMADeleteMeeting cancels a single meeting the authed TMA user organizes (§4.5).
func (a *API) TMADeleteMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	workspaceID, found, err := a.editableWorkspace(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureTMAOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if scope == "this" {
		if err := a.App.CancelMeeting(c.Context(), workspaceID, organizerID, meetingID); err != nil {
			if errors.Is(err, application.ErrForbidden) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
		a.App.Log.Info("tma_meeting_cancelled",
			zap.Int64("telegram_id", bu.TelegramID),
			zap.String("meeting_id", meetingID.String()),
			zap.String("workspace_id", workspaceID.String()))
		return c.SendStatus(fiber.StatusNoContent)
	}
	// scope == "whole"
	n, err := a.App.CancelWholeSeries(c.Context(), workspaceID, organizerID, meetingID)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrForbidden):
			return fiber.NewError(fiber.StatusForbidden, "forbidden")
		case errors.Is(err, application.ErrInvalidInput):
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
	}
	a.App.Log.Info("tma_meeting_cancelled_whole_series",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("workspace_id", workspaceID.String()),
		zap.Int("count", n))
	return c.SendStatus(fiber.StatusNoContent)
}
