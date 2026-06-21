package auth_test

import (
	"testing"
	"time"

	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
)

func TestMiniAppToken_IssueParse(t *testing.T) {
	tok, err := platformauth.NewMiniAppToken("test-jwt-secret-min-16", "lead-cat", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tok.Issue(42, "a@test.com", "user")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tok.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TelegramID != 42 || claims.Email != "a@test.com" || claims.Role != "user" {
		t.Fatalf("claims: %+v", claims)
	}
}

func TestMiniAppToken_WrongIssuerRejected(t *testing.T) {
	tok, err := platformauth.NewMiniAppToken("test-jwt-secret-min-16", "lead-cat", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	other, err := platformauth.NewMiniAppToken("test-jwt-secret-min-16", "other-issuer", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tok.Issue(1, "a@test.com", "user")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Parse(raw); err == nil {
		t.Fatal("expected issuer mismatch error")
	}
}

func TestMiniAppToken_ExpiredRejected(t *testing.T) {
	tok, err := platformauth.NewMiniAppToken("test-jwt-secret-min-16", "lead-cat", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := tok.Issue(1, "a@test.com", "user")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	if _, err := tok.Parse(raw); err == nil {
		t.Fatal("expected expiry error")
	}
}
