package main

import (
	"context"
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

	deliveryhttp "github.com/Jaryq-Lab/notify-bot/internal/delivery/http"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	asynqqueue "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/queue/asynq"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/telegram"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/config"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/log"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/scenario_executor"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/scenario_scheduler"
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

	store := postgres.New(pool, logger)
	cipher, err := crypto.NewTokenCipher(cfg.MasterEncryptionKey)
	if err != nil {
		logger.Fatal("cipher", zap.Error(err))
	}

	rdb := redis.NewClient(redisOpts(cfg.RedisURL))
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("redis", zap.Error(err))
	}

	queueClient, err := asynqqueue.NewClient(cfg.RedisURL, logger)
	if err != nil {
		logger.Fatal("asynq client", zap.Error(err))
	}
	defer queueClient.Close()

	var tgHandler *telegram.MultiHandler
	botOpts := []bot.Option{
		bot.WithDefaultHandler(func(c context.Context, b *bot.Bot, u *models.Update) {
			if tgHandler != nil {
				tgHandler.Handle(c, b, u)
			}
		}),
	}
	// In AUTH_DEV_MODE the bot token may be empty (API-only local dev); bot.New
	// otherwise calls getMe on init and fails. Polling stays disabled below.
	if cfg.AuthDevMode {
		botOpts = append(botOpts, bot.WithSkipGetMe())
	}
	tg, err := bot.New(cfg.BotToken, botOpts...)
	if err != nil {
		logger.Fatal("telegram", zap.Error(err))
	}
	botUsername := "dev"
	if cfg.AuthDevMode {
		logger.Warn("AUTH_DEV_MODE: telegram polling disabled; set real BOT_TOKEN for bot features")
	} else {
		me, err := tg.GetMe(ctx)
		if err != nil {
			logger.Fatal("telegram getMe", zap.Error(err))
		}
		botUsername = me.Username
		tgHandler = telegram.NewMultiHandler(store, cipher, tg, rdb, cfg.BotAdminTelegramIDs, cfg.AuthOTPLog, logger)
	}
	exec := scenario_executor.New(store, cipher, tg, logger)

	asynqHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParsePayload(t)
		if err != nil {
			return err
		}
		runID, _ := uuid.Parse(p.RunID)
		scID, _ := uuid.Parse(p.ScenarioID)
		return exec.Run(c, runID, scID, p.Trigger)
	}

	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, asynqHandler)
	if err != nil {
		logger.Fatal("asynq server", zap.Error(err))
	}
	go func() {
		if err := asynqSrv.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("asynq", zap.Error(err))
		}
	}()

	sched := scenario_scheduler.New(store, queueClient, rdb, logger)
	go sched.Run(ctx)

	app, err := deliveryhttp.NewApp(cfg, store, cipher, queueClient, rdb, tg, logger)
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
