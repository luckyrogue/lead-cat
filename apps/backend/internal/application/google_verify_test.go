package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type googleVerifyRepo struct {
	unimplementedRepo
	enc     []byte
	subject string
	calID   string
}

func (s *googleVerifyRepo) GetGoogleConfig(context.Context, uuid.UUID) ([]byte, string, string, error) {
	return s.enc, s.subject, s.calID, nil
}

type passthroughCipher struct{}

func (passthroughCipher) Encrypt(plain string) ([]byte, error) { return []byte(plain), nil }
func (passthroughCipher) Decrypt(enc []byte) (string, error)  { return string(enc), nil }

type fakeProber struct {
	res docalendar.ProbeResult
	err error
}

func (p *fakeProber) Probe(context.Context, string, string, string) (docalendar.ProbeResult, error) {
	return p.res, p.err
}

func TestVerifyGoogleIntegration_Success(t *testing.T) {
	t.Parallel()
	orgID := uuid.New()
	svc := &Services{
		Store:        &googleVerifyRepo{enc: []byte(`{"k":1}`), subject: "admin@corp.io", calID: "primary"},
		Cipher:       passthroughCipher{},
		GoogleProber: &fakeProber{res: docalendar.ProbeResult{Summary: "Team", TimeZone: "Asia/Almaty"}},
	}
	got, err := svc.VerifyGoogleIntegration(context.Background(), orgID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.OK || got.CalendarSummary != "Team" || got.TimeZone != "Asia/Almaty" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestVerifyGoogleIntegration_NotConfigured(t *testing.T) {
	t.Parallel()
	svc := &Services{Store: &googleVerifyRepo{}}
	_, err := svc.VerifyGoogleIntegration(context.Background(), uuid.New())
	if !errors.Is(err, ErrGoogleNotConfigured) {
		t.Fatalf("want ErrGoogleNotConfigured, got %v", err)
	}
}

func TestVerifyGoogleIntegration_ProbeFailure(t *testing.T) {
	t.Parallel()
	svc := &Services{
		Store:        &googleVerifyRepo{enc: []byte("sa"), subject: "a@b.c"},
		Cipher:       passthroughCipher{},
		GoogleProber: &fakeProber{err: docalendar.ErrProbeSubject},
	}
	_, err := svc.VerifyGoogleIntegration(context.Background(), uuid.New())
	if !errors.Is(err, docalendar.ErrProbeSubject) {
		t.Fatalf("want ErrProbeSubject, got %v", err)
	}
}
