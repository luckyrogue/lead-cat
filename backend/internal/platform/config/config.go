package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	BotToken            string
	DatabaseURL         string
	RedisURL            string
	MasterEncryptionKey string
	HTTPAddr            string
	WebappURL           string
	LogLevel            string
	LogFormat           string
	AutoMigrate         bool
	CalendarStub        bool
	StaticDir           string

	AuthDevMode bool

	JWTSecret string
	JWTIssuer string
	JWTTTL    time.Duration

	CORSAllowedOrigins string

	BotAdminTelegramIDs []int64
}

func Load() (Config, error) {
	cfg := Config{
		BotToken:            strings.TrimSpace(os.Getenv("BOT_TOKEN")),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		RedisURL:            envOr("REDIS_URL", "redis://localhost:6379/0"),
		MasterEncryptionKey: strings.TrimSpace(os.Getenv("MASTER_ENCRYPTION_KEY")),
		HTTPAddr:            envOr("HTTP_ADDR", ":8080"),
		WebappURL:           strings.TrimSpace(os.Getenv("WEBAPP_URL")),
		LogLevel:            envOr("LOG_LEVEL", "info"),
		LogFormat:           envOr("LOG_FORMAT", "json"),
		StaticDir:           envOr("STATIC_DIR", "frontend/dist"),
		JWTSecret:           strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTIssuer:           envOr("JWT_ISSUER", "lead-cat"),
		CORSAllowedOrigins:  envOr("CORS_ALLOWED_ORIGINS", ""),
	}
	if cfg.CORSAllowedOrigins == "" && cfg.WebappURL != "" {
		cfg.CORSAllowedOrigins = cfg.WebappURL
	}
	if cfg.CORSAllowedOrigins == "" {
		cfg.CORSAllowedOrigins = "http://localhost:3000,http://localhost:8080"
	}
	cfg.AutoMigrate = strings.EqualFold(os.Getenv("AUTO_MIGRATE"), "true")
	cfg.CalendarStub = strings.EqualFold(os.Getenv("CALENDAR_STUB"), "true")
	for _, p := range strings.Split(os.Getenv("BOT_ADMIN_TELEGRAM_IDS"), ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			cfg.BotAdminTelegramIDs = append(cfg.BotAdminTelegramIDs, id)
		}
	}
	cfg.AuthDevMode = strings.EqualFold(os.Getenv("AUTH_DEV_MODE"), "true")

	ttlHours := 168
	if v := strings.TrimSpace(os.Getenv("JWT_TTL_HOURS")); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &ttlHours); err != nil {
			return cfg, fmt.Errorf("invalid JWT_TTL_HOURS")
		}
	}
	cfg.JWTTTL = time.Duration(ttlHours) * time.Hour

	if cfg.BotToken == "" {
		if cfg.AuthDevMode {
			cfg.BotToken = fakeDevBotToken
		} else {
			return cfg, fmt.Errorf("BOT_TOKEN is required")
		}
	}
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.MasterEncryptionKey) < 16 {
		return cfg, fmt.Errorf("MASTER_ENCRYPTION_KEY must be at least 16 characters")
	}
	if !cfg.AuthDevMode && len(cfg.JWTSecret) < 16 {
		return cfg, fmt.Errorf("JWT_SECRET must be at least 16 characters unless AUTH_DEV_MODE=true")
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = cfg.MasterEncryptionKey
	}
	return cfg, nil
}

const fakeDevBotToken = "000000000:AAFakeDevTokenForLocalOnly"

// RealBotToken is true when BOT_TOKEN is a non-placeholder value (bot polling can run).
func (c Config) RealBotToken() bool {
	return c.BotToken != "" && c.BotToken != fakeDevBotToken
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
