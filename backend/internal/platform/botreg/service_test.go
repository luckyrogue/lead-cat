package botreg

import (
	"context"
	"errors"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

var errNotFound = errors.New("not found")

type fakeUsers struct {
	byTG    map[int64]postgres.BotUser
	byEmail map[string]postgres.BotUser
	created []postgres.BotUser
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byTG: map[int64]postgres.BotUser{}, byEmail: map[string]postgres.BotUser{}}
}
func (f *fakeUsers) GetBotUserByTelegramID(_ context.Context, id int64) (postgres.BotUser, error) {
	if u, ok := f.byTG[id]; ok {
		return u, nil
	}
	return postgres.BotUser{}, errNotFound
}
func (f *fakeUsers) GetBotUserByEmail(_ context.Context, email string) (postgres.BotUser, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return postgres.BotUser{}, errNotFound
}
func (f *fakeUsers) CreateBotUser(_ context.Context, id int64, name, email, role string) (postgres.BotUser, error) {
	u := postgres.BotUser{TelegramID: id, FullName: name, Email: email, Role: role}
	f.byTG[id] = u
	f.byEmail[email] = u
	f.created = append(f.created, u)
	return u, nil
}

type fakeSessions struct{ m map[int64]State }

func newFakeSessions() *fakeSessions { return &fakeSessions{m: map[int64]State{}} }
func (f *fakeSessions) Get(_ context.Context, id int64) (*State, error) {
	if s, ok := f.m[id]; ok {
		return &s, nil
	}
	return nil, nil
}
func (f *fakeSessions) Set(_ context.Context, id int64, s State) error { f.m[id] = s; return nil }
func (f *fakeSessions) Del(_ context.Context, id int64) error          { delete(f.m, id); return nil }

func newSvc(admins ...int64) (*Service, *fakeUsers, *fakeSessions) {
	u, s := newFakeUsers(), newFakeSessions()
	return New(u, s, admins), u, s
}

func TestHappyPath_User(t *testing.T) {
	svc, users, _ := newSvc()
	ctx := context.Background()
	const tg = int64(42)
	svc.Start(ctx, tg)
	if r, ok := svc.OnText(ctx, tg, "Иванов Иван"); !ok || r == "" {
		t.Fatalf("name step: ok=%v r=%q", ok, r)
	}
	if r, ok := svc.OnText(ctx, tg, "ivan@corp.kz"); !ok || r == "" {
		t.Fatalf("email step: ok=%v r=%q", ok, r)
	}
	if len(users.created) != 1 {
		t.Fatalf("user not created: %+v", users.created)
	}
	got := users.created[0]
	if got.FullName != "Иванов Иван" || got.Email != "ivan@corp.kz" || got.Role != "user" {
		t.Fatalf("bad user: %+v", got)
	}
}

func TestAdminRole(t *testing.T) {
	svc, users, _ := newSvc(42)
	ctx := context.Background()
	svc.Start(ctx, 42)
	svc.OnText(ctx, 42, "Admin User")
	svc.OnText(ctx, 42, "admin@corp.kz")
	if len(users.created) != 1 || users.created[0].Role != "admin" {
		t.Fatalf("expected admin role: %+v", users.created)
	}
}

func TestAlreadyRegistered(t *testing.T) {
	svc, users, sess := newSvc()
	users.byTG[7] = postgres.BotUser{TelegramID: 7}
	r := svc.Start(context.Background(), 7)
	if r == "" {
		t.Fatal("expected welcome-back reply")
	}
	if s, _ := sess.Get(context.Background(), 7); s != nil {
		t.Fatal("should not start a session for a registered user")
	}
}

func TestEmailTaken(t *testing.T) {
	svc, users, _ := newSvc()
	ctx := context.Background()
	users.byEmail["taken@corp.kz"] = postgres.BotUser{Email: "taken@corp.kz"}
	svc.Start(ctx, 9)
	svc.OnText(ctx, 9, "Some One")
	r, ok := svc.OnText(ctx, 9, "taken@corp.kz")
	if !ok || r == "" {
		t.Fatal("expected taken-email reply")
	}
	if len(users.created) != 0 {
		t.Fatal("must not create user for taken email")
	}
}

func TestBadEmail(t *testing.T) {
	svc, users, _ := newSvc()
	ctx := context.Background()
	svc.Start(ctx, 5)
	svc.OnText(ctx, 5, "Name Here")
	if _, ok := svc.OnText(ctx, 5, "not-an-email"); !ok {
		t.Fatal("bad email should be handled (stay)")
	}
	svc.OnText(ctx, 5, "real@corp.kz")
	if len(users.created) != 1 {
		t.Fatal("valid email should create user")
	}
}

func TestNoSessionIgnored(t *testing.T) {
	svc, _, _ := newSvc()
	if _, ok := svc.OnText(context.Background(), 1, "random"); ok {
		t.Fatal("text with no active session must be ignored (ok=false)")
	}
}

func TestEmailNormalized(t *testing.T) {
	svc, users, _ := newSvc()
	ctx := context.Background()
	svc.Start(ctx, 11)
	svc.OnText(ctx, 11, "Name Here")
	svc.OnText(ctx, 11, "  Ivan@Corp.KZ ")
	if len(users.created) != 1 || users.created[0].Email != "ivan@corp.kz" {
		t.Fatalf("email not normalized on create: %+v", users.created)
	}
}
