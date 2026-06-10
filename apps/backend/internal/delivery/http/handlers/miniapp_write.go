package handlers

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

func parseScope(c *fiber.Ctx) (string, error) {
	switch s := c.Query("scope", "this"); s {
	case "this", "whole":
		return s, nil
	default:
		return "", fmt.Errorf("invalid scope %q", s)
	}
}

func mapToSeriesUpdateInput(req miniappUpdateRequest) application.SeriesUpdateInput {
	return application.SeriesUpdateInput{
		Dept:        req.Dept,
		Type:        req.Type,
		Host:        req.Host,
		Description: req.Desc,
		Start:       req.Start,
		End:         req.End,
	}
}

type miniappCreateRequest struct {
	Dept            string   `json:"dept"`
	Type            string   `json:"type"`
	Host            string   `json:"host"`
	Date            string   `json:"date"`
	Start           string   `json:"start"`
	End             string   `json:"end"`
	Recurrence      string   `json:"recurrence"`
	Desc            string   `json:"desc"`
	Participants    []string `json:"participants"`
	RecurrenceUntil *string  `json:"recurrence_until,omitempty"`
	RecurrenceDays  *[]int   `json:"recurrence_days,omitempty"`
}

func toCreateMeetingInput(req miniappCreateRequest, hostFallback string) application.CreateMeetingInput {
	host := strings.TrimSpace(req.Host)
	if host == "" {
		host = hostFallback
	}
	parts := make([]model.MeetingParticipant, 0, len(req.Participants))
	for _, e := range req.Participants {
		if e = strings.TrimSpace(e); e != "" {
			parts = append(parts, model.MeetingParticipant{Email: e})
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

func botUser(c *fiber.Ctx) (model.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(model.BotUser)
	return bu, ok
}

func (a *API) MiniAppCreateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req miniappCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	wsIDs, err := a.App.ListOrganizationsWithGoogle(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if len(wsIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "meetings_not_configured")
	}

	organizationID := wsIDs[0]
	organizerID, err := a.App.EnsureMiniAppOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	m, err := a.App.CreateMeeting(c.Context(), organizationID, organizerID, toCreateMeetingInput(req, bu.FullName))
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) || errors.Is(err, application.ErrGoogleNotConfigured) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.App.Log.Info("miniapp_meeting_created",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", m.ID.String()),
		zap.String("organization_id", organizationID.String()))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
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
	if _, ok := botUser(c); !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	var req miniappConflictRequest
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
	ocs, err := a.App.MeetingSeriesConflicts(c.Context(), req.Participants, start, end, meeting.Recurrence(rec), days, until)
	if err != nil {

		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	occurrences := make([]miniappOccurrenceConflictsDTO, 0, len(ocs))
	for _, oc := range ocs {
		occurrences = append(occurrences, toOccurrenceConflicts(oc, loc))
	}
	return c.JSON(fiber.Map{"occurrences": occurrences})
}

func (a *API) editableOrganization(c *fiber.Ctx, telegramID int64, meetingID uuid.UUID) (uuid.UUID, bool, error) {
	ms, err := a.App.ListEditableMeetings(c.Context(), telegramID)
	if err != nil {
		return uuid.Nil, false, err
	}
	for _, m := range ms {
		if m.ID == meetingID {
			return m.OrganizationID, true, nil
		}
	}
	return uuid.Nil, false, nil
}

type miniappUpdateRequest struct {
	Dept  *string `json:"dept"`
	Type  *string `json:"type"`
	Host  *string `json:"host"`
	Date  *string `json:"date"`
	Start *string `json:"start"`
	End   *string `json:"end"`
	Desc  *string `json:"desc"`
}

func (a *API) MiniAppUpdateMeeting(c *fiber.Ctx) error {
	bu, ok := botUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	var req miniappUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	organizationID, found, err := a.editableOrganization(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureMiniAppOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if scope == "this" {
		m, err := a.App.UpdateMeeting(c.Context(), organizationID, organizerID, meetingID, application.UpdateMeetingInput{
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
		a.App.Log.Info("miniapp_meeting_updated",
			zap.Int64("telegram_id", bu.TelegramID),
			zap.String("meeting_id", meetingID.String()),
			zap.String("organization_id", organizationID.String()))
		return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
	}

	if req.Date != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date cannot be changed for whole-series edit")
	}
	n, err := a.App.UpdateWholeSeries(c.Context(), organizationID, organizerID, meetingID, mapToSeriesUpdateInput(req))
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
	a.App.Log.Info("miniapp_meeting_whole_series_updated",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("organization_id", organizationID.String()),
		zap.Int("count", n))

	m, err := a.App.GetMeeting(c.Context(), organizationID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m)})
}

func (a *API) MiniAppDeleteMeeting(c *fiber.Ctx) error {
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
	organizationID, found, err := a.editableOrganization(c, bu.TelegramID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if !found {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	organizerID, err := a.App.EnsureMiniAppOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	if scope == "this" {
		if err := a.App.CancelMeeting(c.Context(), organizationID, organizerID, meetingID); err != nil {
			if errors.Is(err, application.ErrForbidden) {
				return fiber.NewError(fiber.StatusForbidden, "forbidden")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "internal")
		}
		a.App.Log.Info("miniapp_meeting_cancelled",
			zap.Int64("telegram_id", bu.TelegramID),
			zap.String("meeting_id", meetingID.String()),
			zap.String("organization_id", organizationID.String()))
		return c.SendStatus(fiber.StatusNoContent)
	}

	n, err := a.App.CancelWholeSeries(c.Context(), organizationID, organizerID, meetingID)
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
	a.App.Log.Info("miniapp_meeting_cancelled_whole_series",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("organization_id", organizationID.String()),
		zap.Int("count", n))
	return c.SendStatus(fiber.StatusNoContent)
}
