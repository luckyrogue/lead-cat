package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func webUser(c *fiber.Ctx) (model.PlatformUser, bool) {
	u, ok := c.Locals("web_user").(model.PlatformUser)
	return u, ok
}

func mapMeetingWriteError(err error) error {
	switch {
	case errors.Is(err, application.ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	case errors.Is(err, application.ErrInvalidInput), errors.Is(err, application.ErrGoogleNotConfigured):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
}

func (a *API) WebListMeetings(c *fiber.Ctx) error {
	user, ok := webUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	ms, err := a.App.ListMeetings(c.UserContext(), orgID, user.ID)
	if err != nil {
		a.Log.Error("web_meetings_list_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	if ms == nil {
		ms = []model.Meeting{}
	}
	return c.JSON(fiber.Map{"meetings": ms})
}

func (a *API) WebGetMeeting(c *fiber.Ctx) error {
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	meetingID, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_meeting_id")
	}
	m, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	return c.JSON(fiber.Map{"meeting": m})
}

func (a *API) WebCreateMeeting(c *fiber.Ctx) error {
	user, ok := webUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	var req miniappCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	m, err := a.App.CreateMeeting(c.UserContext(), orgID, user.ID, toCreateMeetingInput(req, user.Email))
	if err != nil {
		return mapMeetingWriteError(err)
	}
	a.Log.Info("web_meeting_created",
		zap.String("org_id", orgID.String()),
		zap.String("meeting_id", m.ID.String()))
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"meeting": m})
}

func (a *API) WebUpdateMeeting(c *fiber.Ctx) error {
	user, ok := webUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	meetingID, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_meeting_id")
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	var req miniappUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	if _, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	if scope == "this" {
		m, err := a.App.UpdateMeeting(c.UserContext(), orgID, user.ID, meetingID, application.UpdateMeetingInput{
			Dept: req.Dept, Type: req.Type, Host: req.Host,
			Date: req.Date, Start: req.Start, End: req.End, Description: req.Desc,
		})
		if err != nil {
			return mapMeetingWriteError(err)
		}
		a.Log.Info("web_meeting_updated", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()))
		return c.JSON(fiber.Map{"meeting": m})
	}
	if req.Date != nil {
		return fiber.NewError(fiber.StatusBadRequest, "date_immutable_for_series")
	}
	n, err := a.App.UpdateWholeSeries(c.UserContext(), orgID, user.ID, meetingID, mapToSeriesUpdateInput(req))
	if err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.Log.Info("web_meeting_series_updated", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()), zap.Int("count", n))
	return c.JSON(fiber.Map{"meeting": m})
}

func (a *API) WebDeleteMeeting(c *fiber.Ctx) error {
	user, ok := webUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	meetingID, err := uuid.Parse(c.Params("mid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_meeting_id")
	}
	scope, err := parseScope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if _, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	}
	if scope == "this" {
		if err := a.App.CancelMeeting(c.UserContext(), orgID, user.ID, meetingID); err != nil {
			return mapMeetingWriteError(err)
		}
		a.Log.Info("web_meeting_cancelled", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()))
		return c.SendStatus(fiber.StatusNoContent)
	}
	n, err := a.App.CancelWholeSeries(c.UserContext(), orgID, user.ID, meetingID)
	if err != nil {
		return mapMeetingWriteError(err)
	}
	a.Log.Info("web_meeting_series_cancelled", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()), zap.Int("count", n))
	return c.SendStatus(fiber.StatusNoContent)
}
