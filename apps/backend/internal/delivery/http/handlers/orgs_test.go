package handlers

import (
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestMemberView(t *testing.T) {
	uid := uuid.New()
	invited := "pending@x.io"

	cases := []struct {
		name              string
		in                model.Member
		email, disp, stat string
	}{
		{
			name:  "linked with telegram",
			in:    model.Member{UserID: &uid, Email: "a@x.io", TelegramUsername: "alice"},
			email: "a@x.io", disp: "@alice", stat: "active",
		},
		{
			name:  "linked email only",
			in:    model.Member{UserID: &uid, Email: "bob@x.io"},
			email: "bob@x.io", disp: "bob", stat: "active",
		},
		{
			name:  "invited pending",
			in:    model.Member{InvitedEmail: &invited},
			email: "pending@x.io", disp: "pending", stat: "invited",
		},
		{
			name:  "telegram prefix stripped",
			in:    model.Member{UserID: &uid, TelegramUsername: "@carol"},
			email: "", disp: "@carol", stat: "active",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			email, disp, stat := memberView(tc.in)
			if email != tc.email || disp != tc.disp || stat != tc.stat {
				t.Fatalf("got (%q,%q,%q) want (%q,%q,%q)", email, disp, stat, tc.email, tc.disp, tc.stat)
			}
		})
	}
}
