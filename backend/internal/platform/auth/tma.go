package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tmaTokenType = "tma"

// TMAClaims is a Telegram Mini App session token. Distinct from TokenClaims:
// a TMA user is a bot_users row keyed by telegram_id, not a platform_users UUID.
type TMAClaims struct {
	TelegramID int64  `json:"tg_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	TokenType  string `json:"tok_typ"` // distinct payload key; not the JWT "typ" header
	jwt.RegisteredClaims
}

// TMAToken mints and verifies TMA session JWTs. Reuses the platform JWT secret.
type TMAToken struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTMAToken(secret, issuer string, ttl time.Duration) (*TMAToken, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &TMAToken{secret: []byte(secret), ttl: ttl, issuer: issuer}, nil
}

func (t *TMAToken) Issue(telegramID int64, email, role string) (string, error) {
	now := time.Now()
	claims := TMAClaims{
		TelegramID: telegramID,
		Email:      email,
		Role:       role,
		TokenType:  tmaTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *TMAToken) Parse(token string) (*TMAClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &TMAClaims{}, func(tok *jwt.Token) (any, error) {
		if tok.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*TMAClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.TokenType != tmaTokenType {
		return nil, fmt.Errorf("not a tma token")
	}
	return claims, nil
}
