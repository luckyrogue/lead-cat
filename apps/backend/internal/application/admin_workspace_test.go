package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeOrgStore struct {
	id        uuid.UUID
	createErr error
	createdTZ string
	createdML string
	created   bool
}

func (f *fakeOrgStore) EnsureDefaultOrganizationID(_ context.Context, tz, ml string, _ uuid.UUID) (uuid.UUID, error) {
	if f.createErr != nil {
		return uuid.Nil, f.createErr
	}
	f.createdTZ = tz
	f.createdML = ml
	f.created = true
	return f.id, nil
}

func TestEnsureDefaultOrganization_Defaults(t *testing.T) {
	want := uuid.New()
	f := &fakeOrgStore{id: want}
	got, err := ensureDefaultOrganization(context.Background(), f, uuid.New())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != want {
		t.Fatalf("id mismatch")
	}
	if f.createdTZ != "Asia/Almaty" {
		t.Fatalf("default tz wrong: %q", f.createdTZ)
	}
	if f.createdML != "" {
		t.Fatalf("default meet link should be empty: %q", f.createdML)
	}
}

func TestEnsureDefaultOrganization_PropagatesStoreError(t *testing.T) {
	boom := errors.New("db down")
	f := &fakeOrgStore{createErr: boom}
	if _, err := ensureDefaultOrganization(context.Background(), f, uuid.New()); !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
}
