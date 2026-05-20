package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenClaims struct {
	UserID  string `json:"uid"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
	AuthSub string `json:"sub"`
	jwt.RegisteredClaims
}

type JWT struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewJWT(secret, issuer string, ttl time.Duration) (*JWT, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &JWT{secret: []byte(secret), ttl: ttl, issuer: issuer}, nil
}

func (j *JWT) Issue(userID uuid.UUID, authSub, email, phone string) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID:  userID.String(),
		Email:   email,
		Phone:   phone,
		AuthSub: authSub,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   authSub,
			Issuer:    j.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.secret)
}

func (j *JWT) Parse(token string) (*TokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &TokenClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*TokenClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
