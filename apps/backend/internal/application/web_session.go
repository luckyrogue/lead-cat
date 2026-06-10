package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

// shouldSlide reports whether a session past its half-life should have its
// expiry extended (sliding-window renewal).
func shouldSlide(created, _ time.Time, now time.Time, ttl time.Duration) bool {
	return now.Sub(created) > ttl/2
}

type webSessionRepo interface {
	CreateWebSession(ctx context.Context, tokenHash []byte, userID uuid.UUID, expiresAt time.Time, ua, ip string) (model.WebSession, error)
	ResolveWebSession(ctx context.Context, tokenHash []byte, now time.Time) (model.WebSession, bool, error)
	TouchWebSession(ctx context.Context, id uuid.UUID, lastSeen, expiresAt time.Time) error
	RevokeWebSession(ctx context.Context, tokenHash []byte, now time.Time) error
}

type webSessionService struct {
	repo  webSessionRepo
	ttl   time.Duration
	clock func() time.Time
}

func newWebSessionService(repo webSessionRepo, ttl time.Duration, clock func() time.Time) *webSessionService {
	if clock == nil {
		clock = time.Now
	}
	return &webSessionService{repo: repo, ttl: ttl, clock: clock}
}

// CreateSession mints a raw session token (returned to set as cookie) and stores only its hash.
func (s *webSessionService) CreateSession(ctx context.Context, userID uuid.UUID, ua, ip string) (rawToken string, err error) {
	raw, err := authweb.NewState(nil)
	if err != nil {
		return "", err
	}
	now := s.clock()
	if _, err := s.repo.CreateWebSession(ctx, authweb.HashToken(raw), userID, now.Add(s.ttl), ua, ip); err != nil {
		return "", err
	}
	return raw, nil
}

// ResolveSession returns the platform_users id for a valid session cookie, sliding expiry if past half-life.
func (s *webSessionService) ResolveSession(ctx context.Context, rawToken string) (model.WebSession, bool, error) {
	now := s.clock()
	w, ok, err := s.repo.ResolveWebSession(ctx, authweb.HashToken(rawToken), now)
	if err != nil || !ok {
		return model.WebSession{}, false, err
	}
	if shouldSlide(w.CreatedAt, w.ExpiresAt, now, s.ttl) {
		_ = s.repo.TouchWebSession(ctx, w.ID, now, now.Add(s.ttl))
	}
	return w, true, nil
}

// RevokeSession invalidates the session associated with the raw token.
func (s *webSessionService) RevokeSession(ctx context.Context, rawToken string) error {
	return s.repo.RevokeWebSession(ctx, authweb.HashToken(rawToken), s.clock())
}
