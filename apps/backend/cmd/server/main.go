package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application"
	model "github.com/luckyrogue/lead-cat/internal/application/model"
	deliveryhttp "github.com/luckyrogue/lead-cat/internal/delivery/http"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	calendargoogle "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/google"
	calendarms "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/microsoft"
	calendarresolver "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/resolver"
	calendarstub "github.com/luckyrogue/lead-cat/internal/infrastructure/calendar/stub"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	smtpemail "github.com/luckyrogue/lead-cat/internal/infrastructure/email/smtp"
	llmanthropic "github.com/luckyrogue/lead-cat/internal/infrastructure/llm/anthropic"
	google "github.com/luckyrogue/lead-cat/internal/infrastructure/oauth/google"
	microsoft "github.com/luckyrogue/lead-cat/internal/infrastructure/oauth/microsoft"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	asynqqueue "github.com/luckyrogue/lead-cat/internal/infrastructure/queue/asynq"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/telegram"
	"github.com/luckyrogue/lead-cat/internal/platform/config"
	"github.com/luckyrogue/lead-cat/internal/platform/employeedir"
	"github.com/luckyrogue/lead-cat/internal/platform/meeting_notifier"
	"github.com/luckyrogue/lead-cat/internal/platform/observability/log"
	"github.com/luckyrogue/lead-cat/internal/platform/reminder_scheduler"
)

