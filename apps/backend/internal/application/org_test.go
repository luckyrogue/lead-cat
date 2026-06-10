package application

import (
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestMemberViews(t *testing.T) {
	ownerID := uuid.New()
	memberID := uuid.New()
	unknownID := uuid.New()

	members := []postgres.Member{
		{UserID: &ownerID, Role: "owner"},
		{UserID: &memberID, Role: "member"},
	}

	views, idx := memberViews(members, ownerID)
	if idx != 0 {
		t.Fatalf("expected idx=0 for owner, got %d", idx)
	}
	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}
	if views[0].Role != "owner" || views[1].Role != "member" {
		t.Fatalf("unexpected roles: %v", views)
	}

	_, idx = memberViews(members, memberID)
	if idx != 1 {
		t.Fatalf("expected idx=1 for member, got %d", idx)
	}

	_, idx = memberViews(members, unknownID)
	if idx != -1 {
		t.Fatalf("expected idx=-1 for unknown, got %d", idx)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Acme Inc.":       "acme-inc",
		"  Hello  World ": "hello-world",
		"Café Münchën":    "cafe-munchen",
		"MixedCASE 123":   "mixedcase-123",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", in, got, want)
		}
	}

	if got := slugify("Компания"); got != "" {
		t.Fatalf("slugify(cyrillic) = %q, want empty", got)
	}
}

func TestRoleAtLeast(t *testing.T) {
	if !RoleAtLeast("owner", "admin") {
		t.Fatal("owner >= admin")
	}
	if !RoleAtLeast("admin", "admin") {
		t.Fatal("admin >= admin")
	}
	if RoleAtLeast("member", "admin") {
		t.Fatal("member < admin")
	}
	if !RoleAtLeast("owner", "member") {
		t.Fatal("owner >= member")
	}
}

func TestCanDemoteOrRemoveBlocksLastOwner(t *testing.T) {
	members := []OrgMemberView{{Role: "owner"}, {Role: "member"}}
	if err := canDemoteOrRemove(members, 0); err == nil {
		t.Fatal("removing/demoting the only owner must fail")
	}
	members = append(members, OrgMemberView{Role: "owner"})
	if err := canDemoteOrRemove(members, 0); err != nil {
		t.Fatalf("two owners -> ok, got %v", err)
	}

	if err := canDemoteOrRemove(members, 1); err != nil {
		t.Fatalf("removing member -> ok, got %v", err)
	}
}
