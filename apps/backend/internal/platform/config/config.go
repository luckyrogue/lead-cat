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

	AuthDevMode       bool
	TrustProxyHeaders bool
	RateLimitDisabled bool
	AppEnv            string
	MetricsToken      string

	JWTSecret string
	JWTIssuer string
	JWTTTL    time.Duration

	CORSAllowedOrigins string

	BotAdminTelegramIDs []int64

	AppBaseURL      string
	WebCookieDomain string
	WebSessionTTL   time.Duration
	MagicLinkTTL    time.Duration

	GoogleClientID        string
	GoogleClientSecret    string
	MicrosoftClientID     string
	MicrosoftClientSecret string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	ReminderEmailEnabled bool
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

		AppBaseURL:      envOr("APP_BASE_URL", "http://localhost:8080"),
		WebCookieDomain: envOr("WEB_COOKIE_DOMAIN", ""),

		GoogleClientID:        strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_ID")),
		GoogleClientSecret:    strings.TrimSpace(os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")),
		MicrosoftClientID:     strings.TrimSpace(os.Getenv("MICROSOFT_OAUTH_CLIENT_ID")),
		MicrosoftClientSecret: strings.TrimSpace(os.Getenv("MICROSOFT_OAUTH_CLIENT_SECRET")),

		SMTPHost:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:     envOr("SMTP_PORT", "587"),
		SMTPUsername: strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		SMTPFrom:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
	if cfg.CORSAllowedOrigins == "" && cfg.WebappURL != "" {
		cfg.CORSAllowedOrigins = cfg.WebappURL
	}
	if cfg.CORSAllowedOrigins == "" {
		if cfg.IsProduction() {
			return cfg, fmt.Errorf("CORS_ALLOWED_ORIGINS is required when APP_ENV=production")
		}
		cfg.CORSAllowedOrigins = "http://localhost:3000,http://localhost:8080"
	}
	cfg.AutoMigrate = strings.EqualFold(os.Getenv("AUTO_MIGRATE"), "true")
	cfg.CalendarStub = strings.EqualFold(os.Getenv("CALENDAR_STUB"), "true")
	cfg.ReminderEmailEnabled = strings.EqualFold(os.Getenv("REMINDER_EMAIL_ENABLED"), "true")
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
	cfg.RateLimitDisabled = strings.EqualFold(os.Getenv("RATE_LIMIT_DISABLED"), "true")
	cfg.TrustProxyHeaders = strings.EqualFold(os.Getenv("TRUST_PROXY_HEADERS"), "true")
	cfg.AppEnv = envOr("APP_ENV", "development")
	cfg.MetricsToken = strings.TrimSpace(os.Getenv("METRICS_TOKEN"))

	ttlHours := 168
	if v := strings.TrimSpace(os.Getenv("JWT_TTL_HOURS")); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &ttlHours); err != nil {
			return cfg, fmt.Errorf("invalid JWT_TTL_HOURS")
		}
	}
	cfg.JWTTTL = time.Duration(ttlHours) * time.Hour

	sessionHours := 720
	if v := strings.TrimSpace(os.Getenv("WEB_SESSION_TTL_HOURS")); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &sessionHours); err != nil {
			return cfg, fmt.Errorf("invalid WEB_SESSION_TTL_HOURS")
		}
	}
	cfg.WebSessionTTL = time.Duration(sessionHours) * time.Hour

	magicMinutes := 15
	if v := strings.TrimSpace(os.Getenv("MAGIC_LINK_TTL_MINUTES")); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &magicMinutes); err != nil {
			return cfg, fmt.Errorf("invalid MAGIC_LINK_TTL_MINUTES")
		}
	}
	cfg.MagicLinkTTL = time.Duration(magicMinutes) * time.Minute

	if cfg.IsProduction() && strings.Contains(cfg.RedisURL, "localhost") {
		return cfg, fmt.Errorf("REDIS_URL must not point to localhost when APP_ENV=production")
	}
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
	if cfg.JWTSecret == "" {
		return cfg, fmt.Errorf("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 16 {
		return cfg, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if cfg.IsProduction() {
		if cfg.AuthDevMode {
			return cfg, fmt.Errorf("AUTH_DEV_MODE must not be set when APP_ENV=production")
		}
		if cfg.RateLimitDisabled {
			return cfg, fmt.Errorf("RATE_LIMIT_DISABLED must not be set when APP_ENV=production")
		}
		if cfg.MetricsToken == "" {
			return cfg, fmt.Errorf("METRICS_TOKEN is required when APP_ENV=production")
		}
	}
	return cfg, nil
}

func (c Config) IsProduction() bool {
	return strings.EqualFold(c.AppEnv, "production")
}

func (c Config) IsDevelopment() bool {
	return strings.EqualFold(c.AppEnv, "development")
}

const fakeDevBotToken = "000000000:AAFakeDevTokenForLocalOnly"

func (c Config) RealBotToken() bool {
	return c.BotToken != "" && c.BotToken != fakeDevBotToken
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
