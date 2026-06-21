package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const miniappTokenType = "miniapp"

type MiniAppClaims struct {
	TelegramID int64  `json:"tg_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	TokenType  string `json:"tok_typ"`
	jwt.RegisteredClaims
}

type MiniAppToken struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewMiniAppToken(secret, issuer string, ttl time.Duration) (*MiniAppToken, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &MiniAppToken{secret: []byte(secret), ttl: ttl, issuer: issuer}, nil
}

func (t *MiniAppToken) Issue(telegramID int64, email, role string) (string, error) {
	now := time.Now()
	claims := MiniAppClaims{
		TelegramID: telegramID,
		Email:      email,
		Role:       role,
		TokenType:  miniappTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *MiniAppToken) Parse(token string) (*MiniAppClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &MiniAppClaims{}, func(tok *jwt.Token) (any, error) {
		if tok.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*MiniAppClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.TokenType != miniappTokenType {
		return nil, fmt.Errorf("not a miniapp token")
	}
	if claims.Issuer != t.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	return claims, nil
}
