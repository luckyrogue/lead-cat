package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
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

func toCreateMeetingInput(req miniappCreateRequest, hostFallback string, userTimezone string) application.CreateMeetingInput {
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
		Timezone:     userTimezone,
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
	organizationID, err := a.App.ResolveMiniAppOrganization(c.Context())
	if err != nil {
		if errors.Is(err, application.ErrGoogleNotConfigured) {
			return fiber.NewError(fiber.StatusBadRequest, "meetings_not_configured")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	organizerID, err := a.App.EnsureMiniAppOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	m, err := a.App.CreateMeeting(c.Context(), organizationID, organizerID, toCreateMeetingInput(req, bu.FullName, bu.Timezone))
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
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m, resolveLoc(bu.Timezone))})
}

func (a *API) miniAppMeetingOrganization(c *fiber.Ctx, meetingID uuid.UUID) (uuid.UUID, error) {
	orgID, err := a.App.ResolveMiniAppOrganization(c.Context())
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := a.App.GetMeeting(c.Context(), orgID, meetingID); err != nil {
		return uuid.Nil, err
	}
	return orgID, nil
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
	bu, organizationID, organizerID, meetingID, err := a.resolveMiniAppMeetingOp(c)
	if err != nil {
		return err
	}
	var req miniappUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if scope == "this" {
		m, err := a.App.UpdateMeeting(c.Context(), organizationID, organizerID, meetingID, application.UpdateMeetingInput{
			Dept: req.Dept, Type: req.Type, Host: req.Host,
			Date: req.Date, Start: req.Start, End: req.End, Description: req.Desc,
			Timezone: bu.Timezone,
		})
		if err != nil {
			return mapMeetingWriteError(err)
		}
		a.App.Log.Info("miniapp_meeting_updated",
			zap.Int64("telegram_id", bu.TelegramID),
			zap.String("meeting_id", meetingID.String()),
			zap.String("organization_id", organizationID.String()))
		return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m, resolveLoc(bu.Timezone))})
	}

	if req.Date != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date cannot be changed for whole-series edit")
	}
	n, err := a.App.UpdateWholeSeries(c.Context(), organizationID, organizerID, meetingID, mapToSeriesUpdateInput(req))
	if err != nil {
		return mapMeetingWriteError(err)
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
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m, resolveLoc(bu.Timezone))})
}

func (a *API) MiniAppDeleteMeeting(c *fiber.Ctx) error {
	bu, organizationID, organizerID, meetingID, err := a.resolveMiniAppMeetingOp(c)
	if err != nil {
		return err
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if scope == "this" {
		if err := a.App.CancelMeeting(c.Context(), organizationID, organizerID, meetingID); err != nil {
			return mapMeetingWriteError(err)
		}
		a.App.Log.Info("miniapp_meeting_cancelled",
			zap.Int64("telegram_id", bu.TelegramID),
			zap.String("meeting_id", meetingID.String()),
			zap.String("organization_id", organizationID.String()))
		return c.SendStatus(fiber.StatusNoContent)
	}

	n, err := a.App.CancelWholeSeries(c.Context(), organizationID, organizerID, meetingID)
	if err != nil {
		return mapMeetingWriteError(err)
	}
	a.App.Log.Info("miniapp_meeting_cancelled_whole_series",
		zap.Int64("telegram_id", bu.TelegramID),
		zap.String("meeting_id", meetingID.String()),
		zap.String("organization_id", organizationID.String()),
		zap.Int("count", n))
	return c.SendStatus(fiber.StatusNoContent)
}

type miniappParticipantRequest struct {
	Email string `json:"email"`
}

func (a *API) resolveMiniAppMeetingOp(c *fiber.Ctx) (bu model.BotUser, orgID, organizerID, meetingID uuid.UUID, err error) {
	bu, ok := botUser(c)
	if !ok {
		return bu, uuid.Nil, uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	meetingID, perr := uuid.Parse(c.Params("id"))
	if perr != nil {
		return bu, uuid.Nil, uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusBadRequest, "invalid meeting id")
	}
	orgID, err = a.miniAppMeetingOrganization(c, meetingID)
	if err != nil {
		if model.IsNotFound(err) {
			return bu, uuid.Nil, uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusNotFound, "not_found")
		}
		return bu, uuid.Nil, uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	organizerID, eerr := a.App.EnsureMiniAppOrganizer(c.Context(), bu.Email, bu.TelegramID)
	if eerr != nil {
		if errors.Is(eerr, application.ErrTelegramLinkedToOtherAccount) {
			return bu, uuid.Nil, uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusConflict, "telegram_linked_to_other_account")
		}
		return bu, uuid.Nil, uuid.Nil, uuid.Nil, fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return bu, orgID, organizerID, meetingID, nil
}

func (a *API) MiniAppAddParticipant(c *fiber.Ctx) error {
	bu, orgID, organizerID, meetingID, err := a.resolveMiniAppMeetingOp(c)
	if err != nil {
		return err
	}
	var req miniappParticipantRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if err := a.App.AddParticipant(c.Context(), orgID, organizerID, meetingID, req.Email); err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.Context(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m, resolveLoc(bu.Timezone))})
}

func (a *API) MiniAppChangeSeriesEnd(c *fiber.Ctx) error {
	bu, organizationID, organizerID, meetingID, err := a.resolveMiniAppMeetingOp(c)
	if err != nil {
		return err
	}
	var req struct {
		Until string `json:"until"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if _, _, err := a.App.ChangeSeriesEnd(c.Context(), organizationID, organizerID, meetingID, req.Until); err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.Context(), organizationID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m, resolveLoc(bu.Timezone))})
}

func (a *API) MiniAppRemoveParticipant(c *fiber.Ctx) error {
	bu, orgID, organizerID, meetingID, err := a.resolveMiniAppMeetingOp(c)
	if err != nil {
		return err
	}
	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email required")
	}
	if err := a.App.RemoveParticipant(c.Context(), orgID, organizerID, meetingID, email); err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.Context(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	return c.JSON(fiber.Map{"meeting": a.toMeetingDTO(c.Context(), m, resolveLoc(bu.Timezone))})
}
