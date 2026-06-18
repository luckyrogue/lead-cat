package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func callerEmail(c *fiber.Ctx) (string, bool) {
	if u, ok := c.Locals("web_user").(model.PlatformUser); ok && u.Email != "" {
		return u.Email, true
	}
	if bu, ok := c.Locals("bot_user").(model.BotUser); ok && bu.Email != "" {
		return bu.Email, true
	}
	return "", false
}

func (a *API) calendarCallbackURL(provider string) string {
	return a.App.AppBaseURL() + "/api/calendar/connect/" + provider + "/callback"
}

func (a *API) CalendarConnectStart(c *fiber.Ctx) error {
	email, ok := callerEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	provider := c.Params("provider")
	authURL, err := a.App.StartCalendarConnect(c.UserContext(), email, provider, a.calendarCallbackURL(provider))
	if errors.Is(err, application.ErrUnknownConnector) {
		return fiber.NewError(fiber.StatusNotFound, "unknown_provider")
	}
	if err != nil {
		a.Log.Error("calendar_connect_start_failed", zap.String("provider", provider), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "start_failed")
	}
	return c.JSON(fiber.Map{"auth_url": authURL})
}

func (a *API) CalendarConnectCallback(c *fiber.Ctx) error {
	provider := c.Params("provider")
	state, code := c.Query("state"), c.Query("code")
	if state == "" || code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "bad_request")
	}
	if err := a.App.FinishCalendarConnect(c.UserContext(), state, code, a.calendarCallbackURL(provider)); err != nil {
		if model.IsNotFound(err) {
			return fiber.NewError(fiber.StatusBadRequest, "bad_state")
		}
		a.Log.Warn("calendar_connect_callback_failed", zap.String("provider", provider), zap.Error(err))
		return fiber.NewError(fiber.StatusBadRequest, "connect_failed")
	}
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(`<!doctype html><meta charset=utf-8><title>Connected</title><body style="font-family:system-ui;text-align:center;padding:3rem"><h2>Calendar connected</h2><p>You can close this tab.</p></body>`)
}

func (a *API) CalendarConnectionsList(c *fiber.Ctx) error {
	email, ok := callerEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	views, err := a.App.ListCalendarConnections(c.UserContext(), email)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	return c.JSON(views)
}

func (a *API) CalendarDisconnect(c *fiber.Ctx) error {
	email, ok := callerEmail(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if err := a.App.DisconnectCalendar(c.UserContext(), email, c.Params("provider")); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "disconnect_failed")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
