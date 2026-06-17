package google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type fakeConfigStore struct {
	enc        []byte
	subject    string
	calendarID string
	err        error
}

func (f *fakeConfigStore) GetGoogleConfig(_ context.Context, _ uuid.UUID) ([]byte, string, string, error) {
	return f.enc, f.subject, f.calendarID, f.err
}

var _ configStore = (*fakeConfigStore)(nil)

func TestFor_ErrNotConfigured(t *testing.T) {
	for name, fcs := range map[string]*fakeConfigStore{
		"empty-enc":     {enc: nil, subject: "s@x.io"},
		"empty-subject": {enc: []byte("x"), subject: ""},
	} {
		t.Run(name, func(t *testing.T) {
			p := NewProvider(fcs, nil)
			if _, err := p.For(context.Background(), uuid.New()); err != docalendar.ErrNotConfigured {
				t.Fatalf("want ErrNotConfigured, got %v", err)
			}
		})
	}
}

func TestFor_CacheHit_DefaultsCalendarID(t *testing.T) {
	org := uuid.New()
	enc := []byte("encrypted-bytes")
	subject := "svc@x.io"
	fcs := &fakeConfigStore{enc: enc, subject: subject, calendarID: ""} // blank -> defaults to "primary"
	p := NewProvider(fcs, nil)

	sum := sha256.Sum256(enc)
	key := org.String() + "|" + subject + "|primary|" + hex.EncodeToString(sum[:])
	want := &adapter{calendarID: "primary"}
	p.cache.Store(key, want)

	got, err := p.For(context.Background(), org)
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	if got != docalendar.Service(want) {
		t.Fatalf("cache hit should return the seeded adapter; got %#v", got)
	}
}
