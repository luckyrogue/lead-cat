package botreg

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeUsers struct {
	byTelegram map[int64]postgres.BotUser
	byEmail    map[string]postgres.BotUser
	created    []postgres.BotUser
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byTelegram: map[int64]postgres.BotUser{}, byEmail: map[string]postgres.BotUser{}}
}
func (f *fakeUsers) GetBotUserByTelegramID(_ context.Context, tid int64) (postgres.BotUser, error) {
	if u, ok := f.byTelegram[tid]; ok {
		return u, nil
	}
	return postgres.BotUser{}, errors.New("not found")
}
func (f *fakeUsers) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return postgres.BotUser{}, errors.New("not found")
}
func (f *fakeUsers) CreateBotUser(_ context.Context, tid int64, fullName, email, role string) (postgres.BotUser, error) {
	u := postgres.BotUser{TelegramID: tid, FullName: fullName, Email: email, Role: role}
	f.created = append(f.created, u)
	f.byTelegram[tid] = u
	return u, nil
}

var _ userStore = (*fakeUsers)(nil)

type fakeSessions struct {
	m map[int64]State
}

func newFakeSessions() *fakeSessions { return &fakeSessions{m: map[int64]State{}} }
func (f *fakeSessions) Get(_ context.Context, tid int64) (*State, error) {
	if s, ok := f.m[tid]; ok {
		c := s
		return &c, nil
	}
	return nil, nil
}
func (f *fakeSessions) Set(_ context.Context, tid int64, s State) error { f.m[tid] = s; return nil }
func (f *fakeSessions) Del(_ context.Context, tid int64) error          { delete(f.m, tid); return nil }

var _ sessions = (*fakeSessions)(nil)

func TestRegistration_HappyPath_NonAdmin(t *testing.T) {
	users := newFakeUsers()
	sess := newFakeSessions()
	s := New(users, sess, nil)
	ctx := context.Background()

	if msg := s.Start(ctx, 100, "ru"); msg == "" {
		t.Fatal("Start should prompt for name")
	}
	if _, ok := sess.m[100]; !ok {
		t.Fatal("Start should set a session")
	}
	if _, ok := s.OnText(ctx, 100, "Иванов Иван", "ru"); !ok {
		t.Fatal("name step should handle text")
	}
	if _, ok := s.OnText(ctx, 100, "ivan@corp.io", "ru"); !ok {
		t.Fatal("email step should handle text")
	}
	if len(users.created) != 1 {
		t.Fatalf("want 1 user created, got %d", len(users.created))
	}
	if users.created[0].Role != "user" || users.created[0].FullName != "Иванов Иван" || users.created[0].Email != "ivan@corp.io" {
		t.Fatalf("created user wrong: %+v", users.created[0])
	}
	if _, ok := sess.m[100]; ok {
		t.Fatal("session should be deleted after registration")
	}
}

func TestRegistration_AdminRole(t *testing.T) {
	users := newFakeUsers()
	sess := newFakeSessions()
	s := New(users, sess, []int64{42})
	ctx := context.Background()
	s.Start(ctx, 42, "ru")
	s.OnText(ctx, 42, "Admin User", "ru")
	s.OnText(ctx, 42, "admin@corp.io", "ru")
	if len(users.created) != 1 || users.created[0].Role != "admin" {
		t.Fatalf("admin id should create admin role: %+v", users.created)
	}
}

func TestStart_AlreadyRegistered(t *testing.T) {
	users := newFakeUsers()
	users.byTelegram[7] = postgres.BotUser{TelegramID: 7}
	sess := newFakeSessions()
	s := New(users, sess, nil)
	msg := s.Start(context.Background(), 7, "ru")
	if msg == "" {
		t.Fatal("should greet returning user")
	}
	if _, ok := sess.m[7]; ok {
		t.Fatal("should not start a session for a registered user")
	}
}

func TestRegistration_RejectsBadEmailAndDuplicate(t *testing.T) {
	users := newFakeUsers()
	users.byEmail["taken@corp.io"] = postgres.BotUser{Email: "taken@corp.io"}
	sess := newFakeSessions()
	s := New(users, sess, nil)
	ctx := context.Background()
	s.Start(ctx, 5, "ru")
	s.OnText(ctx, 5, "Some Name", "ru")
	if _, ok := s.OnText(ctx, 5, "not-an-email", "ru"); !ok {
		t.Fatal("bad email should be handled")
	}
	if len(users.created) != 0 {
		t.Fatal("bad email must not create a user")
	}
	if _, ok := s.OnText(ctx, 5, "taken@corp.io", "ru"); !ok {
		t.Fatal("duplicate email should be handled")
	}
	if len(users.created) != 0 {
		t.Fatal("duplicate email must not create a user")
	}
}

func TestStart_Localized(t *testing.T) {
	svc := New(newFakeUsers(), newFakeSessions(), nil)
	en := svc.Start(context.Background(), 1, "en")
	ru := svc.Start(context.Background(), 2, "ru")
	if en == ru {
		t.Fatalf("Start must differ by language; both = %q", en)
	}
	if !strings.Contains(en, "register") {
		t.Errorf("en Start = %q", en)
	}
}

func TestOnText_NameStep_Localized(t *testing.T) {
	users := newFakeUsers()
	sess := newFakeSessions()
	svc := New(users, sess, nil)
	_ = svc.Start(context.Background(), 1, "en") // sets awaiting_name
	reply, handled := svc.OnText(context.Background(), 1, "John Smith", "en")
	if !handled {
		t.Fatal("expected handled")
	}
	if !strings.Contains(reply, "email") {
		t.Errorf("en ask_email = %q", reply)
	}
}
