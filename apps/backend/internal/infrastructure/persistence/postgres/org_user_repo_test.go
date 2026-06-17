package postgres_test

import (
	"context"
	"testing"
)

func TestUser_UpsertIdempotentByEmail(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	u1, err := s.UpsertUserIdentity(ctx, "sub-1", "same@x.io")
	if err != nil {
		t.Fatalf("upsert1: %v", err)
	}
	u2, err := s.UpsertUserIdentity(ctx, "sub-1", "same@x.io")
	if err != nil {
		t.Fatalf("upsert2: %v", err)
	}
	if u1.ID != u2.ID {
		t.Fatalf("upsert created a second user: %v vs %v", u1.ID, u2.ID)
	}
	got, err := s.GetUserByID(ctx, u1.ID)
	if err != nil || got.Email != "same@x.io" {
		t.Fatalf("get user: %v / %+v", err, got)
	}
}

func TestOrg_CreateAndGet(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	owner, _ := s.UpsertUserIdentity(ctx, "owner", "owner@x.io")
	org, err := s.CreateOrganization(ctx, "Acme", "acme", owner.ID)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	got, err := s.GetOrganization(ctx, org.ID)
	if err != nil || got.Name != "Acme" || got.Slug != "acme" {
		t.Fatalf("get org: %v / %+v", err, got)
	}
}

func TestOrgMembers_RoleAndRemoval(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	owner, _ := s.UpsertUserIdentity(ctx, "owner", "owner@x.io")
	org, _ := s.CreateOrganization(ctx, "Acme", "acme2", owner.ID)

	members, err := s.ListOrgMembers(ctx, org.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) == 0 {
		t.Fatal("expected at least the owner as a member")
	}

	if err := s.UpdateMemberRole(ctx, org.ID, owner.ID, "admin"); err != nil {
		t.Fatalf("update role: %v", err)
	}
	m, ok, err := s.GetOrgMember(ctx, org.ID, owner.ID)
	if err != nil || !ok || m.Role != "admin" {
		t.Fatalf("role not updated: %v ok=%v %+v", err, ok, m)
	}

	if err := s.RemoveMember(ctx, org.ID, owner.ID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, ok, _ := s.GetOrgMember(ctx, org.ID, owner.ID); ok {
		t.Fatal("member still present after removal")
	}
}
