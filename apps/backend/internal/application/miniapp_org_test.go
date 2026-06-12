package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeMiniAppOrgStore struct {
	orgID   uuid.UUID
	enc     []byte
	subject string
	orgErr  error
	cfgErr  error
}

func (f *fakeMiniAppOrgStore) EnsureDefaultOrganizationID(_ context.Context, _, _ string, _ uuid.UUID) (uuid.UUID, error) {
	if f.orgErr != nil {
		return uuid.Nil, f.orgErr
	}
	return f.orgID, nil
}

func (f *fakeMiniAppOrgStore) GetGoogleConfig(_ context.Context, _ uuid.UUID) ([]byte, string, string, error) {
	if f.cfgErr != nil {
		return nil, "", "", f.cfgErr
	}
	return f.enc, f.subject, "", nil
}

func TestResolveMiniAppOrganization_RequiresGoogle(t *testing.T) {
	t.Parallel()
	_, err := resolveMiniAppOrganization(context.Background(), &fakeMiniAppOrgStore{orgID: uuid.New()})
	if !errors.Is(err, ErrGoogleNotConfigured) {
		t.Fatalf("expected ErrGoogleNotConfigured, got %v", err)
	}
}

func TestResolveMiniAppOrganization_OK(t *testing.T) {
	t.Parallel()
	want := uuid.New()
	got, err := resolveMiniAppOrganization(context.Background(), &fakeMiniAppOrgStore{
		orgID: want, enc: []byte("enc"), subject: "svc@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
