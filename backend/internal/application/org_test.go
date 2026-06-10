package application

import "testing"

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
	// Non-latin (e.g. Cyrillic) folds to empty -> slugify returns "" and the
	// caller substitutes a fallback; assert the empty contract here.
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
	// removing a non-owner is always fine
	if err := canDemoteOrRemove(members, 1); err != nil {
		t.Fatalf("removing member -> ok, got %v", err)
	}
}
