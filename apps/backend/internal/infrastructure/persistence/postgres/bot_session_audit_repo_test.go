package postgres_test

import (
	"context"
	"testing"
	"time"

	pg "github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestBotUser_CreateGetAndPrefs(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	if _, err := s.CreateBotUser(ctx, 12345, "Mia", "mia@x.io", "user"); err != nil {
		t.Fatalf("create bot user: %v", err)
	}
	got, err := s.GetBotUserByTelegramID(ctx, 12345)
	if err != nil || got.FullName != "Mia" || got.Email != "mia@x.io" {
		t.Fatalf("get bot user: %v / %+v", err, got)
	}
	if err := s.SetReminderMinutes(ctx, 12345, "15,60"); err != nil {
		t.Fatalf("set reminders: %v", err)
	}
	if err := s.SetBotUserPrefs(ctx, 12345, "Asia/Almaty", "ru"); err != nil {
		t.Fatalf("set prefs: %v", err)
	}
	got, _ = s.GetBotUserByTelegramID(ctx, 12345)
	if got.ReminderMinutes != "15,60" || got.Timezone != "Asia/Almaty" || got.Language != "ru" {
		t.Fatalf("prefs not persisted: %+v", got)
	}
}

func TestWebSession_CreateResolveRevoke(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	owner, _ := s.UpsertUserIdentity(ctx, "ws-owner", "ws@x.io")
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	hash := []byte("token-hash-bytes")
	if _, err := s.CreateWebSession(ctx, hash, owner.ID, now.Add(time.Hour), "ua", "127.0.0.1"); err != nil {
		t.Fatalf("create session: %v", err)
	}
	sess, ok, err := s.ResolveWebSession(ctx, hash, now)
	if err != nil || !ok {
		t.Fatalf("resolve: %v ok=%v", err, ok)
	}
	if sess.UserID != owner.ID {
		t.Fatalf("wrong user: %v", sess.UserID)
	}
	if _, ok, _ := s.ResolveWebSession(ctx, hash, now.Add(2*time.Hour)); ok {
		t.Fatal("expired session should not resolve")
	}
	if err := s.RevokeWebSession(ctx, hash, now); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, ok, _ := s.ResolveWebSession(ctx, hash, now); ok {
		t.Fatal("revoked session should not resolve")
	}
}

func TestAudit_InsertAndList(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	// admin_audit_log.actor_user_id is NOT NULL REFERENCES bot_users(id),
	// so seed a bot_user and use its id as the actor.
	actor, err := s.CreateBotUser(ctx, 777, "Admin", "admin@x.io", "admin")
	if err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	if err := s.InsertAuditEntry(ctx, pg.AuditEntry{
		ActorUserID: actor.ID, ActorTelegramID: 777, ActorEmail: "admin@x.io",
		ActorKind: "bot", Action: "meeting.create",
		TargetKind: "meeting", TargetID: "m1",
	}); err != nil {
		t.Fatalf("insert audit: %v", err)
	}
	entries, err := s.ListAuditEntries(ctx, pg.AuditFilter{Limit: 10})
	if err != nil || len(entries) != 1 || entries[0].Action != "meeting.create" {
		t.Fatalf("list audit: %v / %+v", err, entries)
	}
}

func TestReminderClaim_Idempotent(t *testing.T) {
	testDB.Truncate(t)
	s := newStore()
	ctx := context.Background()
	org := seedOrg(t, s)
	m := seedMeeting(t, s, org)
	first, err := s.TryClaimReminder(ctx, m.ID, 999, 15)
	if err != nil || !first {
		t.Fatalf("first claim should succeed: %v ok=%v", err, first)
	}
	second, err := s.TryClaimReminder(ctx, m.ID, 999, 15)
	if err != nil || second {
		t.Fatalf("second identical claim should be false: %v ok=%v", err, second)
	}
}
