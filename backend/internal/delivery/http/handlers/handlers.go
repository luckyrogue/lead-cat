package handlers

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/telegram"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/copy"
	"github.com/go-telegram/bot"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
)

type API struct {
	App     *application.Services
	Bot     *bot.Bot
	RDB     *redis.Client
	Log     *zap.Logger
	TMA     *telegram.InitDataValidator
	Version string
}

func (a *API) Health(c *fiber.Ctx) error {
	pg := "ok"
	if err := a.App.Store.Ping(c.Context()); err != nil {
		pg = "down"
	}
	rd := "ok"
	if err := a.RDB.Ping(c.Context()).Err(); err != nil {
		rd = "down"
	}
	botOK := a.Bot != nil
	status := fiber.StatusOK
	if pg != "ok" || rd != "ok" {
		status = fiber.StatusServiceUnavailable
	}
	v := a.Version
	if v == "" {
		v = "dev"
	}
	return c.Status(status).JSON(fiber.Map{
		"postgres": pg,
		"redis":    rd,
		"bot_ok":   botOK,
		"version":  v,
	})
}

func (a *API) Metrics(c *fiber.Ctx) error {
	if tok := os.Getenv("METRICS_TOKEN"); tok != "" && c.Get("Authorization") != "Bearer "+tok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(c.Context())
	return nil
}

func (a *API) Me(c *fiber.Ctx) error {
	sub, _ := c.Locals("user_sub").(string)
	u, err := a.App.GetMe(c.Context(), sub)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, copy.APIError("unauthorized"))
	}
	return c.JSON(fiber.Map{
		"id":          u.ID,
		"email":       u.Email,
		"telegram_id": u.TelegramID,
	})
}

func (a *API) LinkTelegram(c *fiber.Ctx) error {
	initData := c.Get("X-Telegram-Init-Data")
	if initData == "" {
		return fiber.NewError(fiber.StatusBadRequest, "X-Telegram-Init-Data required")
	}
	user, err := a.TMA.Validate(initData)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid initData")
	}
	uid := c.Locals("user_id").(uuid.UUID)
	if err := a.App.LinkTelegram(c.Context(), uid, user.ID, user.Username); err != nil {
		a.logHTTPError(c, "link telegram", err)
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) ListWorkspaces(c *fiber.Ctx) error {
	uid := c.Locals("user_id").(uuid.UUID)
	list, err := a.App.ListWorkspaces(c.Context(), uid)
	if err != nil {
		a.logHTTPError(c, "list workspaces", err)
		return fiber.NewError(fiber.StatusInternalServerError, copy.APIError("internal"))
	}
	return c.JSON(list)
}

func (a *API) CreateWorkspace(c *fiber.Ctx) error {
	var body struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, copy.APIError("bad_request"))
	}
	uid := c.Locals("user_id").(uuid.UUID)
	w, err := a.App.CreateWorkspace(c.Context(), body.Slug, body.Name, uid)
	if err != nil {
		a.logHTTPError(c, "create workspace", err)
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(w)
}

func (a *API) GetWorkspace(c *fiber.Ctx) error {
	id := c.Locals("workspace_id").(uuid.UUID)
	w, err := a.App.GetWorkspace(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, copy.APIError("not_found"))
	}
	return c.JSON(w)
}

func (a *API) LinkChat(c *fiber.Ctx) error {
	id := c.Locals("workspace_id").(uuid.UUID)
	var body struct {
		ChatID int64 `json:"chat_id"`
	}
	_ = c.BodyParser(&body)
	if body.ChatID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "chat_id required")
	}
	if err := a.App.LinkChat(c.Context(), id, body.ChatID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) ChatStatus(c *fiber.Ctx) error {
	id := c.Locals("workspace_id").(uuid.UUID)
	w, err := a.App.GetWorkspace(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, copy.APIError("not_found"))
	}
	linked := w.NotifyChatID != nil
	return c.JSON(fiber.Map{"linked": linked, "notify_chat_id": w.NotifyChatID})
}

