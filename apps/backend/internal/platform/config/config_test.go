package config

import (
	"testing"
)

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"AUTH_DEV_MODE", "APP_ENV", "METRICS_TOKEN", "JWT_SECRET",
		"MASTER_ENCRYPTION_KEY", "DATABASE_URL", "BOT_TOKEN",
	} {
		t.Setenv(k, "")
	}
}

func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db?sslmode=disable")
	t.Setenv("MASTER_ENCRYPTION_KEY", "local-dev-encryption-key-min-16")
	t.Setenv("JWT_SECRET", "local-dev-jwt-secret-min-16")
	t.Setenv("APP_ENV", "development")
}

func TestLoad_RequiresJWTSecret(t *testing.T) {
	clearConfigEnv(t)
	baseEnv(t)
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing JWT_SECRET")
	}
}

func TestLoad_NoJWTFallbackToMasterKey(t *testing.T) {
	clearConfigEnv(t)
	baseEnv(t)
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error; JWT must not fall back to MASTER_ENCRYPTION_KEY")
	}
}

func TestLoad_ProductionRejectsAuthDevMode(t *testing.T) {
	clearConfigEnv(t)
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_DEV_MODE", "true")
	t.Setenv("METRICS_TOKEN", "prod-metrics-token")
	t.Setenv("BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for AUTH_DEV_MODE in production")
	}
}

func TestLoad_ProductionRequiresMetricsToken(t *testing.T) {
	clearConfigEnv(t)
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing METRICS_TOKEN in production")
	}
}

func TestLoad_ProductionOK(t *testing.T) {
	clearConfigEnv(t)
	baseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("METRICS_TOKEN", "prod-metrics-token")
	t.Setenv("BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")
	t.Setenv("REDIS_URL", "redis://redis.prod.example:6379/0")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://admin.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.IsProduction() {
		t.Fatal("expected production")
	}
	if cfg.JWTSecret != "local-dev-jwt-secret-min-16" {
		t.Fatalf("unexpected JWT secret: %q", cfg.JWTSecret)
	}
}

func TestLoad_DevAuthModeAllowed(t *testing.T) {
	clearConfigEnv(t)
	baseEnv(t)
	t.Setenv("AUTH_DEV_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.AuthDevMode {
		t.Fatal("expected AuthDevMode")
	}
}

func TestIsProduction(t *testing.T) {
	if !(Config{AppEnv: "production"}).IsProduction() {
		t.Fatal("production")
	}
	if (Config{AppEnv: "development"}).IsProduction() {
		t.Fatal("development")
	}
}

func TestIsDevelopment(t *testing.T) {
	if !(Config{AppEnv: "development"}).IsDevelopment() {
		t.Fatal("development")
	}
	if (Config{AppEnv: "staging"}).IsDevelopment() {
		t.Fatal("staging is not development")
	}
}
