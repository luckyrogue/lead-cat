# Telegram Bot Registration (§3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A `/start` Telegram registration FSM that collects ФИО + corporate email, verifies the email by OTP, and creates a `bot_users` record (Telegram ID ↔ email ↔ name + role).

**Architecture:** A dedicated, testable `botreg` service (FSM with injected `userStore`/`otpSender`/`sessions` interfaces) drives the flow; FSM state lives in Redis. The existing `MultiHandler` routes `/start` and free-text (in private chats) to it. Reuses the existing `platformauth.OTP`. New global `bot_users` table, decoupled from `platform_users`/`employees`.

**Tech Stack:** Go 1.26, pgx, go-redis, go-telegram/bot, existing `platformauth.OTP`. Spec: `docs/superpowers/specs/2026-05-30-bot-registration-design.md`.

**Run from:** `backend/` with `env -u GOROOT go ...`.

---

### Task 1: Migration — bot_users table

**Files:**
- Create: `backend/migrations/20260530140000_bot_users.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE bot_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id BIGINT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS bot_users;
```

- [ ] **Step 2: Apply and verify**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && set -a && . ./.env && set +a && cd backend && env -u GOROOT go run ./cmd/migrate up`
Expected: `OK 20260530140000_bot_users.sql` and `successfully migrated database to version: 20260530140000`.

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/20260530140000_bot_users.sql
git commit -m "feat(bot): bot_users table"
```

---

### Task 2: Model + repository

**Files:**
- Modify: `backend/internal/infrastructure/persistence/postgres/models.go` (append `BotUser`)
- Create: `backend/internal/infrastructure/persistence/postgres/bot_user_repo.go`

(No DB unit test — package has no DB harness; build-verified, exercised by the FSM tests via fakes.)

- [ ] **Step 1: Append the model**

Append to `backend/internal/infrastructure/persistence/postgres/models.go`:
```go
type BotUser struct {
	ID         uuid.UUID `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
}
```

- [ ] **Step 2: Write the repository**

`backend/internal/infrastructure/persistence/postgres/bot_user_repo.go`:
```go
package postgres

import (
	"context"

	"github.com/google/uuid"
)

const botUserCols = `id, telegram_id, full_name, email, role`

func (s *Store) GetBotUserByTelegramID(ctx context.Context, telegramID int64) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE telegram_id = $1`, telegramID).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role)
	return u, err
}

func (s *Store) GetBotUserByEmail(ctx context.Context, email string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `SELECT `+botUserCols+` FROM bot_users WHERE email = $1`, email).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role)
	return u, err
}

func (s *Store) CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (BotUser, error) {
	var u BotUser
	err := s.pool.QueryRow(ctx, `
		INSERT INTO bot_users (telegram_id, full_name, email, role)
		VALUES ($1, $2, $3, $4)
		RETURNING `+botUserCols,
		telegramID, fullName, email, role).
		Scan(&u.ID, &u.TelegramID, &u.FullName, &u.Email, &u.Role)
	return u, err
}
```

- [ ] **Step 3: Build + commit**

Run: `cd backend && env -u GOROOT go build ./...` → builds.
```bash
git add backend/internal/infrastructure/persistence/postgres/models.go backend/internal/infrastructure/persistence/postgres/bot_user_repo.go
git commit -m "feat(bot): bot_users model + repository"
```

---

### Task 3: Config — BOT_ADMIN_TELEGRAM_IDS

**Files:**
- Modify: `backend/internal/platform/config/config.go`

- [ ] **Step 1: Add the field**

In the `Config` struct (after `CalendarStub bool`), add:
```go
	BotAdminTelegramIDs []int64
```

- [ ] **Step 2: Parse it in Load()**

In `config.go`, after the line `cfg.CalendarStub = strings.EqualFold(os.Getenv("CALENDAR_STUB"), "true")`, add:
```go
	for _, p := range strings.Split(os.Getenv("BOT_ADMIN_TELEGRAM_IDS"), ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			cfg.BotAdminTelegramIDs = append(cfg.BotAdminTelegramIDs, id)
		}
	}
```

- [ ] **Step 3: Ensure `strconv` is imported**

Check the import block of `config.go`. If `"strconv"` is not present, add it.

- [ ] **Step 4: Build + commit**

Run: `cd backend && env -u GOROOT go build ./...` → builds.
```bash
git add backend/internal/platform/config/config.go
git commit -m "feat(bot): BOT_ADMIN_TELEGRAM_IDS config"
```

