package application

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
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
	SendMultipart(ctx context.Context, to, subject, textBody, htmlBody, listUnsubscribe string) error
}

type CalendarToken = model.CalendarToken

type CalendarConnector interface {
	Name() string
	AuthURL(state, pkceChallenge, redirectURL string) string
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (CalendarToken, error)
}

type BusyReader = docalendar.BusyReader

type BusyResolver interface {
	ReaderFor(ctx context.Context, email string) (BusyReader, bool)
}
