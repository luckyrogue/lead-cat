package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
	"github.com/luckyrogue/lead-cat/internal/platform/emailtemplates"
)

var ErrInvalidMagicLink = errors.New("invalid or expired magic link")

type magicLinkRepo interface {
	InsertMagicLink(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) error
	ConsumeMagicLink(ctx context.Context, tokenHash []byte, now time.Time) (string, bool, error)
}

type magicLinkService struct {
	repo       magicLinkRepo
	mailer     EmailSender
	appBaseURL string
	webappURL  string
	ttl        time.Duration
	clock      func() time.Time
}

func newMagicLinkService(repo magicLinkRepo, mailer EmailSender, appBaseURL, webappURL string, ttl time.Duration, clock func() time.Time) *magicLinkService {
	if clock == nil {
		clock = time.Now
	}
	return &magicLinkService{repo: repo, mailer: mailer, appBaseURL: appBaseURL, webappURL: webappURL, ttl: ttl, clock: clock}
}

func (s *magicLinkService) magicLinkURL(raw string) string {
	base := strings.TrimRight(s.webappURL, "/")
	if base == "" {
		base = strings.TrimRight(s.appBaseURL, "/")
	}
	return fmt.Sprintf("%s/auth/magic?token=%s", base, raw)
}

func (s *magicLinkService) RequestMagicLink(ctx context.Context, email, language string) error {
	raw, err := authweb.NewState(nil)
	if err != nil {
		return err
	}
	if err := s.repo.InsertMagicLink(ctx, email, authweb.HashToken(raw), s.clock().Add(s.ttl)); err != nil {
		return err
	}
	link := s.magicLinkURL(raw)
	subject, text, html, err := emailtemplates.RenderMagicLink(emailtemplates.MagicLinkData{
		Language:       language,
		SignInURL:      link,
		ExpiresMinutes: int(s.ttl.Minutes()),
	})
	if err != nil {
		return err
	}
	return s.mailer.SendMultipart(ctx, email, subject, text, html, "")
}

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
