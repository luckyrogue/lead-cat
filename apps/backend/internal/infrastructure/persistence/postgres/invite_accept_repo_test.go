package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func seedUser(t *testing.T, s *pg.Store, email string) uuid.UUID {
	t.Helper()
	u, err := s.UpsertUserIdentity(context.Background(), "sub-"+uuid.NewString(), email)
	if err != nil {
		t.Fatalf("seedUser %s: %v", email, err)
	}
	return u.ID
}

func seedInvite(t *testing.T, s *pg.Store, orgID uuid.UUID, email, role string, createdBy uuid.UUID) uuid.UUID {
	t.Helper()
	inv, err := s.CreateInvite(context.Background(), orgID, email, role, []byte("hash-"+uuid.NewString()), time.Now().Add(24*time.Hour), createdBy)
	if err != nil {
		t.Fatalf("seedInvite: %v", err)
	}
	return inv.ID
}

func TestInviteAccept_ListAcceptDecline(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	bobID := seedUser(t, store, "bob@x.com")
	invID := seedInvite(t, store, orgID, "bob@x.com", "member", ownerID)

	views, err := store.ListPendingInvitesForEmail(ctx, "BOB@x.com")
	if err != nil || len(views) != 1 || views[0].InviteID != invID || views[0].OrgName == "" {
		t.Fatalf("list: %v %+v", err, views)
	}

	if err := store.AcceptInvite(ctx, invID, bobID, "eve@x.com"); !errors.Is(err, model.ErrInviteEmailMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
	if err := store.AcceptInvite(ctx, invID, bobID, "bob@x.com"); err != nil {
		t.Fatalf("accept: %v", err)
	}
	views, _ = store.ListPendingInvitesForEmail(ctx, "bob@x.com")
	if len(views) != 0 {
		t.Fatalf("expected no pending after accept, got %v", views)
	}
	if err := store.AcceptInvite(ctx, invID, bobID, "bob@x.com"); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound on re-accept, got %v", err)
	}
}

func TestInviteDecline(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	_ = seedUser(t, store, "carol@x.com")
	invID := seedInvite(t, store, orgID, "carol@x.com", "member", ownerID)
	if err := store.DeclineInvite(ctx, invID, "carol@x.com"); err != nil {
		t.Fatalf("decline: %v", err)
	}
	views, _ := store.ListPendingInvitesForEmail(ctx, "carol@x.com")
	if len(views) != 0 {
		t.Fatalf("declined invite should not be pending, got %v", views)
	}
}
