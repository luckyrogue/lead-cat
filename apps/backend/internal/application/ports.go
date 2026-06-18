package application

import (
	"context"
	"time"
)

const (
	AuthMethodGoogle    = "google"
	AuthMethodMicrosoft = "microsoft"
	AuthMethodMagicLink = "magiclink"
)

type SSOProfile struct {
	Email     string
	Name      string
	AvatarURL string
	Subject   string
	Provider  string
}

type SSOProvider interface {
	Name() string
	AuthURL(state, pkceChallenge, redirectURL string) string
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (SSOProfile, error)
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

type CalendarToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Scopes       string
}

type CalendarConnector interface {
	Name() string
	AuthURL(state, pkceChallenge, redirectURL string) string
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (CalendarToken, error)
}
