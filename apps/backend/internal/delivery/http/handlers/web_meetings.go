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
)

func webUser(c *fiber.Ctx) (model.PlatformUser, bool) {
	u, ok := c.Locals("web_user").(model.PlatformUser)
	return u, ok
}

func mapMeetingWriteError(err error) error {
	switch {
	case errors.Is(err, application.ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	case errors.Is(err, model.ErrMeetingNotEditable):
		return fiber.NewError(fiber.StatusConflict, "meeting_not_editable")
	case errors.Is(err, application.ErrInvalidInput), errors.Is(err, application.ErrGoogleNotConfigured):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
}

type webParticipantRequest struct {
	Email string `json:"email"`
}

type seriesEndRequest struct {
	Until string `json:"until"`
}

func (a *API) WebChangeSeriesEnd(c *fiber.Ctx) error {
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
	var req seriesEndRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	added, removed, err := a.App.ChangeSeriesEnd(c.UserContext(), orgID, user.ID, meetingID, req.Until)
	if err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.Log.Info("web_series_end_changed", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()), zap.Int("added", added), zap.Int("removed", removed))
	return c.JSON(fiber.Map{"meeting": m, "added": added, "removed": removed})
}

func (a *API) WebAddParticipant(c *fiber.Ctx) error {
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
	var req webParticipantRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	if err := a.App.AddParticipant(c.UserContext(), orgID, user.ID, meetingID, req.Email); err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.Log.Info("web_participant_added", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()))
	return c.JSON(fiber.Map{"meeting": m})
}

func (a *API) WebRemoveParticipant(c *fiber.Ctx) error {
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
	email := c.Query("email")
	if email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email_required")
	}
	if err := a.App.RemoveParticipant(c.UserContext(), orgID, user.ID, meetingID, email); err != nil {
		return mapMeetingWriteError(err)
	}
	m, err := a.App.GetMeeting(c.UserContext(), orgID, meetingID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "internal")
	}
	a.Log.Info("web_participant_removed", zap.String("org_id", orgID.String()), zap.String("meeting_id", meetingID.String()))
	return c.JSON(fiber.Map{"meeting": m})
}

// parseMeetingFilter turns raw query values into a model.MeetingFilter.
// status "" or "all" means no status filter; from/to are YYYY-MM-DD (to is an
// inclusive day, stored as the exclusive next-day bound); organizer is a UUID.
func parseMeetingFilter(status, from, to, dept, organizer string) (model.MeetingFilter, error) {
	f := model.MeetingFilter{Dept: strings.TrimSpace(dept)}
	switch status {
	case "", "all":
	case "scheduled", "cancelled":
		f.Status = status
	default:
		return f, fmt.Errorf("invalid status")
	}
	if from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			return f, fmt.Errorf("invalid from")
		}
		f.From = &t
	}
	if to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			return f, fmt.Errorf("invalid to")
		}
		end := t.AddDate(0, 0, 1)
		f.To = &end
	}
	if organizer != "" {
		id, err := uuid.Parse(organizer)
		if err != nil {
			return f, fmt.Errorf("invalid organizer")
		}
		f.Organizer = &id
	}
	return f, nil
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
	filter, err := parseMeetingFilter(c.Query("status"), c.Query("from"), c.Query("to"), c.Query("dept"), c.Query("organizer"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	ms, err := a.App.ListMeetingsFiltered(c.UserContext(), orgID, user.ID, filter)
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
