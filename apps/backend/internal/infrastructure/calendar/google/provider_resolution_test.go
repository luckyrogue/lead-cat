package google

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	oauthgoogle "github.com/luckyrogue/lead-cat/internal/infrastructure/oauth/google"
)

type fakeConnStore struct {
	conn    *model.CalendarConnection
	upserts int
}

func (f *fakeConnStore) GetCalendarConnection(_ context.Context, _, _ string) (model.CalendarConnection, error) {
	if f.conn == nil {
		return model.CalendarConnection{}, sql.ErrNoRows
	}
	return *f.conn, nil
}

func (f *fakeConnStore) UpsertCalendarConnection(_ context.Context, _ model.CalendarConnection) error {
	f.upserts++
	return nil
}

type emptyConfigStore struct{}

func (emptyConfigStore) GetGoogleConfig(_ context.Context, _ uuid.UUID) ([]byte, string, string, error) {
	return nil, "", "", nil
}

func testCipher(t *testing.T) *crypto.TokenCipher {
	t.Helper()
	c, err := crypto.NewTokenCipher("test-master-key")
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	return c
}

func TestFor_NoConnection_FallsBackToSA(t *testing.T) {
	p := NewProvider(&fakeConnStore{conn: nil}, emptyConfigStore{}, testCipher(t), &oauthgoogle.CalendarConnector{})
	_, err := p.For(context.Background(), uuid.New(), "nobody@example.com")
	if !errors.Is(err, docalendar.ErrNotConfigured) {
		t.Fatalf("expected SA fallback (ErrNotConfigured), got %v", err)
	}
}

func TestFor_WithConnection_BuildsUserService(t *testing.T) {
	conn := &model.CalendarConnection{Email: "a@x.com", Provider: "google", AccessToken: "at", RefreshToken: "rt", Expiry: time.Now().Add(time.Hour)}
	p := NewProvider(&fakeConnStore{conn: conn}, emptyConfigStore{}, testCipher(t), &oauthgoogle.CalendarConnector{})
	svc, err := p.For(context.Background(), uuid.New(), "a@x.com")
	if err != nil || svc == nil {
		t.Fatalf("expected per-user service, got svc=%v err=%v", svc, err)
	}
}
