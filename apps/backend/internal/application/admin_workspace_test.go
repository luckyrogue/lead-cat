package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeOrgStore struct {
	id  uuid.UUID
	err error
}

func (f *fakeOrgStore) EnsureDefaultOrganizationID(_ context.Context, tz, ml string, _ uuid.UUID) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	return f.id, nil
}

func TestEnsureDefaultOrganization_Defaults(t *testing.T) {
	t.Parallel()
	want := uuid.New()
	got, err := ensureDefaultOrganization(context.Background(), &fakeOrgStore{id: want}, uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestEnsureDefaultOrganization_PropagatesStoreError(t *testing.T) {
	t.Parallel()
	_, err := ensureDefaultOrganization(context.Background(), &fakeOrgStore{err: errors.New("boom")}, uuid.Nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
