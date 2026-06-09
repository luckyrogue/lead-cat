package http

import (
	"os"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/handlers"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
	"github.com/luckyrogue/lead-cat/internal/platform/config"
)

func NewApp(cfg config.Config, store *postgres.Store, cipher *crypto.TokenCipher, rdb *redis.Client, tg *bot.Bot, log *zap.Logger, services *application.Services) (*fiber.App, error) {
	_ = cipher
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": "error", "message": err.Error()})
		},
	})
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.RequestContext())
	app.Use(middleware.PrometheusHTTP())
	app.Use(logger.New())
	origins := strings.Split(cfg.CORSAllowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(origins, ","),
		AllowHeaders: "Authorization, Content-Type, X-Telegram-Init-Data",
	}))

	miniappToken, err := platformauth.NewMiniAppToken(cfg.JWTSecret, cfg.JWTIssuer, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	api := &handlers.API{
		App:          services,
		Bot:          tg,
		RDB:          rdb,
		Log:          log,
		InitData:     telegram.NewInitDataValidator(cfg.BotToken),
		Version:      os.Getenv("APP_VERSION"),
		MiniAppToken: miniappToken,
		AuthDevMode:  cfg.AuthDevMode,
	}

	app.Get("/api/health", api.Health)
	app.Get("/openapi.json", handlers.OpenAPI)
	app.Get("/metrics", api.Metrics)

	app.Post("/api/auth/miniapp", api.MiniAppAuth)
	app.All("/api/auth/*", handlers.PlatformGone)
	app.All("/api/me", handlers.PlatformGone)
	app.All("/api/me/*", handlers.PlatformGone)
	app.All("/api/workspaces", handlers.PlatformGone)
	app.All("/api/workspaces/*", handlers.PlatformGone)

	miniappAuth := middleware.NewMiniAppAuth(miniappToken, store)
	miniapp := app.Group("/api/miniapp", miniappAuth.Middleware)
	miniapp.Get("/me", api.MiniAppMe)
	miniapp.Get("/meetings", api.MiniAppMyMeetings)
	miniapp.Get("/schedule", api.MiniAppSchedule)
	miniapp.Get("/employees", api.MiniAppEmployees)
	miniapp.Post("/free-slots", api.MiniAppFreeSlots)
	miniapp.Post("/meetings", api.MiniAppCreateMeeting)
	miniapp.Post("/conflicts", api.MiniAppConflicts)
	miniapp.Patch("/meetings/:id", api.MiniAppUpdateMeeting)
	miniapp.Delete("/meetings/:id", api.MiniAppDeleteMeeting)

	miniappAdmin := miniapp.Group("/admin", middleware.RequireBotAdmin)
	miniappAdmin.Get("/workspace", api.MiniAppAdminGetWorkspace)
	miniappAdmin.Post("/workspace", api.MiniAppAdminCreateWorkspace)
	miniappAdmin.Get("/integrations", api.MiniAppAdminGetIntegrations)
	miniappAdmin.Patch("/integrations", api.MiniAppAdminPatchIntegrations)
	miniappAdmin.Post("/integrations/verify", api.MiniAppAdminVerifyIntegrations)
	miniappAdmin.Get("/chat/status", api.MiniAppAdminChatStatus)
	miniappAdmin.Post("/chat/link", api.MiniAppAdminChatLink)
	miniappAdmin.Get("/members", api.MiniAppAdminListMembers)
	miniappAdmin.Post("/members/sync-chat", api.MiniAppAdminMembersSyncChat)
	miniappAdmin.Get("/audit", api.MiniAppAdminListAudit)

	if stat, err := os.Stat(cfg.StaticDir); err == nil && stat.IsDir() {
		app.Static("/", cfg.StaticDir, fiber.Static{
			Index:  "index.html",
			Browse: false,
		})
		app.Get("*", func(c *fiber.Ctx) error {
			if strings.HasPrefix(c.Path(), "/api") || c.Path() == "/metrics" {
				return c.Next()
			}
			return c.SendFile(cfg.StaticDir + "/index.html")
		})
	}

	return app, nil
}
