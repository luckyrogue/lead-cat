package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }

func TestShouldSlideWhenPastHalfLife(t *testing.T) {
	ttl := 720 * time.Hour
	created := mustTime("2026-06-01T00:00:00Z")
	expires := created.Add(ttl)
	if shouldSlide(created, expires, created.Add(100*time.Hour), ttl) {
		t.Fatal("slid too early (before half-life)")
	}
	if !shouldSlide(created, expires, created.Add(400*time.Hour), ttl) {
		t.Fatal("should slide past half-life")
	}
}

type fakeWebSessionRepo struct {
	sessions map[string]model.WebSession
}

func newFakeWebSessionRepo() *fakeWebSessionRepo {
	return &fakeWebSessionRepo{sessions: map[string]model.WebSession{}}
}

func (f *fakeWebSessionRepo) CreateWebSession(_ context.Context, tokenHash []byte, userID uuid.UUID, expiresAt time.Time, ua, ip string) (model.WebSession, error) {
	w := model.WebSession{
		ID: uuid.New(), TokenHash: tokenHash, UserID: userID,
		CreatedAt: time.Now(), LastSeenAt: time.Now(), ExpiresAt: expiresAt,
		UserAgent: ua, IP: ip,
	}
	f.sessions[string(tokenHash)] = w
	return w, nil
}

func (f *fakeWebSessionRepo) ResolveWebSession(_ context.Context, tokenHash []byte, now time.Time) (model.WebSession, bool, error) {
	w, ok := f.sessions[string(tokenHash)]
	if !ok || w.RevokedAt != nil || !w.ExpiresAt.After(now) {
		return model.WebSession{}, false, nil
	}
	return w, true, nil
}

func (f *fakeWebSessionRepo) TouchWebSession(_ context.Context, id uuid.UUID, lastSeen, expiresAt time.Time) error {
	for k, w := range f.sessions {
		if w.ID == id {
			w.LastSeenAt = lastSeen
			w.ExpiresAt = expiresAt
			f.sessions[k] = w
		}
	}
	return nil
}

func (f *fakeWebSessionRepo) RevokeWebSession(_ context.Context, tokenHash []byte, now time.Time) error {
	if w, ok := f.sessions[string(tokenHash)]; ok {
		w.RevokedAt = &now
		f.sessions[string(tokenHash)] = w
	}
	return nil
}

func TestWebSessionServiceCreateResolveRevoke(t *testing.T) {
	repo := newFakeWebSessionRepo()
	svc := newWebSessionService(repo, 720*time.Hour, nil)

	raw, err := svc.CreateSession(context.Background(), uuid.New(), "ua", "127.0.0.1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	w, ok, err := svc.ResolveSession(context.Background(), raw)
	if err != nil || !ok {
		t.Fatalf("resolve: ok=%v err=%v", ok, err)
	}
	if w.IP != "127.0.0.1" {
		t.Fatalf("ip mismatch: %q", w.IP)
	}

	if err := svc.RevokeSession(context.Background(), raw); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	_, ok, err = svc.ResolveSession(context.Background(), raw)
	if err != nil || ok {
		t.Fatalf("expected session gone after revoke: ok=%v err=%v", ok, err)
	}
}
