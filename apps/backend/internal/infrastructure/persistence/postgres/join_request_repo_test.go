package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestJoinRequest_Lifecycle(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	daveID := seedUser(t, store, "dave@x.com")

	if err := store.CreateJoinRequest(ctx, orgID, daveID); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.CreateJoinRequest(ctx, orgID, daveID); err != nil {
		t.Fatalf("create idempotent: %v", err)
	}
	mine, err := store.ListJoinRequestsForUser(ctx, daveID)
	if err != nil || len(mine) != 1 || mine[0].Status != "pending" || mine[0].OrgName == "" {
		t.Fatalf("list-mine: %v %+v", err, mine)
	}
	pend, err := store.ListPendingJoinRequests(ctx, orgID)
	if err != nil || len(pend) != 1 || pend[0].Email != "dave@x.com" {
		t.Fatalf("list-pending: %v %+v", err, pend)
	}
	if err := store.AcceptJoinRequest(ctx, orgID, pend[0].RequestID, ownerID); err != nil {
		t.Fatalf("accept: %v", err)
	}
	if _, ok, _ := store.GetOrgMember(ctx, orgID, daveID); !ok {
		t.Fatal("expected membership after accept")
	}
	if pend2, _ := store.ListPendingJoinRequests(ctx, orgID); len(pend2) != 0 {
		t.Fatalf("expected 0 pending after accept, got %d", len(pend2))
	}
}

func TestJoinRequest_Decline(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	eveID := seedUser(t, store, "eve@x.com")
	if err := store.CreateJoinRequest(ctx, orgID, eveID); err != nil {
		t.Fatal(err)
	}
	pend, _ := store.ListPendingJoinRequests(ctx, orgID)
	if err := store.DeclineJoinRequest(ctx, orgID, pend[0].RequestID, ownerID); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, ok, _ := store.GetOrgMember(ctx, orgID, eveID); ok {
		t.Fatal("declined request must not create membership")
	}
	if p, _ := store.ListPendingJoinRequests(ctx, orgID); len(p) != 0 {
		t.Fatalf("declined request should not be pending")
	}
	if err := store.CreateJoinRequest(ctx, orgID, eveID); err != nil {
		t.Fatalf("re-request after decline: %v", err)
	}
}

func TestJoinRequest_AcceptNotFound(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	daveID := seedUser(t, store, "dave2@x.com")
	if err := store.CreateJoinRequest(ctx, orgID, daveID); err != nil {
		t.Fatal(err)
	}
	pend, _ := store.ListPendingJoinRequests(ctx, orgID)
	if err := store.AcceptJoinRequest(ctx, orgID, pend[0].RequestID, ownerID); err != nil {
		t.Fatal(err)
	}
	if err := store.AcceptJoinRequest(ctx, orgID, pend[0].RequestID, ownerID); !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound on re-accept, got %v", err)
	}
}

func TestDeclineJoinRequest_NotFound(t *testing.T) {
	testDB.Truncate(t)
	store := newStore()
	ctx := context.Background()
	orgID, ownerID := seedOrg(t, store)
	unknownRequestID := uuid.New()
	err := store.DeclineJoinRequest(ctx, orgID, unknownRequestID, ownerID)
	if !model.IsNotFound(err) {
		t.Fatalf("expected IsNotFound for unknown request id, got %v", err)
	}
}