---

### Task 4: Registration FSM service (TDD core)

**Files:**
- Create: `backend/internal/platform/botreg/service.go`
- Test: `backend/internal/platform/botreg/service_test.go`

- [ ] **Step 1: Write the failing test**

`backend/internal/platform/botreg/service_test.go`:
```go
package botreg

import (
	"context"
	"errors"
	"testing"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
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

type fakeOTP struct {
	sent     []string
	validCode string
}

func (f *fakeOTP) Send(_ context.Context, _, dest string) (string, error) {
	f.sent = append(f.sent, dest)
	return f.validCode, nil
}
func (f *fakeOTP) Verify(_ context.Context, _, _, code string) (bool, error) {
	return code == f.validCode, nil
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

func newSvc(admins ...int64) (*Service, *fakeUsers, *fakeOTP, *fakeSessions) {
	u, o, s := newFakeUsers(), &fakeOTP{validCode: "1234"}, newFakeSessions()
	return New(u, o, s, admins), u, o, s
}

func TestHappyPath_User(t *testing.T) {
	svc, users, otp, _ := newSvc()
	ctx := context.Background()
	const tg = int64(42)

	svc.Start(ctx, tg)

	if r, ok := svc.OnText(ctx, tg, "Иванов Иван"); !ok || r == "" {
		t.Fatalf("name step: ok=%v r=%q", ok, r)
	}
	if r, ok := svc.OnText(ctx, tg, "ivan@corp.kz"); !ok || r == "" {
		t.Fatalf("email step: ok=%v r=%q", ok, r)
	}
	if len(otp.sent) != 1 || otp.sent[0] != "ivan@corp.kz" {
		t.Fatalf("otp not sent: %+v", otp.sent)
	}
	if _, ok := svc.OnText(ctx, tg, "1234"); !ok {
		t.Fatal("otp step not handled")
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
	svc, users, _, _ := newSvc(42)
	ctx := context.Background()
	svc.Start(ctx, 42)
	svc.OnText(ctx, 42, "Admin User")
	svc.OnText(ctx, 42, "admin@corp.kz")
	svc.OnText(ctx, 42, "1234")
	if len(users.created) != 1 || users.created[0].Role != "admin" {
		t.Fatalf("expected admin role: %+v", users.created)
	}
}

func TestAlreadyRegistered(t *testing.T) {
	svc, users, _, sess := newSvc()
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
	svc, users, otp, _ := newSvc()
	ctx := context.Background()
	users.byEmail["taken@corp.kz"] = postgres.BotUser{Email: "taken@corp.kz"}
	svc.Start(ctx, 9)
	svc.OnText(ctx, 9, "Some One")
	svc.OnText(ctx, 9, "taken@corp.kz")
	if len(otp.sent) != 0 {
		t.Fatal("must not send OTP for a taken email")
	}
}

func TestBadEmailThenBadOTP(t *testing.T) {
	svc, users, _, _ := newSvc()
	ctx := context.Background()
	svc.Start(ctx, 5)
	svc.OnText(ctx, 5, "Name Here")
	if _, ok := svc.OnText(ctx, 5, "not-an-email"); !ok {
		t.Fatal("bad email should be handled (stay)")
	}
	svc.OnText(ctx, 5, "real@corp.kz")
	svc.OnText(ctx, 5, "0000") // wrong code
	if len(users.created) != 0 {
		t.Fatal("wrong OTP must not create a user")
	}
}

func TestNoSessionIgnored(t *testing.T) {
	svc, _, _, _ := newSvc()
	if _, ok := svc.OnText(context.Background(), 1, "random"); ok {
		t.Fatal("text with no active session must be ignored (ok=false)")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/platform/botreg/ -v`
Expected: FAIL — `New`, `Service`, `State` undefined.

- [ ] **Step 3: Write the service**

