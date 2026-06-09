package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// tmaAdminBotUser extracts the authed admin bot user. Caller may assume role==admin
// (the RequireBotAdmin middleware enforced it upstream).
func tmaAdminBotUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok && bu.Role == "admin"
}

// withAuditActor enriches the user-context with the actor identity so the
// application-layer Audit helper can record who did what.
func (a *API) withAuditActor(c *fiber.Ctx) {
	bu, ok := tmaAdminBotUser(c)
	if !ok {
		return
	}
	c.SetUserContext(application.WithAuditActor(c.UserContext(), application.AuditContext{
		UserID:     bu.ID,
		TelegramID: bu.TelegramID,
		Email:      bu.Email,
	}))
}

// adminWorkspaceID returns the singleton Lead Cat workspace id, creating it
// implicitly on first call. The admin's platform user_id (if any) becomes
// owner_user_id; if the admin has no paired platform account, NULL is stored.
func (a *API) adminWorkspaceID(c *fiber.Ctx) (uuid.UUID, error) {
	bu, _ := tmaAdminBotUser(c)
	ownerID := uuid.Nil
	if u, ok, err := a.App.Store.GetPlatformUserIDByTelegramID(c.Context(), bu.TelegramID); err == nil && ok {
		ownerID = u
	}
	return a.App.EnsureSingleWorkspace(c.Context(), ownerID)
}

// GET /api/tma/admin/workspace
func (a *API) TMAAdminGetWorkspace(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	w, err := a.App.GetWorkspace(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	view, err := a.App.GetIntegrations(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var chatID int64
	if w.NotifyChatID != nil {
		chatID = *w.NotifyChatID
	}
	return c.JSON(fiber.Map{
		"id":                 id,
		"name":               w.Name,
		"tz":                 w.TZ,
		"meet_link":          w.MeetLink,
		"has_google":         view.HasGoogle,
		"google_subject":     view.GoogleSubject,
		"google_calendar_id": view.GoogleCalendarID,
		"has_chat":           w.NotifyChatID != nil,
		"chat_id":            chatID,
	})
}

// POST /api/tma/admin/workspace
func (a *API) TMAAdminCreateWorkspace(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	return c.JSON(fiber.Map{"id": id})
}

// GET /api/tma/admin/integrations
func (a *API) TMAAdminGetIntegrations(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	view, err := a.App.GetIntegrations(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"has_google":         view.HasGoogle,
		"google_subject":     view.GoogleSubject,
		"google_calendar_id": view.GoogleCalendarID,
		"meet_link":          view.MeetLink,
		"tz":                 view.TZ,
	})
}

// PATCH /api/tma/admin/integrations
func (a *API) TMAAdminPatchIntegrations(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	var body struct {
		GoogleSAJSON     string `json:"google_sa_json"`
		GoogleSubject    string `json:"google_subject"`
		GoogleCalendarID string `json:"google_calendar_id"`
		MeetLink         string `json:"meet_link"`
		TZ               string `json:"tz"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if body.GoogleSAJSON != "" || body.GoogleSubject != "" || body.GoogleCalendarID != "" {
		if err := a.App.SetGoogleConfig(c.Context(), id, body.GoogleSAJSON, body.GoogleSubject, body.GoogleCalendarID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		a.App.Audit(c.UserContext(), "google_config_updated", "workspace", id.String(), map[string]any{
			"subject":         body.GoogleSubject,
			"calendar_id":     body.GoogleCalendarID,
			"has_new_sa_json": body.GoogleSAJSON != "",
		})
	}
	if body.MeetLink != "" || body.TZ != "" {
		if err := a.App.PatchIntegrations(c.Context(), id, "", "", "", "", body.MeetLink, body.TZ); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// POST /api/tma/admin/integrations/verify
func (a *API) TMAAdminVerifyIntegrations(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	res, err := a.App.VerifyGoogleIntegration(c.Context(), id)
	if err != nil {
		code, status := mapVerifyError(err)
		a.App.Audit(c.UserContext(), "google_verified", "workspace", id.String(), map[string]any{
			"ok":         false,
			"error_code": code,
		})
		return fiber.NewError(status, code)
	}
	a.App.Audit(c.UserContext(), "google_verified", "workspace", id.String(), map[string]any{
		"ok":               true,
		"calendar_summary": res.CalendarSummary,
		"time_zone":        res.TimeZone,
	})
	return c.JSON(res)
}

func mapVerifyError(err error) (code string, status int) {
	switch {
	case errors.Is(err, application.ErrGoogleSAInvalid):
		return "google_sa_invalid", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleSubjectInvalid):
		return "google_subject_invalid", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleCalendarNotAccessible):
		return "google_calendar_not_accessible", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleAPIDisabled):
		return "google_api_disabled", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleNotConfigured):
		return "google_not_configured", fiber.StatusBadRequest
	default:
		return "internal_error", fiber.StatusInternalServerError
	}
}

// GET /api/tma/admin/chat/status
func (a *API) TMAAdminChatStatus(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	w, err := a.App.GetWorkspace(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var chatID int64
	if w.NotifyChatID != nil {
		chatID = *w.NotifyChatID
	}
	return c.JSON(fiber.Map{
		"linked":  w.NotifyChatID != nil,
		"chat_id": chatID,
	})
}

// POST /api/tma/admin/chat/link
func (a *API) TMAAdminChatLink(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	var body struct {
		ChatID    int64  `json:"chat_id"`
		ChatTitle string `json:"chat_title"`
	}
	if err := c.BodyParser(&body); err != nil || body.ChatID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if err := a.App.LinkChat(c.Context(), id, body.ChatID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.App.Audit(c.UserContext(), "chat_linked", "workspace", id.String(), map[string]any{
		"chat_id":    body.ChatID,
		"chat_title": body.ChatTitle,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

// GET /api/tma/admin/members
func (a *API) TMAAdminListMembers(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	members, err := a.App.ListMembers(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"members": members})
}

// POST /api/tma/admin/members/sync-chat
func (a *API) TMAAdminMembersSyncChat(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	n, err := a.App.SyncChatMembers(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.App.Audit(c.UserContext(), "members_synced", "workspace", id.String(), map[string]any{
		"added": n,
	})
	return c.JSON(fiber.Map{"added": n})
}