func (a *API) GetIntegrations(c *fiber.Ctx) error {
	id := c.Locals("workspace_id").(uuid.UUID)
	v, err := a.App.GetIntegrations(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(v)
}

func (a *API) PatchIntegrations(c *fiber.Ctx) error {
	id := c.Locals("workspace_id").(uuid.UUID)
	var body struct {
		VCSProvider  string `json:"vcs_provider"`
		VCSNamespace string `json:"vcs_namespace"`
		VCSBaseURL   string `json:"vcs_base_url"`
		VCSToken     string `json:"vcs_token"`
		MeetLink     string `json:"meet_link"`
		TZ           string `json:"tz"`
	}
	_ = c.BodyParser(&body)
	if err := a.App.PatchIntegrations(c.Context(), id, body.VCSProvider, body.VCSNamespace, body.VCSBaseURL, body.VCSToken, body.MeetLink, body.TZ); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) VerifyIntegrations(c *fiber.Ctx) error {
	id := c.Locals("workspace_id").(uuid.UUID)
	if err := a.App.VerifyIntegrations(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *API) ListScenarios(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	list, err := a.App.ListScenarios(c.Context(), wid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

func (a *API) CreateScenario(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	var body struct {
		Name       string          `json:"name"`
		Definition json.RawMessage `json:"definition"`
	}
	_ = c.BodyParser(&body)
	sc, err := a.App.CreateScenario(c.Context(), wid, body.Name, body.Definition)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(sc)
}

func (a *API) GetScenario(c *fiber.Ctx) error {
	sid, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scenario id")
	}
	sc, err := a.App.GetScenario(c.Context(), sid)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, copy.APIError("not_found"))
	}
	return c.JSON(sc)
}

func (a *API) UpdateScenario(c *fiber.Ctx) error {
	sid, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scenario id")
	}
	var body struct {
		Name       string          `json:"name"`
		Enabled    *bool           `json:"enabled"`
		Definition json.RawMessage `json:"definition"`
	}
	_ = c.BodyParser(&body)
	sc, err := a.App.UpdateScenario(c.Context(), sid, body.Name, body.Enabled, body.Definition)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(sc)
}

func (a *API) DeleteScenario(c *fiber.Ctx) error {
	sid, err := uuid.Parse(c.Params("sid"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid scenario id")
	}
	if err := a.App.DeleteScenario(c.Context(), sid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) RunScenario(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	sid, _ := uuid.Parse(c.Params("sid"))
	runID, err := a.App.RunScenario(c.Context(), sid, wid)
	if err != nil {
		a.logHTTPError(c, "run scenario", err, zap.String("scenario_id", sid.String()))
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"run_id": runID})
}

func (a *API) ListRuns(c *fiber.Ctx) error {
	sid, _ := uuid.Parse(c.Params("sid"))
	runs, err := a.App.ListRuns(c.Context(), sid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(runs)
}

func (a *API) ListMembers(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	m, err := a.App.ListMembers(c.Context(), wid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(m)
}

func (a *API) AddMember(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	var body struct {
		TelegramUsername string `json:"telegram_username"`
		Role             string `json:"role"`
	}
	_ = c.BodyParser(&body)
	if body.Role == "" {
		body.Role = "developer"
	}
	m, err := a.App.AddMember(c.Context(), wid, body.TelegramUsername, body.Role)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(m)
}

func (a *API) DeleteMember(c *fiber.Ctx) error {
	mid, err := uuid.Parse(c.Params("memberId"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid member id")
	}
	if err := a.App.DeleteMember(c.Context(), mid); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (a *API) SyncChatMembers(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	if a.Bot == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "telegram bot unavailable")
	}
	n, err := telegram.SyncChatMembers(c.Context(), a.Bot, a.App.Store, wid)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"synced": n})
}

func (a *API) PatchMemberVCS(c *fiber.Ctx) error {
	wid := c.Locals("workspace_id").(uuid.UUID)
	username := strings.TrimPrefix(c.Params("username"), "@")
	var body struct {
		GitHubLogin string `json:"github_login"`
		GitLabLogin string `json:"gitlab_login"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, copy.APIError("bad_request"))
	}
	if err := a.App.SetMemberVCS(c.Context(), wid, username, body.GitHubLogin, body.GitLabLogin); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}
