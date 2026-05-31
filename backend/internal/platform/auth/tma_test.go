package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTMAToken_RoundTrip(t *testing.T) {
	tok, err := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	s, err := tok.Issue(42, "a@b.kz", "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := tok.Parse(s)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.TelegramID != 42 || claims.Email != "a@b.kz" || claims.Role != "admin" || claims.TokenType != "tma" {
		t.Fatalf("bad claims: %+v", claims)
	}
}

func TestTMAToken_RejectsNonTMAType(t *testing.T) {
	tok, _ := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	bad := jwt.NewWithClaims(jwt.SigningMethodHS256, TMAClaims{TelegramID: 1, TokenType: "platform"})
	s, _ := bad.SignedString(tok.secret)
	if _, err := tok.Parse(s); err == nil {
		t.Fatal("expected rejection of non-tma typ")
	}
}

func TestTMAToken_RejectsWrongSecret(t *testing.T) {
	a, _ := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	b, _ := NewTMAToken("ffffffffffffffffffff", "lead-cat", time.Hour)
	s, _ := a.Issue(1, "x@y.kz", "user")
	if _, err := b.Parse(s); err == nil {
		t.Fatal("expected rejection of wrong-secret token")
	}
}

func TestTMAToken_RejectsExpired(t *testing.T) {
	tok, _ := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, TMAClaims{
		TelegramID: 1, TokenType: "tma",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	})
	s, _ := expired.SignedString(tok.secret)
	if _, err := tok.Parse(s); err == nil {
		t.Fatal("expected rejection of expired token")
	}
}

func TestNewTMAToken_ShortSecret(t *testing.T) {
	if _, err := NewTMAToken("short", "lead-cat", time.Hour); err == nil {
		t.Fatal("expected error for short secret")
	}
}
