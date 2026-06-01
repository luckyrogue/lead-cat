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

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/delivery/http/handlers"
	"github.com/Jaryq-Lab/notify-bot/internal/delivery/http/middleware"
	oauthsvc "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/auth/oauth"
	webauthnsvc "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/auth/webauthn"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/telegram"
	platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/config"
)

func NewApp(cfg config.Config, store *postgres.Store, cipher *crypto.TokenCipher, rdb *redis.Client, tg *bot.Bot, log *zap.Logger, services *application.Services) (*fiber.App, error) {
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

	jwtSvc, err := platformauth.NewJWT(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	if err != nil {
		return nil, err
	}
	tmaToken, err := platformauth.NewTMAToken(cfg.JWTSecret, cfg.JWTIssuer, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	sessStore := platformauth.NewSessionStore(rdb)
	otpSvc := platformauth.NewOTP(rdb, log, cfg.AuthOTPLog)
	wan, err := webauthnsvc.NewService(cfg.WebAuthnRPID, cfg.WebAuthnRPOrigin, store, sessStore)
	if err != nil {
		return nil, err
	}
	oauthSvc := oauthsvc.NewService(oauthsvc.Config{
		WebappURL:          cfg.WebappURL,
		GitHubClientID:     cfg.GitHubOAuthClientID,
		GitHubClientSecret: cfg.GitHubOAuthClientSecret,
		GitLabClientID:     cfg.GitLabOAuthClientID,
		GitLabClientSecret: cfg.GitLabOAuthClientSecret,
		GitLabBaseURL:      cfg.GitLabOAuthBaseURL,
	}, store, sessStore)

	authH := &handlers.AuthAPI{
		Store:        store,
		JWT:          jwtSvc,
		OTP:          otpSvc,
		WebAuthn:     wan,
		OAuth:        oauthSvc,
		Webapp:       cfg.WebappURL,
		OTPLogExpose: cfg.AuthOTPLog,
	}
	authMW := middleware.NewAuth(cfg, jwtSvc, store, log)

	api := &handlers.API{
		App:         services,
		Bot:         tg,
		RDB:         rdb,
		Log:         log,
		TMA:         telegram.NewInitDataValidator(cfg.BotToken),
		Version:     os.Getenv("APP_VERSION"),
		TMAToken:    tmaToken,
		AuthDevMode: cfg.AuthDevMode,
	}

	app.Get("/api/health", api.Health)
	app.Get("/metrics", api.Metrics)

	authPub := app.Group("/api/auth")
	authPub.Get("/config", authH.Config)
	authPub.Post("/email/send-code", authH.SendEmailCode)
	authPub.Post("/email/verify", authH.VerifyEmailCode)
	authPub.Post("/phone/send-code", authH.SendPhoneCode)
	authPub.Post("/phone/verify", authH.VerifyPhoneCode)
	authPub.Post("/passkey/login/begin", authH.PasskeyLoginBegin)
	authPub.Post("/passkey/login/finish", authH.PasskeyLoginFinish)
	authPub.Get("/oauth/:provider", authH.OAuthStart)
	authPub.Get("/oauth/callback", authH.OAuthCallback)
	authPub.Post("/tma", api.TMAAuth)

	ap := app.Group("/api", authMW.Middleware)
	ap.Post("/auth/passkey/register/begin", authH.PasskeyRegisterBegin)
	ap.Post("/auth/passkey/register/finish", authH.PasskeyRegisterFinish)
	ap.Get("/me", api.Me)
	ap.Post("/me/link-telegram", api.LinkTelegram)
	ap.Get("/workspaces", api.ListWorkspaces)
	ap.Post("/workspaces", api.CreateWorkspace)

	ws := ap.Group("/workspaces/:id", middleware.RequireWorkspaceAccess(store))
	ws.Get("", api.GetWorkspace)
	ws.Post("/chat/link", api.LinkChat)
	ws.Get("/chat/status", api.ChatStatus)
	ws.Get("/integrations", api.GetIntegrations)
	ws.Patch("/integrations", api.PatchIntegrations)
	ws.Post("/integrations/verify", api.VerifyIntegrations)
	ws.Get("/members", api.ListMembers)
	ws.Post("/members", api.AddMember)
	ws.Delete("/members/:memberId", api.DeleteMember)
	ws.Post("/members/sync-chat", api.SyncChatMembers)
	ws.Patch("/members/:username/vcs", api.PatchMemberVCS)
	ws.Get("/scenarios", api.ListScenarios)
	ws.Post("/scenarios", api.CreateScenario)
	ws.Get("/scenarios/:sid", api.GetScenario)
	ws.Patch("/scenarios/:sid", api.UpdateScenario)
	ws.Delete("/scenarios/:sid", api.DeleteScenario)
	ws.Post("/scenarios/:sid/run", api.RunScenario)
	ws.Get("/scenarios/:sid/runs", api.ListRuns)
	ws.Get("/employees", api.ListEmployees)
	ws.Get("/meetings", api.ListMeetings)
	ws.Post("/meetings", api.CreateMeeting)
	ws.Get("/meetings/:mid", api.GetMeeting)
	ws.Delete("/meetings/:mid", api.DeleteMeeting)
	ws.Post("/meetings/conflicts", api.MeetingConflicts)
	ws.Post("/meetings/free-slots", api.FreeSlots)

	tmaAuth := middleware.NewTMAAuth(tmaToken, store)
	tma := app.Group("/api/tma", tmaAuth.Middleware)
	tma.Get("/me", api.TMAMe)
	tma.Get("/meetings", api.TMAMyMeetings)
	tma.Get("/schedule", api.TMASchedule)
	tma.Get("/employees", api.TMAEmployees)
	tma.Post("/free-slots", api.TMAFreeSlots)

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