func main() {
	logger := log.MustNew(envOr("LOG_LEVEL", "info"), envOr("LOG_FORMAT", "json"))
	defer func() { _ = logger.Sync() }()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("config", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("postgres", zap.Error(err))
	}
	defer pool.Close()

	if cfg.AutoMigrate {
		if err := runMigrations(pool); err != nil {
			logger.Fatal("migrate", zap.Error(err))
		}
	}

	cipher, err := crypto.NewTokenCipher(cfg.MasterEncryptionKey)
	if err != nil {
		logger.Fatal("cipher", zap.Error(err))
	}
	store := postgres.New(pool, logger, cipher)
	employeedir.Seed(ctx, store, logger)

	rdb := redis.NewClient(redisOpts(cfg.RedisURL))
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("redis", zap.Error(err))
	}

	queueClient, err := asynqqueue.NewClient(cfg.RedisURL, logger)
	if err != nil {
		logger.Fatal("asynq client", zap.Error(err))
	}
	defer queueClient.Close()

	var calendarConnector *google.CalendarConnector
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		calendarConnector = google.NewCalendarConnector(cfg.GoogleClientID, cfg.GoogleClientSecret)
	}
	var msConn *microsoft.CalendarConnector
	if cfg.MicrosoftClientID != "" && cfg.MicrosoftClientSecret != "" {
		msConn = microsoft.NewCalendarConnector(cfg.MicrosoftClientID, cfg.MicrosoftClientSecret)
	}
	var calProvider application.CalendarProvider
	if cfg.CalendarStub {
		calProvider = calendarstub.NewProvider()
	} else {
		gprov := calendargoogle.NewProvider(store, store, cipher, calendarConnector)
		var msFactory interface {
			For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool)
		}
		if msConn != nil {
			msFactory = calendarms.NewFactory(store, msConn)
		}
		calProvider = calendarresolver.New(store, gprov, msFactory)
	}
	services := &application.Services{Store: store, Cipher: cipher, Queue: queueClient, Calendar: calProvider, GoogleProber: calendargoogle.NewProber(), Log: logger}
	services.WireCQRS()

	sso := map[string]application.SSOProvider{}
	if cfg.GoogleClientID != "" {
		if p, err := google.New(context.Background(), cfg.GoogleClientID, cfg.GoogleClientSecret); err != nil {
			logger.Warn("google_oidc_init_failed", zap.Error(err))
		} else {
			sso["google"] = p
			logger.Info("web_auth_provider_configured", zap.String("provider", "google"))
		}
	}
	if cfg.MicrosoftClientID != "" {
		if p, err := microsoft.New(context.Background(), cfg.MicrosoftClientID, cfg.MicrosoftClientSecret); err != nil {
			logger.Warn("microsoft_oidc_init_failed", zap.Error(err))
		} else {
			sso["microsoft"] = p
			logger.Info("web_auth_provider_configured", zap.String("provider", "microsoft"))
		}
	}
	connectors := map[string]application.CalendarConnector{}
	if calendarConnector != nil {
		connectors["google"] = calendarConnector
	}
	if msConn != nil {
		connectors["microsoft"] = msConn
	}
	services.ConfigureCalendarConnectors(connectors)

	sender := smtpemail.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	if len(sso) == 0 && cfg.SMTPHost == "" {
		logger.Info("web_auth_disabled_missing_config")
	}
	services.ConfigureWebAuth(sso, sender, cfg.AppBaseURL, cfg.WebSessionTTL, cfg.MagicLinkTTL)

	var tgHandler *telegram.MultiHandler
	botOpts := []bot.Option{
		bot.WithDefaultHandler(func(c context.Context, b *bot.Bot, u *models.Update) {
			if tgHandler != nil {
				tgHandler.Handle(c, b, u)
			}
		}),
	}
	if cfg.AuthDevMode && !cfg.RealBotToken() {
		botOpts = append(botOpts, bot.WithSkipGetMe())
	}
	tg, err := bot.New(cfg.BotToken, botOpts...)
	if err != nil {
		logger.Fatal("telegram", zap.Error(err))
	}
	services.Bot = tg
	services.SetChatSyncer(func(ctx context.Context, workspaceID uuid.UUID) (int, error) {
		return telegram.SyncChatMembers(ctx, tg, store, workspaceID)
	})
	botUsername := "dev"
	if cfg.RealBotToken() {
		me, err := tg.GetMe(ctx)
		if err != nil {
			logger.Fatal("telegram getMe", zap.Error(err))
		}
		botUsername = me.Username
		planner := llmanthropic.New(os.Getenv("ANTHROPIC_API_KEY"))
		tgHandler = telegram.NewMultiHandler(store, tg, rdb, cfg.BotAdminTelegramIDs, cfg.WebappURL, services, planner, logger)
		if _, cerr := tg.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: telegram.PublicCommands()}); cerr != nil {
			logger.Warn("set_my_commands", zap.Error(cerr))
		}
	} else if cfg.AuthDevMode {
		logger.Warn("AUTH_DEV_MODE: telegram polling disabled (set BOT_TOKEN for /start and bot commands)")
	} else {
		logger.Fatal("telegram", zap.Error(fmt.Errorf("BOT_TOKEN is required")))
	}
	notifier := meeting_notifier.New(store, tg, logger)
	meetingCreatedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseMeetingCreated(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.OrganizationID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleCreated(c, wid, mid)
	}
	meetingUpdatedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseMeetingUpdated(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.OrganizationID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleUpdated(c, wid, mid)
	}
	participantAddedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseParticipant(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.OrganizationID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleParticipantAdded(c, wid, mid, p.Email)
	}
	participantRemovedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseParticipant(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.OrganizationID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleParticipantRemoved(c, wid, mid, p.Email)
	}
	meetingCancelledHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseMeetingCancelled(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.OrganizationID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleCancelled(c, wid, mid)
	}

	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, map[string]asynq.HandlerFunc{
		asynqqueue.TaskMeetingCreated:     meetingCreatedHandler,
		asynqqueue.TaskMeetingUpdated:     meetingUpdatedHandler,
		asynqqueue.TaskParticipantAdded:   participantAddedHandler,
		asynqqueue.TaskParticipantRemoved: participantRemovedHandler,
		asynqqueue.TaskMeetingCancelled:   meetingCancelledHandler,
	})
	if err != nil {
		logger.Fatal("asynq server", zap.Error(err))
	}
	go func() {
		if err := asynqSrv.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("asynq", zap.Error(err))
		}
	}()

	remSched := reminder_scheduler.New(store, tg, rdb, sender, cfg.ReminderEmailEnabled, cfg.WebappURL, logger)
	go remSched.Run(ctx)

	app, err := deliveryhttp.NewApp(cfg, store, cipher, rdb, tg, logger, services)
	if err != nil {
		logger.Fatal("http", zap.Error(err))
	}

	if tgHandler != nil {
		go tg.Start(ctx)
	}

	go func() {
		logger.Info("http listen", zap.String("addr", cfg.HTTPAddr))
		if err := app.Listen(cfg.HTTPAddr); err != nil {
			logger.Error("http", zap.Error(err))
		}
	}()

	logger.Info("lead-cat started", zap.String("bot", botUsername))
	<-ctx.Done()
	asynqSrv.Shutdown()
	_ = app.Shutdown()
}

func runMigrations(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "migrations")
}

func redisOpts(url string) *redis.Options {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return &redis.Options{Addr: "localhost:6379"}
	}
	return opt
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
