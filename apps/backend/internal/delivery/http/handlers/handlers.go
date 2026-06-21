package handlers

import (
	"crypto/subtle"
	"time"

	"github.com/go-telegram/bot"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
)

type API struct {
	App               *application.Services
	Bot               *bot.Bot
	RDB               *redis.Client
	Log               *zap.Logger
	InitData          *telegram.InitDataValidator
	MiniAppToken      *platformauth.MiniAppToken
	AuthDevMode       bool
	AppEnv            string
	MetricsToken      string
	TrustProxyHeaders bool
	WebSessionTTL     time.Duration
	Version           string

	WebCookieDomain string
}

func (a *API) Health(c *fiber.Ctx) error {
	pg := "ok"
	if err := a.App.Ping(c.Context()); err != nil {
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
	if a.MetricsToken != "" {
		auth := c.Get("Authorization")
		want := "Bearer " + a.MetricsToken
		if subtle.ConstantTimeCompare([]byte(auth), []byte(want)) != 1 {
			return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
		}
	}
	fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())(c.Context())
	return nil
}