`backend/internal/platform/botreg/service.go`:
```go
// Package botreg drives the Telegram /start registration FSM: ФИО -> email ->
// OTP -> create bot_users. It depends on small interfaces so the flow is
// testable without Redis, Postgres, or Telegram.
package botreg

import (
	"context"
	"net/mail"
	"strings"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// State is the per-user FSM state (stored in Redis between messages).
type State struct {
	Step     string `json:"step"` // awaiting_name | awaiting_email | awaiting_otp
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

const (
	stepName  = "awaiting_name"
	stepEmail = "awaiting_email"
	stepOTP   = "awaiting_otp"
)

type userStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (postgres.BotUser, error)
}

type otpSender interface {
	Send(ctx context.Context, channel, dest string) (string, error)
	Verify(ctx context.Context, channel, dest, code string) (bool, error)
}

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	users    userStore
	otp      otpSender
	sessions sessions
	admins   map[int64]bool
}

func New(users userStore, otp otpSender, sess sessions, adminIDs []int64) *Service {
	admins := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = true
	}
	return &Service{users: users, otp: otp, sessions: sess, admins: admins}
}

// Start handles /start: returns a welcome for registered users, otherwise opens
// the registration flow and prompts for the full name.
func (s *Service) Start(ctx context.Context, telegramID int64) string {
	if _, err := s.users.GetBotUserByTelegramID(ctx, telegramID); err == nil {
		return "С возвращением! 🐾 Открой приложение из меню."
	}
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepName})
	return "Привет! Давай зарегистрируемся.\nВведи ФИО (Фамилия Имя Отчество):"
}

// OnText feeds a free-text message into an active registration session. The
// bool is false (and reply empty) when there is no active session.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (string, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return "", false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepName:
		if text == "" {
			return "Введи ФИО:", true
		}
		st.FullName = text
		st.Step = stepEmail
		_ = s.sessions.Set(ctx, telegramID, *st)
		return "Теперь корпоративную почту:", true

	case stepEmail:
		if _, perr := mail.ParseAddress(text); perr != nil {
			return "Не похоже на email. Попробуй ещё раз:", true
		}
		if _, gerr := s.users.GetBotUserByEmail(ctx, text); gerr == nil {
			return "Эта почта уже привязана к другому аккаунту.", true
		}
		if _, serr := s.otp.Send(ctx, "email", text); serr != nil {
			return "Не смог отправить код, попробуй позже.", true
		}
		st.Email = text
		st.Step = stepOTP
		_ = s.sessions.Set(ctx, telegramID, *st)
		return "Отправил код на почту. Введи его:", true

	case stepOTP:
		ok, _ := s.otp.Verify(ctx, "email", st.Email, text)
		if !ok {
			return "Неверный код. Попробуй ещё раз:", true
		}
		role := "user"
		if s.admins[telegramID] {
			role = "admin"
		}
		if _, cerr := s.users.CreateBotUser(ctx, telegramID, st.FullName, st.Email, role); cerr != nil {
			return "Не удалось завершить регистрацию, попробуй позже.", true
		}
		_ = s.sessions.Del(ctx, telegramID)
		return "Готово, " + st.FullName + "! 🐾", true
	}
	return "", false
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd backend && env -u GOROOT go test ./internal/platform/botreg/ -v`
Expected: PASS (all FSM tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/botreg/service.go backend/internal/platform/botreg/service_test.go
git commit -m "feat(bot): registration FSM service (name/email/OTP)"
```

---

### Task 5: Redis-backed session store

**Files:**
- Create: `backend/internal/platform/botreg/redis_sessions.go`

- [ ] **Step 1: Write the Redis sessions store**

`backend/internal/platform/botreg/redis_sessions.go`:
```go
package botreg

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisSessions stores FSM state in Redis with a TTL, keyed by Telegram ID.
type RedisSessions struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRedisSessions(rdb *redis.Client) *RedisSessions {
	return &RedisSessions{rdb: rdb, ttl: 15 * time.Minute}
}

func (r *RedisSessions) key(telegramID int64) string {
	return "botreg:" + strconv.FormatInt(telegramID, 10)
}

func (r *RedisSessions) Get(ctx context.Context, telegramID int64) (*State, error) {
	raw, err := r.rdb.Get(ctx, r.key(telegramID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *RedisSessions) Set(ctx context.Context, telegramID int64, s State) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, r.key(telegramID), raw, r.ttl).Err()
}

func (r *RedisSessions) Del(ctx context.Context, telegramID int64) error {
	return r.rdb.Del(ctx, r.key(telegramID)).Err()
}
```

- [ ] **Step 2: Build + commit**

Run: `cd backend && env -u GOROOT go build ./...` → builds (`*RedisSessions` satisfies the `botreg.sessions` interface).
```bash
git add backend/internal/platform/botreg/redis_sessions.go
git commit -m "feat(bot): redis-backed registration sessions"
```

---

### Task 6: Wire registration into the bot handler

**Files:**
- Modify: `backend/internal/infrastructure/telegram/multitenant.go` (Registrar field + routing)
- Modify: `backend/cmd/server/main.go` (build deps + new NewMultiHandler args)

- [ ] **Step 1: Add the Registrar to MultiHandler**

In `multitenant.go`, add imports `"github.com/redis/go-redis/v9"`, `platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"`, and `"github.com/Jaryq-Lab/notify-bot/internal/platform/botreg"`. Change the struct + constructor:
```go
type MultiHandler struct {
	store     *postgres.Store
	executor  *scenario_executor.Executor
	registrar *botreg.Service
	log       *zap.Logger
}

func NewMultiHandler(store *postgres.Store, cipher *crypto.TokenCipher, b *bot.Bot, rdb *redis.Client, adminIDs []int64, otpLog bool, log *zap.Logger) *MultiHandler {
	otp := platformauth.NewOTP(rdb, log, otpLog)
	registrar := botreg.New(store, otp, botreg.NewRedisSessions(rdb), adminIDs)
	return &MultiHandler{
		store:     store,
		executor:  scenario_executor.New(store, cipher, b, log),
		registrar: registrar,
		log:       log,
	}
}
```

- [ ] **Step 2: Route /start and free-text in Handle**

In `multitenant.go` `Handle`, replace the command-parse block:
```go
	cmd, ok := parseCommand(text)
	if !ok {
		return
	}
	switch cmd {
```
with:
```go
	cmd, ok := parseCommand(text)
	if !ok {
		if isPrivate {
			if reply, handled := h.registrar.OnText(ctx, from.ID, text); handled {
				h.reply(ctx, b, update.Message, reply)
			}
		}
		return
	}
	switch cmd {
	case "/start":
		if isPrivate {
			h.reply(ctx, b, update.Message, h.registrar.Start(ctx, from.ID))
		}
```
(Keep the existing `/chatid`, `/test`, `/report` cases after the new `/start` case.)

- [ ] **Step 3: Update the call site in cmd/server**

In `backend/cmd/server/main.go`, find the line:
```go
		tgHandler = telegram.NewMultiHandler(store, cipher, tg, logger)
```
and replace with:
```go
		tgHandler = telegram.NewMultiHandler(store, cipher, tg, rdb, cfg.BotAdminTelegramIDs, cfg.AuthOTPLog, logger)
```
(`rdb`, `cfg`, `logger` are all already in scope there.)

- [ ] **Step 4: Build, vet, test**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./... && env -u GOROOT go test -count=1 ./...`
Expected: all green (botreg FSM tests pass; everything compiles).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/telegram/multitenant.go backend/cmd/server/main.go
git commit -m "feat(bot): wire /start registration into the telegram handler"
```

---

### Task 7: Docs + env example

**Files:**
- Modify: `deploy/.env.example` (add `BOT_ADMIN_TELEGRAM_IDS=`)
- Modify: `docs/MEETINGS.md` (note bot registration)

- [ ] **Step 1: Add the env var**

Append to `deploy/.env.example` (near the bot/auth flags):
```
BOT_ADMIN_TELEGRAM_IDS=
```

- [ ] **Step 2: Update `docs/MEETINGS.md`**

In the Backend section, after the existing increment notes, add:
```markdown
> **Bot registration (done):** `/start` FSM (ФИО → corporate email → OTP) creates a global `bot_users` record (Telegram ID ↔ email ↔ name + role). Admins bootstrapped via `BOT_ADMIN_TELEGRAM_IDS`. FSM state in Redis; OTP reuses the email auth service. Requires the bot to be polling (real `BOT_TOKEN`, non-dev). Per-participant notifications (§5) will join `email → bot_users.telegram_id`.
```

- [ ] **Step 3: Format and commit**

Run `make fmt-check` (run `make fmt` if docs reflow; stage only these two).
```bash
git add deploy/.env.example docs/MEETINGS.md
git commit -m "docs(bot): document /start registration + BOT_ADMIN_TELEGRAM_IDS"
```

---

## Done criteria

- `make lint` → 0 issues; `make test` → all pass (incl. the `botreg` FSM suite); `make typecheck` → 0; `make fmt-check` → green; `make build`.
- The `botreg` FSM unit tests cover: happy path (user + admin role), already-registered, email-taken, invalid email, wrong OTP, no-session-ignored.
- Manual (out of CI, needs a real `BOT_TOKEN` polling): `/start` → ФИО → email → OTP code (from email or `AUTH_OTP_LOG`) → `bot_users` row created; second `/start` → "welcome back".
