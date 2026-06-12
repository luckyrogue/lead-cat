package postgres

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestMeetingFilter(t *testing.T) {
	org := uuid.New()
	org2 := uuid.New()
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	t.Run("empty filter is org-only", func(t *testing.T) {
		where, args := meetingFilter(org, model.MeetingFilter{})
		if where != "organization_id = $1" {
			t.Fatalf("where = %q", where)
		}
		if len(args) != 1 || args[0] != org {
			t.Fatalf("args = %v", args)
		}
	})

	t.Run("all fields", func(t *testing.T) {
		where, args := meetingFilter(org, model.MeetingFilter{
			Status: "scheduled", From: &from, To: &to, Dept: "eng", Organizer: &org2,
		})
		want := "organization_id = $1 AND status = $2 AND starts_at >= $3 AND starts_at < $4 AND dept ILIKE $5 AND organizer_user_id = $6"
		if where != want {
			t.Fatalf("where = %q", where)
		}
		if len(args) != 6 {
			t.Fatalf("len(args) = %d", len(args))
		}
		if args[1] != "scheduled" || args[2] != from || args[3] != to || args[4] != "%eng%" || args[5] != org2 {
			t.Fatalf("args = %v", args)
		}
	})

	t.Run("status all is ignored", func(t *testing.T) {
		where, args := meetingFilter(org, model.MeetingFilter{Status: "all"})
		if where != "organization_id = $1" || len(args) != 1 {
			t.Fatalf("where = %q args = %v", where, args)
		}
	})
}
