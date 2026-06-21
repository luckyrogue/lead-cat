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
	"github.com/luckyrogue/lead-cat/internal/delivery/http/httperr"
	"github.com/luckyrogue/lead-cat/internal/delivery/http/middleware"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
	"github.com/luckyrogue/lead-cat/internal/platform/config"
)

func NewApp(cfg config.Config, store middleware.OrgMemberResolver, cipher *crypto.TokenCipher, rdb *redis.Client, tg *bot.Bot, log *zap.Logger, services *application.Services) (*fiber.App, error) {
	_ = cipher
	app := fiber.New(fiber.Config{
		EnableTrustedProxyCheck: cfg.TrustProxyHeaders,
		ProxyHeader:             "X-Real-IP",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": "error", "message": httperr.PublicMessage(err)})
		},
	})
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.SecurityHeaders(cfg.IsProduction()))
	app.Use(middleware.RequestContext())
	app.Use(middleware.PrometheusHTTP())
	app.Use(logger.New())
	origins := strings.Split(cfg.CORSAllowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	if cfg.AppBaseURL != "" {
		origins = append(origins, cfg.AppBaseURL)
	}
	origins = dedupeNonEmpty(origins)
	app.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(origins, ","),
		AllowHeaders:     "Authorization, Content-Type, X-Telegram-Init-Data, X-CSRF-Token, X-Org-Id",
		AllowCredentials: true,
	}))

	miniappToken, err := platformauth.NewMiniAppToken(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	if err != nil {
		return nil, err
	}

	rateLimit := func(max int, window time.Duration, prefix string) fiber.Handler {
		if cfg.RateLimitDisabled {
			return func(c *fiber.Ctx) error { return c.Next() }
		}
		return middleware.RateLimit(rdb, log, max, window, prefix, cfg.TrustProxyHeaders, cfg.AuthDevMode)
	}

	api := &handlers.API{
		App:               services,
		Bot:               tg,
		RDB:               rdb,
		Log:               log,
		InitData:          telegram.NewInitDataValidator(cfg.BotToken),
		Version:           os.Getenv("APP_VERSION"),
		MiniAppToken:      miniappToken,
		AuthDevMode:       cfg.AuthDevMode,
		AppEnv:            cfg.AppEnv,
		MetricsToken:      cfg.MetricsToken,
		TrustProxyHeaders: cfg.TrustProxyHeaders,
		WebSessionTTL:     cfg.WebSessionTTL,

		WebCookieDomain: cfg.WebCookieDomain,
	}

	app.Get("/api/health", api.Health)
	app.Get("/openapi.json", handlers.OpenAPI)
	app.Get("/metrics", api.Metrics)

	app.Post("/api/auth/miniapp", rateLimit(10, time.Minute, "miniapp_auth"), api.MiniAppAuth)

	webAuth := middleware.NewWebAuth(services)
	web := app.Group("/api/auth/web")
	web.Get("/:provider/start", api.WebAuthStart)
	web.Get("/:provider/callback", api.WebAuthCallback)
	web.Post("/magic/request", rateLimit(5, 15*time.Minute, "magic"), api.WebMagicRequest)
	web.Get("/magic/verify", api.WebMagicVerify)
	web.Post("/magic/verify", rateLimit(10, time.Minute, "magic_verify"), api.WebMagicVerifyPOST)
	web.Post("/logout", webAuth.Middleware, api.WebLogout)
	web.Get("/me", webAuth.Middleware, api.WebMe)
	web.Get("/me/settings", webAuth.Middleware, api.WebGetMeSettings)
	web.Patch("/me/settings", webAuth.Middleware, api.WebPatchMeSettings)
	web.Get("/me/invites", webAuth.Middleware, api.WebMyInvites)
	web.Post("/me/invites/:iid/accept", webAuth.Middleware, api.WebAcceptInvite)
	web.Post("/me/invites/:iid/decline", webAuth.Middleware, api.WebDeclineInvite)
	web.Post("/me/join-requests", webAuth.Middleware, api.WebRequestToJoin)
	web.Get("/me/join-requests", webAuth.Middleware, api.WebMyJoinRequests)

	registerRetiredPlatformAuth(app)
	app.All("/api/me", handlers.PlatformGone)
	app.All("/api/me/*", handlers.PlatformGone)
	app.All("/api/workspaces", handlers.PlatformGone)
	app.All("/api/workspaces/*", handlers.PlatformGone)

	orgs := app.Group("/api/orgs", webAuth.Middleware)
	orgs.Post("", api.CreateOrg)
	orgs.Get("", api.ListMyOrgs)
	scoped := orgs.Group("/:id", middleware.RequireOrgMember(store))
	scoped.Get("/meetings", api.WebListMeetings)
	scoped.Post("/meetings", api.WebCreateMeeting)
	scoped.Get("/meetings/:mid", api.WebGetMeeting)
	scoped.Patch("/meetings/:mid", api.WebUpdateMeeting)
	scoped.Delete("/meetings/:mid", api.WebDeleteMeeting)
	scoped.Post("/meetings/:mid/participants", api.WebAddParticipant)
	scoped.Delete("/meetings/:mid/participants", api.WebRemoveParticipant)
	scoped.Patch("/meetings/:mid/series-end", api.WebChangeSeriesEnd)
	scoped.Get("/members", api.ListOrgMembers)
	scoped.Patch("/members/:uid/role", middleware.RequireOrgRole("admin"), api.UpdateMemberRole)
	scoped.Delete("/members/:uid", middleware.RequireOrgRole("admin"), api.RemoveMember)
	scoped.Get("/invites", middleware.RequireOrgRole("admin"), api.ListInvites)
	scoped.Post("/invites", middleware.RequireOrgRole("admin"), api.InviteMember)
	scoped.Delete("/invites/:iid", middleware.RequireOrgRole("admin"), api.DeleteInvite)
	scoped.Get("/join-requests", middleware.RequireOrgRole("admin"), api.OrgJoinRequests)
	scoped.Post("/join-requests/:rid/accept", middleware.RequireOrgRole("admin"), api.AcceptJoinRequest)
	scoped.Post("/join-requests/:rid/decline", middleware.RequireOrgRole("admin"), api.DeclineJoinRequest)

	miniappAuth := middleware.NewMiniAppAuth(miniappToken, services)
	miniapp := app.Group("/api/miniapp", miniappAuth.Middleware)
	miniapp.Get("/me", api.MiniAppMe)
	miniapp.Get("/settings", api.MiniAppGetSettings)
	miniapp.Patch("/settings", api.MiniAppPatchSettings)
	miniapp.Get("/meetings", api.MiniAppMyMeetings)
	miniapp.Get("/meetings/:id", api.MiniAppGetMeeting)
	miniapp.Get("/schedule", api.MiniAppSchedule)
	miniapp.Get("/employees", api.MiniAppEmployees)
	miniapp.Post("/free-slots", api.MiniAppFreeSlots)
	miniapp.Post("/meetings", api.MiniAppCreateMeeting)
	miniapp.Post("/conflicts", api.MiniAppConflicts)
	miniapp.Patch("/meetings/:id", api.MiniAppUpdateMeeting)
	miniapp.Delete("/meetings/:id", api.MiniAppDeleteMeeting)
	miniapp.Post("/meetings/:id/participants", api.MiniAppAddParticipant)
	miniapp.Delete("/meetings/:id/participants", api.MiniAppRemoveParticipant)
	miniapp.Patch("/meetings/:id/series-end", api.MiniAppChangeSeriesEnd)

	booking := app.Group("/api/booking", webAuth.Middleware)
	booking.Get("/event-types", api.BookingListEventTypes)
	booking.Post("/event-types", api.BookingCreateEventType)
	booking.Patch("/event-types/:id", api.BookingUpdateEventType)
	booking.Delete("/event-types/:id", api.BookingDeleteEventType)

	surveys := app.Group("/api/surveys", webAuth.Middleware)
	surveys.Get("", api.SurveyList)
	surveys.Post("", api.SurveyCreate)
	surveys.Get("/:id", api.SurveyGet)
	surveys.Patch("/:id", api.SurveyUpdate)
	surveys.Delete("/:id", api.SurveyDelete)
	surveys.Get("/:id/responses", api.SurveyResponses)
	surveys.Get("/:id/responses.csv", api.SurveyResponsesCSV)

	app.Get("/api/book/:slug", rateLimit(60, time.Minute, "book_get"), api.PublicBooking)
	app.Post("/api/book/:slug", rateLimit(10, time.Hour, "book_post"), api.PublicBookingSubmit)

	// Public survey delivery (unauthenticated, token-based, rate-limited).
	app.Get("/api/survey/:token", api.PublicSurveyGet)
	app.Post("/api/survey/:token", rateLimit(20, time.Minute, "survey_submit"), api.PublicSurveySubmit)

	app.Get("/api/calendar/connect/:provider/callback", api.CalendarConnectCallback)

	calendarWeb := app.Group("/api/calendar", webAuth.Middleware)
	calendarWeb.Post("/connect/:provider/start", api.CalendarConnectStart)
	calendarWeb.Get("/connections", api.CalendarConnectionsList)
	calendarWeb.Delete("/connections/:provider", api.CalendarDisconnect)

	miniapp.Post("/calendar/connect/:provider/start", api.CalendarConnectStart)
	miniapp.Get("/calendar/connections", api.CalendarConnectionsList)
	miniapp.Delete("/calendar/connections/:provider", api.CalendarDisconnect)

	miniappAdmin := miniapp.Group("/admin", middleware.RequireBotAdmin(cfg.BotAdminTelegramIDs))
	miniappAdmin.Get("/organization", api.MiniAppAdminGetWorkspace)
	miniappAdmin.Post("/organization", api.MiniAppAdminCreateWorkspace)
	miniappAdmin.Get("/workspace", handlers.DeprecatedAdminWorkspace(api.MiniAppAdminGetWorkspace))
	miniappAdmin.Post("/workspace", handlers.DeprecatedAdminWorkspace(api.MiniAppAdminCreateWorkspace))
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

func registerRetiredPlatformAuth(app *fiber.App) {
	legacyPrefixes := []string{
		"/api/auth/email",
		"/api/auth/phone",
		"/api/auth/passkey",
		"/api/auth/oauth",
		"/api/auth/otp",
		"/api/auth/login",
		"/api/auth/register",
		"/api/auth/refresh",
	}
	for _, prefix := range legacyPrefixes {
		app.All(prefix, handlers.PlatformGone)
		app.All(prefix+"/*", handlers.PlatformGone)
	}
	app.All("/api/auth/*", handlers.PlatformGone)
}

func dedupeNonEmpty(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
