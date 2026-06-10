package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

// ErrInvalidMagicLink is returned when a magic-link token is unknown, expired,
// or already consumed.
var ErrInvalidMagicLink = errors.New("invalid or expired magic link")

// magicLinkRepo is the persistence port for magic-link tokens.
type magicLinkRepo interface {
	InsertMagicLink(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) error
	ConsumeMagicLink(ctx context.Context, tokenHash []byte, now time.Time) (string, bool, error)
}

// magicLinkService handles issuing and verifying one-time sign-in links.
type magicLinkService struct {
	repo    magicLinkRepo
	mailer  EmailSender
	baseURL string
	ttl     time.Duration
	clock   func() time.Time
}

// newMagicLinkService constructs a magicLinkService. clock may be nil (defaults to time.Now).
func newMagicLinkService(repo magicLinkRepo, mailer EmailSender, baseURL string, ttl time.Duration, clock func() time.Time) *magicLinkService {
	if clock == nil {
		clock = time.Now
	}
	return &magicLinkService{repo: repo, mailer: mailer, baseURL: baseURL, ttl: ttl, clock: clock}
}

// RequestMagicLink generates a one-time token, stores its hash, and emails the
// raw token to the given address.
func (s *magicLinkService) RequestMagicLink(ctx context.Context, email string) error {
	raw, err := authweb.NewState(nil)
	if err != nil {
		return err
	}
	if err := s.repo.InsertMagicLink(ctx, email, authweb.HashToken(raw), s.clock().Add(s.ttl)); err != nil {
		return err
	}
	link := fmt.Sprintf("%s/api/auth/web/magic/verify?token=%s", s.baseURL, raw)
	body := fmt.Sprintf(`<p>Sign in to Lead Cat:</p><p><a href="%s">%s</a></p><p>This link expires in %d minutes.</p>`,
		link, link, int(s.ttl.Minutes()))
	return s.mailer.Send(ctx, email, "Your Lead Cat sign-in link", body)
}

// VerifyMagicLink consumes the token and returns the associated email address.
// Returns ErrInvalidMagicLink if the token is unknown, expired, or already used.
func (s *magicLinkService) VerifyMagicLink(ctx context.Context, rawToken string) (string, error) {
	email, ok, err := s.repo.ConsumeMagicLink(ctx, authweb.HashToken(rawToken), s.clock())
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidMagicLink
	}
	return email, nil
}
