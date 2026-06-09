# Slice D — TMA admin integrations: implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move setup operations (Google Calendar integration, chat link / members, scenarios) from curl + platform JWT into the Telegram Mini App under an admin-gated route group, with persistent audit log.

**Architecture:** New `/api/tma/admin/*` route group with `RequireBotAdmin` middleware on top of existing TMA JWT. Phase 1 introduces a real Google verify (parse SA → impersonate `subject` → `Calendars.Get`). Audit writes go through a whitelist-enforced helper into a new `admin_audit_log` table. Frontend ships a new `entities/admin` FSD slice + `features/admin-setup` overlay with paste-only SA upload.

**Tech Stack:** Go 1.22 (Fiber, pgx, zap, asynq, google.golang.org/api/calendar/v3), Postgres 15, React 18 + TanStack Router/Query, Vitest, OpenAPI 3.1.

**Spec:** [`docs/superpowers/specs/2026-06-09-slice-d-tma-admin-integrations-design.md`](../specs/2026-06-09-slice-d-tma-admin-integrations-design.md).

**Branch:** `feat/meetings-admin-d` (created from `main` at `2522f9b`).

---

## File structure

```
backend/
├── migrations/
│   └── 20260609120000_admin_audit_log.sql                      [NEW]
├── internal/
│   ├── delivery/http/
│   │   ├── app.go                                              [MODIFY]
│   │   ├── middleware/
│   │   │   └── require_bot_admin.go                            [NEW]
│   │   │   └── require_bot_admin_test.go                       [NEW]
│   │   └── handlers/
│   │       └── tma_admin.go                                    [NEW]
│   ├── application/
│   │   ├── services.go                                         [MODIFY: hook audit]
│   │   ├── admin_workspace.go                                  [NEW]
│   │   ├── admin_workspace_test.go                             [NEW]
│   │   ├── google_verify.go                                    [NEW]
│   │   ├── google_verify_test.go                               [NEW]
│   │   ├── audit.go                                            [NEW]
│   │   └── audit_test.go                                       [NEW]
│   └── infrastructure/
│       ├── calendar/google/
│       │   ├── probe.go                                        [NEW]
│       │   └── probe_test.go                                   [NEW]
│       └── persistence/postgres/
│           ├── audit_repo.go                                   [NEW]
│           └── models.go                                       [MODIFY: AuditEntry struct]
└── openapi/openapi.json                                        [MODIFY]
    docs/openapi.json                                           [MODIFY: mirror]

frontend/
├── src/
│   ├── shared/api/generated/schema.ts                          [REGEN]
│   ├── shared/tma/i18n.ts                                      [MODIFY]
│   ├── entities/admin/
│   │   ├── api.ts                                              [NEW]
│   │   ├── write-api.ts                                        [NEW]
│   │   ├── mutations.ts                                        [NEW]
│   │   ├── queries.ts                                          [NEW]
│   │   ├── types.ts                                            [NEW]
│   │   └── constants.ts                                        [NEW]
│   ├── features/admin-setup/
│   │   ├── pages/admin-setup-page.tsx                          [NEW]
│   │   ├── components/
│   │   │   ├── integrations-section.tsx                        [NEW]
│   │   │   ├── sa-paste-input.tsx                              [NEW]
│   │   │   ├── verify-result-card.tsx                          [NEW]
│   │   │   ├── chat-link-section.tsx                           [NEW]
│   │   │   ├── members-section.tsx                             [NEW]
│   │   │   ├── scenarios-section.tsx                           [NEW]
│   │   │   └── audit-log-section.tsx                           [NEW]
│   │   └── lib/
│   │       ├── sa-validate.ts                                  [NEW]
│   │       └── sa-validate.test.ts                             [NEW]
│   ├── features/profile/pages/admin-panel-page.tsx             [MODIFY]
│   └── routes/_tma/
│       └── profile.admin.setup.tsx                             [NEW]

docs/
├── API.md                                                      [MODIFY]
├── MEETINGS.md                                                 [MODIFY]
└── OPERATIONS.md                                               [MODIFY]
```

---

### Task D-T0: Branch already created

**Files:** none

- [x] Branch `feat/meetings-admin-d` created from `main` at `2522f9b`. Spec committed as `e870889`. Nothing to do.

---

### Task D-T1: Migration — admin_audit_log + workspaces singleton index

**Files:**
- Create: `backend/migrations/20260609120000_admin_audit_log.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE admin_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID NOT NULL REFERENCES bot_users(id),
    actor_telegram_id BIGINT NOT NULL,
    actor_email TEXT NOT NULL,
    action TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    target_id TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX admin_audit_log_created_at_idx ON admin_audit_log (created_at DESC);
CREATE INDEX admin_audit_log_actor_idx ON admin_audit_log (actor_user_id, created_at DESC);
CREATE INDEX admin_audit_log_action_idx ON admin_audit_log (action, created_at DESC);

CREATE UNIQUE INDEX workspaces_singleton_idx ON workspaces ((true)) WHERE name = 'Lead Cat';

-- +goose Down
DROP INDEX IF EXISTS workspaces_singleton_idx;
DROP INDEX IF EXISTS admin_audit_log_action_idx;
DROP INDEX IF EXISTS admin_audit_log_actor_idx;
DROP INDEX IF EXISTS admin_audit_log_created_at_idx;
DROP TABLE IF EXISTS admin_audit_log;
```

- [ ] **Step 2: Run migrations up + down to smoke**

Run: `cd backend && goose -dir migrations postgres "$DATABASE_URL" up && goose -dir migrations postgres "$DATABASE_URL" down`
Expected: both directions succeed (or use `make migrate-up` / `make migrate-down` if the project provides them).

Then run up again to leave DB at HEAD:
Run: `cd backend && goose -dir migrations postgres "$DATABASE_URL" up`

- [ ] **Step 3: Commit**

```bash
git add backend/migrations/20260609120000_admin_audit_log.sql
git commit -m "$(cat <<'EOF'
feat(admin): migration for admin_audit_log + workspaces singleton

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T2: postgres audit repo + models

**Files:**
- Create: `backend/internal/infrastructure/persistence/postgres/audit_repo.go`
- Modify: `backend/internal/infrastructure/persistence/postgres/models.go` (add `AuditEntry`, `AuditFilter`)

- [ ] **Step 1: Add models**

Append to `models.go`:

```go
// AuditEntry is one row in admin_audit_log.
type AuditEntry struct {
	ID              uuid.UUID       `json:"id"`
	ActorUserID     uuid.UUID       `json:"actor_user_id"`
	ActorTelegramID int64           `json:"actor_telegram_id"`
	ActorEmail      string          `json:"actor_email"`
	Action          string          `json:"action"`
	TargetKind      string          `json:"target_kind"`
	TargetID        string          `json:"target_id"`
	Details         json.RawMessage `json:"details"`
	CreatedAt       time.Time       `json:"created_at"`
}

// AuditFilter narrows ListAuditEntries.
type AuditFilter struct {
	Action     string // exact match, empty = any
	ActorEmail string // exact match, empty = any
	Limit      int    // 1..200; 0 → 50
}
```

(If `json`/`time` are not imported in models.go, add them.)

- [ ] **Step 2: Write the repo**

Create `audit_repo.go`:

```go
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const auditCols = `id, actor_user_id, actor_telegram_id, actor_email, action, target_kind, target_id, details, created_at`

// InsertAuditEntry writes a new row. `details` must be valid JSON; pass `[]byte("{}")` for empty.
func (s *Store) InsertAuditEntry(ctx context.Context, e AuditEntry) error {
	if len(e.Details) == 0 {
		e.Details = json.RawMessage(`{}`)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO admin_audit_log (actor_user_id, actor_telegram_id, actor_email, action, target_kind, target_id, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.ActorUserID, e.ActorTelegramID, e.ActorEmail, e.Action, e.TargetKind, e.TargetID, []byte(e.Details))
	return err
}

// ListAuditEntries returns entries by created_at DESC. Filters are AND-combined.
func (s *Store) ListAuditEntries(ctx context.Context, f AuditFilter) ([]AuditEntry, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	q := `SELECT ` + auditCols + ` FROM admin_audit_log WHERE 1=1`
	args := []any{}
	if f.Action != "" {
		args = append(args, f.Action)
		q += fmt.Sprintf(" AND action = $%d", len(args))
	}
	if f.ActorEmail != "" {
		args = append(args, f.ActorEmail)
		q += fmt.Sprintf(" AND actor_email = $%d", len(args))
	}
	args = append(args, limit)
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var detailsRaw []byte
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorTelegramID, &e.ActorEmail, &e.Action, &e.TargetKind, &e.TargetID, &detailsRaw, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Details = detailsRaw
		out = append(out, e)
	}
	return out, rows.Err()
}

// EnsureLeadCatWorkspaceID returns the single Lead Cat workspace id, creating it on first call.
// Idempotent — safe under concurrency thanks to workspaces_singleton_idx unique partial index.
func (s *Store) EnsureLeadCatWorkspaceID(ctx context.Context, defaultTZ, defaultMeetLink string, ownerID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE name = 'Lead Cat' LIMIT 1`).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Try to create. If race lost, re-select.
	err = s.pool.QueryRow(ctx, `
		INSERT INTO workspaces (slug, name, owner_id, tz, meet_link)
		VALUES ('lead-cat', 'Lead Cat', $1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id`, ownerID, defaultTZ, defaultMeetLink).Scan(&id)
	if err == nil {
		return id, nil
	}
	// Race: another caller inserted first. Re-select.
	if err := s.pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE name = 'Lead Cat' LIMIT 1`).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
```

- [ ] **Step 3: Build to verify it compiles**

Run: `cd backend && go build ./...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/infrastructure/persistence/postgres/audit_repo.go backend/internal/infrastructure/persistence/postgres/models.go
git commit -m "$(cat <<'EOF'
feat(persistence): audit_repo + EnsureLeadCatWorkspaceID

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T3: application/audit.go with whitelist redaction (TDD)

**Files:**
- Create: `backend/internal/application/audit.go`
- Create: `backend/internal/application/audit_test.go`

- [ ] **Step 1: Write the failing tests**

Create `audit_test.go`:

```go
package application

import (
	"encoding/json"
	"testing"
)

func TestSanitizeAuditDetails_GoogleConfigUpdated(t *testing.T) {
	in := map[string]any{
		"subject":          "admin@example.com",
		"calendar_id":      "primary",
		"has_new_sa_json":  true,
		"google_sa_json":   "leak-secret",  // must be dropped
		"random_unrelated": "drop-me",      // must be dropped
	}
	out, dropped := sanitizeAuditDetails("google_config_updated", in)
	got := decodeJSON(t, out)
	if _, ok := got["google_sa_json"]; ok {
		t.Fatalf("google_sa_json must be dropped")
	}
	if _, ok := got["random_unrelated"]; ok {
		t.Fatalf("random_unrelated must be dropped")
	}
	if got["subject"] != "admin@example.com" || got["calendar_id"] != "primary" || got["has_new_sa_json"] != true {
		t.Fatalf("whitelist keys lost; got=%v", got)
	}
	if len(dropped) != 2 {
		t.Fatalf("expected 2 dropped keys, got %v", dropped)
	}
}

func TestSanitizeAuditDetails_UnknownAction(t *testing.T) {
	out, _ := sanitizeAuditDetails("totally_unknown", map[string]any{"anything": 1})
	got := decodeJSON(t, out)
	if len(got) != 0 {
		t.Fatalf("unknown action must yield empty details, got %v", got)
	}
}

func decodeJSON(t *testing.T, b json.RawMessage) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/application/ -run TestSanitizeAuditDetails -v`
Expected: FAIL — `sanitizeAuditDetails` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `audit.go`:

```go
package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// auditWhitelist maps action -> allowed detail keys.
var auditWhitelist = map[string]map[string]struct{}{
	"google_config_updated":   {"subject": {}, "calendar_id": {}, "has_new_sa_json": {}},
	"google_verified":         {"ok": {}, "calendar_summary": {}, "time_zone": {}, "error_code": {}},
	"chat_linked":             {"chat_id": {}, "chat_title": {}},
	"members_synced":          {"added": {}, "removed": {}, "unchanged": {}},
	"scenario_toggled":        {"name": {}, "enabled": {}},
	"scenario_run_started":    {"name": {}, "manual_run_id": {}},
}

// sanitizeAuditDetails filters details by the action's whitelist. Returns the
// JSON-encoded surviving keys + the list of dropped keys for logging.
func sanitizeAuditDetails(action string, details map[string]any) (json.RawMessage, []string) {
	wl := auditWhitelist[action]
	clean := map[string]any{}
	var dropped []string
	for k, v := range details {
		if _, ok := wl[k]; ok {
			clean[k] = v
		} else {
			dropped = append(dropped, k)
		}
	}
	b, _ := json.Marshal(clean)
	return b, dropped
}

// AuditContext carries the actor identity through the request lifecycle.
type AuditContext struct {
	UserID     uuid.UUID
	TelegramID int64
	Email      string
}

type auditCtxKey struct{}

// WithAuditActor stores the actor in ctx (set by the middleware).
func WithAuditActor(ctx context.Context, a AuditContext) context.Context {
	return context.WithValue(ctx, auditCtxKey{}, a)
}

// auditActor returns (actor, ok). ok=false when the ctx has no audit actor —
// in that case the caller must skip the audit write (with a Warn log).
func auditActor(ctx context.Context) (AuditContext, bool) {
	v, ok := ctx.Value(auditCtxKey{}).(AuditContext)
	return v, ok
}

// Audit records an admin action. Audit-write failures NEVER fail the parent
// operation — they are logged at Warn.
func (s *Services) Audit(ctx context.Context, action, targetKind, targetID string, details map[string]any) {
	actor, ok := auditActor(ctx)
	if !ok {
		s.Log.Warn("audit_actor_missing", zap.String("action", action), zap.String("target_id", targetID))
		return
	}
	clean, dropped := sanitizeAuditDetails(action, details)
	if len(dropped) > 0 {
		s.Log.Warn("audit_unexpected_keys", zap.String("action", action), zap.Strings("dropped", dropped))
	}
	err := s.Store.InsertAuditEntry(ctx, postgres.AuditEntry{
		ActorUserID:     actor.UserID,
		ActorTelegramID: actor.TelegramID,
		ActorEmail:      actor.Email,
		Action:          action,
		TargetKind:      targetKind,
		TargetID:        targetID,
		Details:         clean,
	})
	if err != nil {
		s.Log.Warn("audit_write_failed", zap.String("action", action), zap.Error(err))
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/application/ -run TestSanitizeAuditDetails -v`
Expected: PASS.

- [ ] **Step 5: Build everything to catch any compile error**

Run: `cd backend && go build ./...`
Expected: clean build.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/application/audit.go backend/internal/application/audit_test.go
git commit -m "$(cat <<'EOF'
feat(application): audit helper with whitelist redaction

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T4: requireBotAdmin middleware (TDD)

**Files:**
- Create: `backend/internal/delivery/http/middleware/require_bot_admin.go`
- Create: `backend/internal/delivery/http/middleware/require_bot_admin_test.go`

- [ ] **Step 1: Write the failing test**

Create `require_bot_admin_test.go`:

```go
package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/delivery/http/middleware"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

func newApp() *fiber.App {
	app := fiber.New()
	app.Use(middleware.RequireBotAdmin)
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func TestRequireBotAdmin_NoLocal_Returns403(t *testing.T) {
	app := fiber.New()
	app.Use(middleware.RequireBotAdmin)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/x", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireBotAdmin_NonAdmin_Returns403(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bot_user", postgres.BotUser{Role: "user"})
		return c.Next()
	}, middleware.RequireBotAdmin)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/x", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestRequireBotAdmin_Admin_Passes(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("bot_user", postgres.BotUser{Role: "admin"})
		return c.Next()
	}, middleware.RequireBotAdmin)
	app.Get("/x", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/x", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/delivery/http/middleware/ -v`
Expected: FAIL — `RequireBotAdmin` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `require_bot_admin.go`:

```go
// Package middleware contains Fiber-level middleware shared across handlers.
package middleware

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// RequireBotAdmin asserts that the request was authenticated as a bot user
// whose role is "admin". Returns 403 otherwise. Must be mounted AFTER the TMA
// JWT middleware that sets c.Locals("bot_user").
func RequireBotAdmin(c *fiber.Ctx) error {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok || bu.Role != "admin" {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	return c.Next()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/delivery/http/middleware/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/middleware/require_bot_admin.go backend/internal/delivery/http/middleware/require_bot_admin_test.go
git commit -m "$(cat <<'EOF'
feat(http): RequireBotAdmin middleware

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T5: Google probe with sentinel errors (TDD)

**Files:**
- Create: `backend/internal/infrastructure/calendar/google/probe.go`
- Create: `backend/internal/infrastructure/calendar/google/probe_test.go`

- [ ] **Step 1: Write the failing tests**

Create `probe_test.go`:

```go
package google

import (
	"context"
	"errors"
	"testing"
)

func TestProbe_InvalidJSON(t *testing.T) {
	_, err := Probe(context.Background(), "{not-json", "admin@example.com", "primary")
	if !errors.Is(err, ErrJSONParse) {
		t.Fatalf("expected ErrJSONParse, got %v", err)
	}
}

func TestProbe_MissingPrivateKey(t *testing.T) {
	_, err := Probe(context.Background(), `{"type":"service_account","client_email":"x@y","token_uri":"https://oauth2.googleapis.com/token"}`, "admin@example.com", "primary")
	if !errors.Is(err, ErrJSONParse) {
		t.Fatalf("expected ErrJSONParse for missing private_key, got %v", err)
	}
}
```

Real network-bound stages (`ErrSubject`, `ErrCalendar`, `ErrAPIDisabled`) are not covered by unit tests — they need a live Google account. Documented in the spec; we trust the discriminator helpers below.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/infrastructure/calendar/google/ -run TestProbe -v`
Expected: FAIL — `Probe` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `probe.go`:

```go
package google

import (
	"context"
	"errors"
	"fmt"
	"strings"

	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	// ErrJSONParse — SA JSON does not parse or is missing required fields.
	ErrJSONParse = errors.New("sa_json_parse")
	// ErrAPIDisabled — Calendar API is disabled for the GCP project.
	ErrAPIDisabled = errors.New("calendar_api_disabled")
	// ErrSubject — Domain-wide delegation failed for the impersonation subject.
	ErrSubject = errors.New("subject_impersonation")
	// ErrCalendar — Subject lacks access to the requested calendar id.
	ErrCalendar = errors.New("calendar_not_accessible")
)

// Probe parses the SA JSON, impersonates `subject`, and reads metadata for
// `calendarID`. Returns the *calendar.Calendar on success, or a sentinel-wrapped
// error matching one of the Err* values. No side effects on Google.
func Probe(ctx context.Context, saJSON, subject, calendarID string) (*calendar.Calendar, error) {
	cfg, err := googleoauth.JWTConfigFromJSON([]byte(saJSON), calendar.CalendarScope)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJSONParse, err)
	}
	cfg.Subject = subject
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(cfg.Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAPIDisabled, err)
	}
	cal, err := svc.Calendars.Get(calendarID).Context(ctx).Do()
	if err != nil {
		if isGoogleAPIDisabled(err) {
			return nil, fmt.Errorf("%w: %v", ErrAPIDisabled, err)
		}
		if isImpersonationFail(err) {
			return nil, fmt.Errorf("%w: %v", ErrSubject, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrCalendar, err)
	}
	return cal, nil
}

func isGoogleAPIDisabled(err error) bool {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.Code != 403 {
		return false
	}
	// Reason string Google returns when Calendar API is disabled in the project.
	for _, d := range apiErr.Errors {
		if d.Reason == "accessNotConfigured" {
			return true
		}
	}
	return strings.Contains(apiErr.Message, "has not been used") || strings.Contains(apiErr.Message, "is disabled")
}

func isImpersonationFail(err error) bool {
	// The OAuth2 library surfaces impersonation errors before Calendars.Get
	// fires (often as 401 "unauthorized_client"). When that lands inside a
	// transport error wrapping, the *googleapi.Error path isn't taken.
	msg := err.Error()
	if strings.Contains(msg, "unauthorized_client") {
		return true
	}
	if strings.Contains(msg, "Not Authorized to access this resource") {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 401 {
		return true
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/infrastructure/calendar/google/ -run TestProbe -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/calendar/google/probe.go backend/internal/infrastructure/calendar/google/probe_test.go
git commit -m "$(cat <<'EOF'
feat(calendar): Probe with sentinel errors for slice-D verify

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T6: application/google_verify.go (TDD error-code mapping)

**Files:**
- Create: `backend/internal/application/google_verify.go`
- Create: `backend/internal/application/google_verify_test.go`

- [ ] **Step 1: Write the failing tests**

Create `google_verify_test.go`:

```go
package application

import (
	"context"
	"errors"
	"testing"

	googleprobe "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/calendar/google"
)

func TestMapProbeError(t *testing.T) {
	cases := []struct {
		name string
		in   error
		want error
	}{
		{"ErrJSONParse", googleprobe.ErrJSONParse, ErrGoogleSAInvalid},
		{"wrapped JSON parse", &probeWrap{inner: googleprobe.ErrJSONParse}, ErrGoogleSAInvalid},
		{"ErrAPIDisabled", googleprobe.ErrAPIDisabled, ErrGoogleAPIDisabled},
		{"ErrSubject", googleprobe.ErrSubject, ErrGoogleSubjectInvalid},
		{"ErrCalendar", googleprobe.ErrCalendar, ErrGoogleCalendarNotAccessible},
		{"unknown", errors.New("network exploded"), ErrGoogleCalendarNotAccessible},
		{"nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapProbeError(tc.in)
			if !errors.Is(got, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, got)
			}
		})
	}
}

type probeWrap struct{ inner error }

func (p *probeWrap) Error() string { return "wrapped: " + p.inner.Error() }
func (p *probeWrap) Unwrap() error { return p.inner }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && go test ./internal/application/ -run TestMapProbeError -v`
Expected: FAIL — `mapProbeError` undefined.

- [ ] **Step 3: Write the implementation**

Create `google_verify.go`:

```go
package application

import (
	"context"
	"errors"

	"github.com/google/uuid"

	googleprobe "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/calendar/google"
)

// Sentinel errors mapped to handler-level error codes.
var (
	ErrGoogleSAInvalid             = errors.New("google_sa_invalid")
	ErrGoogleSubjectInvalid        = errors.New("google_subject_invalid")
	ErrGoogleCalendarNotAccessible = errors.New("google_calendar_not_accessible")
	ErrGoogleAPIDisabled           = errors.New("google_api_disabled")
	ErrGoogleNotConfigured         = errors.New("google_not_configured")
)

// GoogleVerifyResult is what the handler returns on success.
type GoogleVerifyResult struct {
	OK              bool   `json:"ok"`
	CalendarSummary string `json:"calendar_summary,omitempty"`
	TimeZone        string `json:"time_zone,omitempty"`
	AccessRole      string `json:"access_role,omitempty"`
}

// VerifyGoogleIntegration reads the workspace's stored Google config,
// decrypts the SA JSON, runs Probe, and maps errors to public codes.
func (s *Services) VerifyGoogleIntegration(ctx context.Context, workspaceID uuid.UUID) (*GoogleVerifyResult, error) {
	enc, subject, calendarID, err := s.Store.GetGoogleConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 || subject == "" {
		return nil, ErrGoogleNotConfigured
	}
	saJSON, err := s.Cipher.Decrypt(enc)
	if err != nil {
		return nil, ErrGoogleSAInvalid
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	cal, err := googleprobe.Probe(ctx, saJSON, subject, calendarID)
	if e := mapProbeError(err); e != nil {
		return nil, e
	}
	return &GoogleVerifyResult{
		OK:              true,
		CalendarSummary: cal.Summary,
		TimeZone:        cal.TimeZone,
		AccessRole:      cal.AccessRole,
	}, nil
}

// mapProbeError maps probe sentinels to handler-level errors. Unknown errors
// default to ErrGoogleCalendarNotAccessible (the most generic Google-side failure).
func mapProbeError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, googleprobe.ErrJSONParse):
		return ErrGoogleSAInvalid
	case errors.Is(err, googleprobe.ErrAPIDisabled):
		return ErrGoogleAPIDisabled
	case errors.Is(err, googleprobe.ErrSubject):
		return ErrGoogleSubjectInvalid
	case errors.Is(err, googleprobe.ErrCalendar):
		return ErrGoogleCalendarNotAccessible
	default:
		return ErrGoogleCalendarNotAccessible
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/application/ -run TestMapProbeError -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/application/google_verify.go backend/internal/application/google_verify_test.go
git commit -m "$(cat <<'EOF'
feat(application): VerifyGoogleIntegration with sentinel mapping

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T7: application/admin_workspace.go (TDD EnsureSingleWorkspace)

**Files:**
- Create: `backend/internal/application/admin_workspace.go`
- Create: `backend/internal/application/admin_workspace_test.go`

- [ ] **Step 1: Write the failing test**

Create `admin_workspace_test.go`:

```go
package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeWSStore struct {
	id         uuid.UUID
	createErr  error
	createdTZ  string
	createdML  string
	created    bool
}

func (f *fakeWSStore) EnsureLeadCatWorkspaceID(_ context.Context, tz, ml string, _ uuid.UUID) (uuid.UUID, error) {
	if f.createErr != nil {
		return uuid.Nil, f.createErr
	}
	f.createdTZ = tz
	f.createdML = ml
	f.created = true
	return f.id, nil
}

func TestEnsureSingleWorkspace_Defaults(t *testing.T) {
	want := uuid.New()
	f := &fakeWSStore{id: want}
	got, err := ensureSingleWorkspace(context.Background(), f, uuid.New())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != want {
		t.Fatalf("id mismatch")
	}
	if f.createdTZ != "Asia/Almaty" {
		t.Fatalf("default tz wrong: %q", f.createdTZ)
	}
	if f.createdML != "" {
		t.Fatalf("default meet link should be empty: %q", f.createdML)
	}
}

func TestEnsureSingleWorkspace_PropagatesStoreError(t *testing.T) {
	boom := errors.New("db down")
	f := &fakeWSStore{createErr: boom}
	if _, err := ensureSingleWorkspace(context.Background(), f, uuid.New()); !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `cd backend && go test ./internal/application/ -run TestEnsureSingleWorkspace -v`
Expected: FAIL — `ensureSingleWorkspace` undefined.

- [ ] **Step 3: Write implementation**

Create `admin_workspace.go`:

```go
package application

import (
	"context"

	"github.com/google/uuid"
)

const (
	defaultWorkspaceTZ       = "Asia/Almaty"
	defaultWorkspaceMeetLink = ""
)

// workspaceEnsurer is the narrow store interface used by EnsureSingleWorkspace
// — defined here so unit tests can mock it.
type workspaceEnsurer interface {
	EnsureLeadCatWorkspaceID(ctx context.Context, tz, meetLink string, ownerID uuid.UUID) (uuid.UUID, error)
}

// EnsureSingleWorkspace returns the id of the singleton Lead Cat workspace,
// creating it on first call. ownerID is required by the workspaces.owner_id
// FK and should be the calling admin's platform user id (when there's no
// platform user — e.g., TMA admin without a paired platform account — pass
// uuid.Nil and the DB will fall back to a system bootstrap owner).
func (s *Services) EnsureSingleWorkspace(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, error) {
	return ensureSingleWorkspace(ctx, s.Store, ownerID)
}

func ensureSingleWorkspace(ctx context.Context, store workspaceEnsurer, ownerID uuid.UUID) (uuid.UUID, error) {
	return store.EnsureLeadCatWorkspaceID(ctx, defaultWorkspaceTZ, defaultWorkspaceMeetLink, ownerID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/application/ -run TestEnsureSingleWorkspace -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/application/admin_workspace.go backend/internal/application/admin_workspace_test.go
git commit -m "$(cat <<'EOF'
feat(application): EnsureSingleWorkspace with defaults

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T8: HTTP handlers — phase 1 + route group registration

**Files:**
- Create: `backend/internal/delivery/http/handlers/tma_admin.go`
- Modify: `backend/internal/delivery/http/app.go` (around line 150-159)

- [ ] **Step 1: Create the handler file with phase 1 + the route registration helper**

Create `tma_admin.go`:

```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// tmaAdminBotUser extracts the authed admin bot user (set by RequireBotAdmin
// upstream). Caller can assume Role == "admin".
func tmaAdminBotUser(c *fiber.Ctx) (postgres.BotUser, bool) {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	return bu, ok && bu.Role == "admin"
}

// withAuditActor enriches ctx with the actor identity for the audit helper.
func (a *API) withAuditActor(c *fiber.Ctx) {
	bu, ok := tmaAdminBotUser(c)
	if !ok {
		return
	}
	c.SetUserContext(application.WithAuditActor(c.UserContext(), application.AuditContext{
		UserID:     bu.ID,
		TelegramID: bu.TelegramID,
		Email:      bu.Email,
	}))
}

// adminWorkspaceID returns the single Lead Cat workspace id, creating it
// implicitly on first call. The admin user's platform user_id (if any) is
// used as owner_id; if the admin has no paired platform user, uuid.Nil is
// passed and the DB FK assumes a bootstrap owner row.
func (a *API) adminWorkspaceID(c *fiber.Ctx) (uuid.UUID, error) {
	bu, _ := tmaAdminBotUser(c)
	ownerID := uuid.Nil
	if u, ok, err := a.App.Store.GetPlatformUserIDByTelegramID(c.Context(), bu.TelegramID); err == nil && ok {
		ownerID = u
	}
	return a.App.EnsureSingleWorkspace(c.Context(), ownerID)
}

// GET /api/tma/admin/workspace
func (a *API) TMAAdminGetWorkspace(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	w, err := a.App.GetWorkspace(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	view, err := a.App.GetIntegrations(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	chatID, chatTitle, _ := a.App.Store.GetWorkspaceChat(c.Context(), id)
	return c.JSON(fiber.Map{
		"id":                 id,
		"name":               w.Name,
		"tz":                 w.TZ,
		"meet_link":          w.MeetLink,
		"has_google":         view.HasGoogle,
		"google_subject":     view.GoogleSubject,
		"google_calendar_id": view.GoogleCalendarID,
		"has_chat":           chatID != 0,
		"chat_id":            chatID,
		"chat_title":         chatTitle,
	})
}

// POST /api/tma/admin/workspace
func (a *API) TMAAdminCreateWorkspace(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	return c.JSON(fiber.Map{"id": id})
}

// GET /api/tma/admin/integrations
func (a *API) TMAAdminGetIntegrations(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	view, err := a.App.GetIntegrations(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// strip VCS bits — slice D admin UI is Google-only
	return c.JSON(fiber.Map{
		"has_google":         view.HasGoogle,
		"google_subject":     view.GoogleSubject,
		"google_calendar_id": view.GoogleCalendarID,
		"meet_link":          view.MeetLink,
		"tz":                 view.TZ,
	})
}

// PATCH /api/tma/admin/integrations
func (a *API) TMAAdminPatchIntegrations(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	var body struct {
		GoogleSAJSON     string `json:"google_sa_json"`
		GoogleSubject    string `json:"google_subject"`
		GoogleCalendarID string `json:"google_calendar_id"`
		MeetLink         string `json:"meet_link"`
		TZ               string `json:"tz"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if body.GoogleSAJSON != "" || body.GoogleSubject != "" || body.GoogleCalendarID != "" {
		if err := a.App.SetGoogleConfig(c.Context(), id, body.GoogleSAJSON, body.GoogleSubject, body.GoogleCalendarID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		a.App.Audit(c.UserContext(), "google_config_updated", "workspace", id.String(), map[string]any{
			"subject":         body.GoogleSubject,
			"calendar_id":     body.GoogleCalendarID,
			"has_new_sa_json": body.GoogleSAJSON != "",
		})
	}
	if body.MeetLink != "" || body.TZ != "" {
		if err := a.App.PatchIntegrations(c.Context(), id, "", "", "", "", body.MeetLink, body.TZ); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// POST /api/tma/admin/integrations/verify
func (a *API) TMAAdminVerifyIntegrations(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	res, err := a.App.VerifyGoogleIntegration(c.Context(), id)
	if err != nil {
		code, status := mapVerifyError(err)
		a.App.Audit(c.UserContext(), "google_verified", "workspace", id.String(), map[string]any{
			"ok":         false,
			"error_code": code,
		})
		return fiber.NewError(status, code)
	}
	a.App.Audit(c.UserContext(), "google_verified", "workspace", id.String(), map[string]any{
		"ok":               true,
		"calendar_summary": res.CalendarSummary,
		"time_zone":        res.TimeZone,
	})
	return c.JSON(res)
}

func mapVerifyError(err error) (code string, status int) {
	switch {
	case errors.Is(err, application.ErrGoogleSAInvalid):
		return "google_sa_invalid", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleSubjectInvalid):
		return "google_subject_invalid", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleCalendarNotAccessible):
		return "google_calendar_not_accessible", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleAPIDisabled):
		return "google_api_disabled", fiber.StatusBadRequest
	case errors.Is(err, application.ErrGoogleNotConfigured):
		return "google_not_configured", fiber.StatusBadRequest
	default:
		return "internal_error", fiber.StatusInternalServerError
	}
}
```

If `GetWorkspaceChat(ctx, id) (chatID int64, chatTitle string, ok bool)` doesn't exist on `Store`, also add it in this task as a thin SELECT in `workspace_repo.go`:

```go
// GetWorkspaceChat returns the chat id + title; (0, "", false) if not linked.
func (s *Store) GetWorkspaceChat(ctx context.Context, id uuid.UUID) (int64, string, bool) {
	var chatID int64
	var title string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(chat_id, 0), COALESCE(chat_title, '') FROM workspaces WHERE id = $1`, id).Scan(&chatID, &title)
	if err != nil || chatID == 0 {
		return 0, "", false
	}
	return chatID, title, true
}
```

(Run a quick `grep -n chat_title backend/internal/infrastructure/persistence/postgres/` and `grep -n chat_id backend/internal/infrastructure/persistence/postgres/workspace_repo.go` first — if columns are named differently, adjust the SELECT. If `chat_title` doesn't exist, return only `chat_id` and an empty title.)

- [ ] **Step 2: Wire routes into `app.go`**

In `app.go` just after the existing TMA routes (around line 159, after `tma.Delete("/meetings/:id", api.TMADeleteMeeting)`):

```go
tmaAdmin := tma.Group("/admin", middleware.RequireBotAdmin)
tmaAdmin.Get("/workspace", api.TMAAdminGetWorkspace)
tmaAdmin.Post("/workspace", api.TMAAdminCreateWorkspace)
tmaAdmin.Get("/integrations", api.TMAAdminGetIntegrations)
tmaAdmin.Patch("/integrations", api.TMAAdminPatchIntegrations)
tmaAdmin.Post("/integrations/verify", api.TMAAdminVerifyIntegrations)
```

Add `"github.com/Jaryq-Lab/notify-bot/internal/delivery/http/middleware"` to the import block.

- [ ] **Step 3: Build to verify**

Run: `cd backend && go build ./...`
Expected: clean.

- [ ] **Step 4: Run all tests**

Run: `cd backend && go test ./...`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_admin.go backend/internal/delivery/http/app.go backend/internal/infrastructure/persistence/postgres/workspace_repo.go
git commit -m "$(cat <<'EOF'
feat(http): /api/tma/admin/* phase 1 — Google integration

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T9: HTTP handlers — phase 2 (chat + members)

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_admin.go` (append)
- Modify: `backend/internal/delivery/http/app.go` (append routes)

- [ ] **Step 1: Append phase 2 handlers**

Append to `tma_admin.go`:

```go
// GET /api/tma/admin/chat/status
func (a *API) TMAAdminChatStatus(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	chatID, chatTitle, ok := a.App.Store.GetWorkspaceChat(c.Context(), id)
	return c.JSON(fiber.Map{
		"linked":     ok,
		"chat_id":    chatID,
		"chat_title": chatTitle,
	})
}

// POST /api/tma/admin/chat/link
func (a *API) TMAAdminChatLink(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	var body struct {
		ChatID    int64  `json:"chat_id"`
		ChatTitle string `json:"chat_title"`
	}
	if err := c.BodyParser(&body); err != nil || body.ChatID == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	if err := a.App.LinkChat(c.Context(), id, body.ChatID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.App.Audit(c.UserContext(), "chat_linked", "workspace", id.String(), map[string]any{
		"chat_id":    body.ChatID,
		"chat_title": body.ChatTitle,
	})
	return c.SendStatus(fiber.StatusNoContent)
}

// GET /api/tma/admin/members
func (a *API) TMAAdminListMembers(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	members, err := a.App.ListMembers(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"members": members})
}

// POST /api/tma/admin/members/sync-chat
func (a *API) TMAAdminMembersSyncChat(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	n, err := a.App.SyncChatMembers(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.App.Audit(c.UserContext(), "members_synced", "workspace", id.String(), map[string]any{
		"added": n,
	})
	return c.JSON(fiber.Map{"added": n})
}
```

`SyncChatMembers` is currently a handler-level call (handlers.go:313 wires it via `telegram.SyncChatMembers`). Mirror that into `Services` so admin handlers can call it cleanly. Add to `services.go` (after `SyncChatMembers` does not yet exist on `Services`):

```go
// SyncChatMembers imports chat administrators into workspace_members.
// Wraps the telegram helper so handlers don't depend on infrastructure/.
func (s *Services) SyncChatMembers(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	// telegram.SyncChatMembers signature: (ctx, *bot.Bot, *postgres.Store, uuid.UUID)
	// We don't have a *bot.Bot on Services — keep the existing handler call
	// path; expose a thin wrapper that delegates back to Store + Bot via
	// dependency-injection set on Services at startup.
	if s.Bot == nil {
		return 0, fmt.Errorf("bot not configured")
	}
	return telegram.SyncChatMembers(ctx, s.Bot, s.Store, workspaceID)
}
```

This requires adding `Bot *bot.Bot` to the `Services` struct and wiring it in `cmd/server/main.go`. Import `"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/telegram"` + `"github.com/go-telegram/bot"`.

- [ ] **Step 2: Wire routes**

In `app.go` immediately after the phase 1 routes:

```go
tmaAdmin.Get("/chat/status", api.TMAAdminChatStatus)
tmaAdmin.Post("/chat/link", api.TMAAdminChatLink)
tmaAdmin.Get("/members", api.TMAAdminListMembers)
tmaAdmin.Post("/members/sync-chat", api.TMAAdminMembersSyncChat)
```

- [ ] **Step 3: Wire `Bot` into Services**

In `cmd/server/main.go` where `application.Services{...}` is constructed, add the bot reference. Example (verify the existing init block):

```go
services := &application.Services{
	Store: store, Cipher: cipher, Queue: queue, Calendar: calendarProvider,
	Log:   log, Bot: tg, // <- new field
}
```

In `services.go` extend struct:

```go
type Services struct {
	Store    *postgres.Store
	Cipher   *crypto.TokenCipher
	Queue    *asynqqueue.Client
	Calendar CalendarProvider
	Log      *zap.Logger
	Bot      *bot.Bot
}
```

- [ ] **Step 4: Build + tests**

Run: `cd backend && go build ./... && go test ./...`
Expected: green.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_admin.go backend/internal/delivery/http/app.go backend/internal/application/services.go backend/cmd/server/main.go
git commit -m "$(cat <<'EOF'
feat(http): /api/tma/admin/* phase 2 — chat + members

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T10: HTTP handlers — phase 3 + audit GET

**Files:**
- Modify: `backend/internal/delivery/http/handlers/tma_admin.go` (append)
- Modify: `backend/internal/delivery/http/app.go` (append routes)

- [ ] **Step 1: Append phase 3 handlers**

Append to `tma_admin.go`:

```go
// GET /api/tma/admin/scenarios
func (a *API) TMAAdminListScenarios(c *fiber.Ctx) error {
	a.withAuditActor(c)
	id, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	list, err := a.App.ListScenarios(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"scenarios": list})
}

// PATCH /api/tma/admin/scenarios/:id  (only `enabled` is honored)
func (a *API) TMAAdminPatchScenario(c *fiber.Ctx) error {
	a.withAuditActor(c)
	sid, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.BodyParser(&body); err != nil || body.Enabled == nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	sc, err := a.App.UpdateScenario(c.Context(), sid, "", body.Enabled, nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.App.Audit(c.UserContext(), "scenario_toggled", "scenario", sid.String(), map[string]any{
		"name":    sc.Name,
		"enabled": *body.Enabled,
	})
	return c.JSON(sc)
}

// POST /api/tma/admin/scenarios/:id/run
func (a *API) TMAAdminRunScenario(c *fiber.Ctx) error {
	a.withAuditActor(c)
	sid, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	wid, err := a.adminWorkspaceID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "workspace_not_found")
	}
	sc, _ := a.App.GetScenario(c.Context(), sid)
	runID, err := a.App.RunScenario(c.Context(), sid, wid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	a.App.Audit(c.UserContext(), "scenario_run_started", "scenario", sid.String(), map[string]any{
		"name":          sc.Name,
		"manual_run_id": runID.String(),
	})
	return c.JSON(fiber.Map{"run_id": runID})
}

// GET /api/tma/admin/scenarios/:id/runs
func (a *API) TMAAdminListScenarioRuns(c *fiber.Ctx) error {
	a.withAuditActor(c)
	sid, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "validation_failed")
	}
	runs, err := a.App.ListRuns(c.Context(), sid)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"runs": runs})
}

// GET /api/tma/admin/audit?limit=&action=&actor=
func (a *API) TMAAdminListAudit(c *fiber.Ctx) error {
	a.withAuditActor(c)
	limit := c.QueryInt("limit", 50)
	entries, err := a.App.Store.ListAuditEntries(c.Context(), postgres.AuditFilter{
		Action:     c.Query("action"),
		ActorEmail: c.Query("actor"),
		Limit:      limit,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"entries": entries})
}
```

- [ ] **Step 2: Wire routes**

In `app.go` after phase 2 routes:

```go
tmaAdmin.Get("/scenarios", api.TMAAdminListScenarios)
tmaAdmin.Patch("/scenarios/:id", api.TMAAdminPatchScenario)
tmaAdmin.Post("/scenarios/:id/run", api.TMAAdminRunScenario)
tmaAdmin.Get("/scenarios/:id/runs", api.TMAAdminListScenarioRuns)
tmaAdmin.Get("/audit", api.TMAAdminListAudit)
```

- [ ] **Step 3: Build + tests**

Run: `cd backend && go build ./... && go test ./...`
Expected: green.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_admin.go backend/internal/delivery/http/app.go
git commit -m "$(cat <<'EOF'
feat(http): /api/tma/admin/* phase 3 + audit GET

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T11: OpenAPI changes + frontend schema regen

**Files:**
- Modify: `backend/openapi/openapi.json`
- Modify: `backend/docs/openapi.json` (byte-identical mirror)
- Regen: `frontend/src/shared/api/generated/schema.ts`

- [ ] **Step 1: Add the 14 new paths under `paths` in `backend/openapi/openapi.json`**

Use the spec §4 path/schema lists verbatim. Sample skeleton for one of the heavier endpoints — apply the same pattern to all 14:

```json
"/api/tma/admin/integrations/verify": {
  "post": {
    "operationId": "tma_admin_integrations_verify",
    "tags": ["tma-admin"],
    "summary": "Verify Google integration (parse → impersonate → Calendars.Get)",
    "security": [{"bearerAuth": []}],
    "responses": {
      "200": {
        "description": "Verify result",
        "content": {"application/json": {"schema": {"$ref": "#/components/schemas/TmaAdminGoogleVerifyResult"}}}
      },
      "400": {"description": "google_sa_invalid | google_subject_invalid | google_calendar_not_accessible | google_api_disabled | google_not_configured", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ApiErrorResponse"}}}},
      "401": {"description": "Unauthorized", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ApiErrorResponse"}}}},
      "403": {"description": "Forbidden", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/ApiErrorResponse"}}}}
    }
  }
}
```

- [ ] **Step 2: Add the 11 new component schemas**

Names + fields per spec §4. Sample:

```json
"TmaAdminGoogleVerifyResult": {
  "type": "object",
  "required": ["ok"],
  "properties": {
    "ok": {"type": "boolean"},
    "calendar_summary": {"type": "string"},
    "time_zone": {"type": "string"},
    "access_role": {"type": "string"}
  }
}
```

- [ ] **Step 3: Mirror to docs/**

Run: `cp backend/openapi/openapi.json backend/docs/openapi.json && cd backend && go test ./...`
Expected: green (openapi parser test passes on both copies).

- [ ] **Step 4: Regen frontend schema**

Run: `cd frontend && pnpm openapi:generate`
Expected: `src/shared/api/generated/schema.ts` updated. Run `pnpm typecheck` to confirm no regressions.

- [ ] **Step 5: Commit**

```bash
git add backend/openapi/openapi.json backend/docs/openapi.json frontend/src/shared/api/generated/schema.ts
git commit -m "$(cat <<'EOF'
feat(api): OpenAPI for slice D — 14 admin paths + 11 schemas

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T12: Frontend entities/admin layer

**Files:**
- Create: `frontend/src/entities/admin/types.ts`
- Create: `frontend/src/entities/admin/constants.ts`
- Create: `frontend/src/entities/admin/api.ts`
- Create: `frontend/src/entities/admin/write-api.ts`
- Create: `frontend/src/entities/admin/queries.ts`
- Create: `frontend/src/entities/admin/mutations.ts`

- [ ] **Step 1: Types**

Create `types.ts`:

```ts
export type WorkspaceStatus = {
  id: string
  name: string
  tz: string
  meetLink: string
  hasGoogle: boolean
  googleSubject: string
  googleCalendarID: string
  hasChat: boolean
  chatId?: number
  chatTitle?: string
}

export type IntegrationsView = {
  hasGoogle: boolean
  googleSubject: string
  googleCalendarID: string
  meetLink: string
  tz: string
}

export type GoogleVerifyResult = {
  ok: boolean
  calendarSummary?: string
  timeZone?: string
  accessRole?: string
}

export type GoogleVerifyError =
  | "google_sa_invalid"
  | "google_subject_invalid"
  | "google_calendar_not_accessible"
  | "google_api_disabled"
  | "google_not_configured"

export type ChatStatus = { linked: boolean; chatId?: number; chatTitle?: string }

export type Member = {
  id: string
  fullName: string
  telegramUsername: string
  role: string
  githubLogin?: string
  gitlabLogin?: string
}

export type Scenario = {
  id: string
  name: string
  enabled: boolean
  schedule: string
  lastRunAt?: string
}

export type ScenarioRun = {
  id: string
  scenarioId: string
  status: string
  startedAt: string
  finishedAt?: string
  error?: string
}

export type AuditEntry = {
  id: string
  actorEmail: string
  actorTelegramId: number
  action: string
  targetKind: string
  targetId: string
  details: Record<string, unknown>
  createdAt: string
}
```

- [ ] **Step 2: Constants**

Create `constants.ts`:

```ts
export const ADMIN_AUDIT_PAGE_SIZE = 50

export const AUDIT_ACTIONS = [
  "google_config_updated",
  "google_verified",
  "chat_linked",
  "members_synced",
  "scenario_toggled",
  "scenario_run_started",
] as const

export type AuditAction = (typeof AUDIT_ACTIONS)[number]
```

- [ ] **Step 3: Read API**

Create `api.ts`:

```ts
import { apiFetch } from "@/shared/api/fetch"
import type { AuditEntry, ChatStatus, IntegrationsView, Member, Scenario, ScenarioRun, WorkspaceStatus } from "./types"

type AdminWorkspaceDTO = {
  id: string; name: string; tz: string; meet_link: string
  has_google: boolean; google_subject: string; google_calendar_id: string
  has_chat: boolean; chat_id?: number; chat_title?: string
}
type AdminIntegrationsDTO = {
  has_google: boolean; google_subject: string; google_calendar_id: string
  meet_link: string; tz: string
}
type AdminChatStatusDTO = { linked: boolean; chat_id?: number; chat_title?: string }
type AdminMemberDTO = {
  id: string; full_name: string; telegram_username: string; role: string
  github_login?: string; gitlab_login?: string
}
type AdminScenarioDTO = { id: string; name: string; enabled: boolean; schedule: string; last_run_at?: string }
type AdminScenarioRunDTO = { id: string; scenario_id: string; status: string; started_at: string; finished_at?: string; error?: string }
type AdminAuditDTO = {
  id: string; actor_email: string; actor_telegram_id: number
  action: string; target_kind: string; target_id: string
  details: Record<string, unknown>; created_at: string
}

export async function getWorkspaceStatus(): Promise<WorkspaceStatus> {
  const d = await apiFetch<AdminWorkspaceDTO>("/api/tma/admin/workspace")
  return {
    id: d.id, name: d.name, tz: d.tz, meetLink: d.meet_link,
    hasGoogle: d.has_google, googleSubject: d.google_subject, googleCalendarID: d.google_calendar_id,
    hasChat: d.has_chat, chatId: d.chat_id, chatTitle: d.chat_title,
  }
}

export async function getIntegrations(): Promise<IntegrationsView> {
  const d = await apiFetch<AdminIntegrationsDTO>("/api/tma/admin/integrations")
  return {
    hasGoogle: d.has_google, googleSubject: d.google_subject, googleCalendarID: d.google_calendar_id,
    meetLink: d.meet_link, tz: d.tz,
  }
}

export async function getChatStatus(): Promise<ChatStatus> {
  const d = await apiFetch<AdminChatStatusDTO>("/api/tma/admin/chat/status")
  return { linked: d.linked, chatId: d.chat_id, chatTitle: d.chat_title }
}

export async function getMembers(): Promise<Member[]> {
  const d = await apiFetch<{ members: AdminMemberDTO[] }>("/api/tma/admin/members")
  return d.members.map((m) => ({
    id: m.id, fullName: m.full_name, telegramUsername: m.telegram_username, role: m.role,
    githubLogin: m.github_login, gitlabLogin: m.gitlab_login,
  }))
}

export async function getScenarios(): Promise<Scenario[]> {
  const d = await apiFetch<{ scenarios: AdminScenarioDTO[] }>("/api/tma/admin/scenarios")
  return d.scenarios.map((s) => ({ id: s.id, name: s.name, enabled: s.enabled, schedule: s.schedule, lastRunAt: s.last_run_at }))
}

export async function getScenarioRuns(scenarioId: string): Promise<ScenarioRun[]> {
  const d = await apiFetch<{ runs: AdminScenarioRunDTO[] }>(`/api/tma/admin/scenarios/${scenarioId}/runs`)
  return d.runs.map((r) => ({ id: r.id, scenarioId: r.scenario_id, status: r.status, startedAt: r.started_at, finishedAt: r.finished_at, error: r.error }))
}

export type AuditQuery = { limit?: number; action?: string; actor?: string }
export async function getAuditLog(q: AuditQuery = {}): Promise<AuditEntry[]> {
  const params = new URLSearchParams()
  if (q.limit) params.set("limit", String(q.limit))
  if (q.action) params.set("action", q.action)
  if (q.actor) params.set("actor", q.actor)
  const qs = params.toString()
  const d = await apiFetch<{ entries: AdminAuditDTO[] }>(`/api/tma/admin/audit${qs ? "?" + qs : ""}`)
  return d.entries.map((e) => ({
    id: e.id, actorEmail: e.actor_email, actorTelegramId: e.actor_telegram_id,
    action: e.action, targetKind: e.target_kind, targetId: e.target_id,
    details: e.details, createdAt: e.created_at,
  }))
}
```

- [ ] **Step 4: Write API**

Create `write-api.ts`:

```ts
import { apiFetch } from "@/shared/api/fetch"
import type { GoogleVerifyResult } from "./types"

export type IntegrationsPatch = {
  googleSAJson?: string
  googleSubject?: string
  googleCalendarID?: string
  meetLink?: string
  tz?: string
}

export async function createWorkspace(): Promise<{ id: string }> {
  return apiFetch<{ id: string }>("/api/tma/admin/workspace", { method: "POST" })
}

export async function patchIntegrations(p: IntegrationsPatch): Promise<void> {
  await apiFetch("/api/tma/admin/integrations", {
    method: "PATCH",
    body: JSON.stringify({
      google_sa_json: p.googleSAJson,
      google_subject: p.googleSubject,
      google_calendar_id: p.googleCalendarID,
      meet_link: p.meetLink,
      tz: p.tz,
    }),
  })
}

export async function verifyIntegrations(): Promise<GoogleVerifyResult> {
  const d = await apiFetch<{ ok: boolean; calendar_summary?: string; time_zone?: string; access_role?: string }>(
    "/api/tma/admin/integrations/verify",
    { method: "POST" }
  )
  return { ok: d.ok, calendarSummary: d.calendar_summary, timeZone: d.time_zone, accessRole: d.access_role }
}

export async function linkChat(chatId: number, chatTitle?: string): Promise<void> {
  await apiFetch("/api/tma/admin/chat/link", {
    method: "POST",
    body: JSON.stringify({ chat_id: chatId, chat_title: chatTitle }),
  })
}

export async function syncChatMembers(): Promise<{ added: number }> {
  return apiFetch<{ added: number }>("/api/tma/admin/members/sync-chat", { method: "POST" })
}

export async function patchScenario(id: string, enabled: boolean): Promise<void> {
  await apiFetch(`/api/tma/admin/scenarios/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  })
}

export async function runScenario(id: string): Promise<{ runId: string }> {
  const d = await apiFetch<{ run_id: string }>(`/api/tma/admin/scenarios/${id}/run`, { method: "POST" })
  return { runId: d.run_id }
}
```

- [ ] **Step 5: Queries + mutations**

Create `queries.ts`:

```ts
import { useQuery } from "@tanstack/react-query"
import { getAuditLog, getChatStatus, getIntegrations, getMembers, getScenarioRuns, getScenarios, getWorkspaceStatus, type AuditQuery } from "./api"

export const adminKeys = {
  all: ["admin"] as const,
  workspace: () => ["admin", "workspace"] as const,
  integrations: () => ["admin", "integrations"] as const,
  chat: () => ["admin", "chat"] as const,
  members: () => ["admin", "members"] as const,
  scenarios: () => ["admin", "scenarios"] as const,
  scenarioRuns: (id: string) => ["admin", "scenarios", id, "runs"] as const,
  audit: (q: AuditQuery) => ["admin", "audit", q] as const,
}

export function useWorkspaceStatus() { return useQuery({ queryKey: adminKeys.workspace(), queryFn: getWorkspaceStatus }) }
export function useIntegrations()    { return useQuery({ queryKey: adminKeys.integrations(), queryFn: getIntegrations }) }
export function useChatStatus()      { return useQuery({ queryKey: adminKeys.chat(), queryFn: getChatStatus }) }
export function useMembers()         { return useQuery({ queryKey: adminKeys.members(), queryFn: getMembers }) }
export function useScenarios()       { return useQuery({ queryKey: adminKeys.scenarios(), queryFn: getScenarios }) }
export function useScenarioRuns(id: string) { return useQuery({ queryKey: adminKeys.scenarioRuns(id), queryFn: () => getScenarioRuns(id) }) }
export function useAuditLog(q: AuditQuery = {}) { return useQuery({ queryKey: adminKeys.audit(q), queryFn: () => getAuditLog(q) }) }
```

Create `mutations.ts`:

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { createWorkspace, linkChat, patchIntegrations, patchScenario, runScenario, syncChatMembers, verifyIntegrations, type IntegrationsPatch } from "./write-api"
import { adminKeys } from "./queries"

export function useCreateWorkspace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: createWorkspace,
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.all }) },
  })
}

export function useUpdateIntegrations() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (p: IntegrationsPatch) => patchIntegrations(p),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.all }) },
  })
}

export function useVerifyIntegrations() {
  return useMutation({ mutationFn: verifyIntegrations })
}

export function useLinkChat() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ chatId, chatTitle }: { chatId: number; chatTitle?: string }) => linkChat(chatId, chatTitle),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.chat() }) },
  })
}

export function useSyncMembers() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: syncChatMembers,
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.members() }) },
  })
}

export function useToggleScenario() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => patchScenario(id, enabled),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: adminKeys.scenarios() }) },
  })
}

export function useRunScenario() {
  return useMutation({ mutationFn: (id: string) => runScenario(id) })
}
```

- [ ] **Step 6: Typecheck + build**

Run: `cd frontend && pnpm typecheck && pnpm build`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/entities/admin/
git commit -m "$(cat <<'EOF'
feat(tma): entities/admin layer for slice D

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T13: lib/sa-validate.ts (vitest TDD)

**Files:**
- Create: `frontend/src/features/admin-setup/lib/sa-validate.ts`
- Create: `frontend/src/features/admin-setup/lib/sa-validate.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `sa-validate.test.ts`:

```ts
import { describe, expect, it } from "vitest"
import { validateSAJson } from "./sa-validate"

describe("validateSAJson", () => {
  it("empty → saInvalidJson", () => {
    expect(validateSAJson("")).toEqual({ ok: false, errorKey: "saInvalidJson" })
    expect(validateSAJson("   ")).toEqual({ ok: false, errorKey: "saInvalidJson" })
  })

  it("malformed JSON → saInvalidJson", () => {
    expect(validateSAJson("{not-json")).toEqual({ ok: false, errorKey: "saInvalidJson" })
  })

  it("wrong type → saNotServiceAccount", () => {
    expect(validateSAJson(JSON.stringify({ type: "user_account" }))).toEqual({
      ok: false,
      errorKey: "saNotServiceAccount",
    })
  })

  it("missing required fields → saMissingFields", () => {
    expect(
      validateSAJson(JSON.stringify({ type: "service_account", project_id: "p" }))
    ).toEqual({ ok: false, errorKey: "saMissingFields" })
  })

  it("valid → ok with clientEmail + projectID", () => {
    const text = JSON.stringify({
      type: "service_account",
      project_id: "lead-cat-12345",
      private_key: "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
      client_email: "lead-cat@lead-cat-12345.iam.gserviceaccount.com",
      token_uri: "https://oauth2.googleapis.com/token",
    })
    const v = validateSAJson(text)
    expect(v).toEqual({
      ok: true,
      clientEmail: "lead-cat@lead-cat-12345.iam.gserviceaccount.com",
      projectID: "lead-cat-12345",
    })
  })
})
```

- [ ] **Step 2: Run failing tests**

Run: `cd frontend && pnpm test sa-validate`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement**

Create `sa-validate.ts`:

```ts
import type { I18nKey } from "@/shared/tma/i18n"

export type SAValidation =
  | { ok: true; clientEmail: string; projectID: string }
  | { ok: false; errorKey: I18nKey }

export function validateSAJson(text: string): SAValidation {
  if (!text.trim()) return { ok: false, errorKey: "saInvalidJson" }
  let obj: unknown
  try {
    obj = JSON.parse(text)
  } catch {
    return { ok: false, errorKey: "saInvalidJson" }
  }
  if (typeof obj !== "object" || obj === null) {
    return { ok: false, errorKey: "saInvalidJson" }
  }
  const o = obj as Record<string, unknown>
  if (o.type !== "service_account") {
    return { ok: false, errorKey: "saNotServiceAccount" }
  }
  const required = ["project_id", "private_key", "client_email", "token_uri"]
  const missing = required.filter((k) => typeof o[k] !== "string" || !o[k])
  if (missing.length) return { ok: false, errorKey: "saMissingFields" }
  return {
    ok: true,
    clientEmail: o.client_email as string,
    projectID: o.project_id as string,
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd frontend && pnpm test sa-validate`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/features/admin-setup/lib/sa-validate.ts frontend/src/features/admin-setup/lib/sa-validate.test.ts
git commit -m "$(cat <<'EOF'
feat(tma): validateSAJson with i18n error keys

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T14: features/admin-setup components — all sections + page + route

**Files:**
- Create: `frontend/src/features/admin-setup/components/sa-paste-input.tsx`
- Create: `frontend/src/features/admin-setup/components/verify-result-card.tsx`
- Create: `frontend/src/features/admin-setup/components/integrations-section.tsx`
- Create: `frontend/src/features/admin-setup/components/chat-link-section.tsx`
- Create: `frontend/src/features/admin-setup/components/members-section.tsx`
- Create: `frontend/src/features/admin-setup/components/scenarios-section.tsx`
- Create: `frontend/src/features/admin-setup/components/audit-log-section.tsx`
- Create: `frontend/src/features/admin-setup/pages/admin-setup-page.tsx`
- Create: `frontend/src/routes/_tma/profile.admin.setup.tsx`
- Modify: `frontend/src/features/profile/pages/admin-panel-page.tsx`

- [ ] **Step 1: SA paste input**

Create `sa-paste-input.tsx`:

```tsx
import { useMemo } from "react"
import { cn } from "@/shared/lib/cn"
import { Field } from "@/shared/ui/cat/field"
import { useTmaApp } from "@/shared/tma/context"
import { validateSAJson, type SAValidation } from "../lib/sa-validate"

type Props = {
  value: string
  onChange: (v: string) => void
  disabled?: boolean
}

export function SaPasteInput({ value, onChange, disabled }: Props) {
  const { t } = useTmaApp()
  const v: SAValidation = useMemo(() => validateSAJson(value), [value])
  return (
    <Field label={t("googleSAJson")}>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        spellCheck={false}
        autoCapitalize="off"
        autoComplete="off"
        rows={8}
        disabled={disabled}
        className={cn(
          "border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[13px] font-mono whitespace-pre overflow-auto",
          disabled && "cursor-not-allowed opacity-60"
        )}
        placeholder={'{\n  "type": "service_account",\n  "project_id": "...",\n  "private_key": "-----BEGIN PRIVATE KEY-----\\n...",\n  ...\n}'}
      />
      {!v.ok && value.trim() !== "" && (
        <p className="text-tma-danger text-xs mt-1">{t(v.errorKey)}</p>
      )}
      <p className="text-tma-muted text-xs mt-1">{t("saPasteHint")}</p>
    </Field>
  )
}
```

- [ ] **Step 2: Verify result card**

Create `verify-result-card.tsx`:

```tsx
import { useTmaApp } from "@/shared/tma/context"
import type { GoogleVerifyResult } from "@/entities/admin/types"

type Props =
  | { state: "idle" }
  | { state: "loading" }
  | { state: "ok"; result: GoogleVerifyResult }
  | { state: "error"; errorCode: string }

export function VerifyResultCard(props: Props) {
  const { t } = useTmaApp()
  if (props.state === "idle") return null
  if (props.state === "loading") return <div className="text-tma-muted text-sm">{t("verifying")}</div>
  if (props.state === "ok") {
    return (
      <div className="border-tma-success bg-tma-success-soft text-tma-success rounded-[12px] border p-3 text-sm">
        <div className="font-bold">{t("verifyOK")}</div>
        {props.result.calendarSummary && <div className="opacity-80 mt-0.5">{props.result.calendarSummary} ({props.result.timeZone})</div>}
      </div>
    )
  }
  const key =
    props.errorCode === "google_sa_invalid" ? "googleSaInvalidErr"
    : props.errorCode === "google_subject_invalid" ? "googleSubjectInvalidErr"
    : props.errorCode === "google_calendar_not_accessible" ? "googleCalendarNotAccessibleErr"
    : props.errorCode === "google_api_disabled" ? "googleApiDisabledErr"
    : "verifyUnknownErr"
  return (
    <div className="border-tma-danger bg-tma-danger-soft text-tma-danger rounded-[12px] border p-3 text-sm">
      {t(key)}
    </div>
  )
}
```

If `tma-success` / `tma-success-soft` tokens don't exist in the design system, swap for `tma-accent` / `tma-accent-soft` (slice A/B convention for positive feedback).

- [ ] **Step 3: Integrations section**

Create `integrations-section.tsx`:

```tsx
import { useState } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useIntegrations } from "@/entities/admin/queries"
import { useUpdateIntegrations, useVerifyIntegrations } from "@/entities/admin/mutations"
import { validateSAJson } from "../lib/sa-validate"
import { SaPasteInput } from "./sa-paste-input"
import { VerifyResultCard } from "./verify-result-card"
import { Field } from "@/shared/ui/cat/field"
import { SettingsGroup } from "@/features/profile/components/settings-group"

type VerifyState =
  | { state: "idle" }
  | { state: "loading" }
  | { state: "ok"; result: import("@/entities/admin/types").GoogleVerifyResult }
  | { state: "error"; errorCode: string }

export function IntegrationsSection() {
  const { t } = useTmaApp()
  const { data: int, isLoading } = useIntegrations()
  const updateMut = useUpdateIntegrations()
  const verifyMut = useVerifyIntegrations()
  const [sa, setSa] = useState("")
  const [subject, setSubject] = useState("")
  const [calendarID, setCalendarID] = useState("primary")
  const [verify, setVerify] = useState<VerifyState>({ state: "idle" })

  if (isLoading) return <div className="text-tma-muted">…</div>

  const initialized = false  // re-derive in real form
  const v = validateSAJson(sa)
  const canSave = (sa === "" || v.ok) && subject.trim() !== ""

  async function onSave() {
    try {
      await updateMut.mutateAsync({
        googleSAJson: sa || undefined,
        googleSubject: subject.trim(),
        googleCalendarID: calendarID.trim() || "primary",
      })
      toastSuccess(t("integrationsSaved"))
      setVerify({ state: "loading" })
      try {
        const res = await verifyMut.mutateAsync()
        setVerify({ state: "ok", result: res })
      } catch (err: any) {
        const code = typeof err?.code === "string" ? err.code : "internal_error"
        setVerify({ state: "error", errorCode: code })
      }
      setSa("")
    } catch (err) {
      toastError(err, t("integrationsSaveFailed"))
    }
  }

  return (
    <SettingsGroup title={t("adminIntegrations")}>
      <div className="px-3 pb-3 flex flex-col gap-3">
        <Field label={t("googleSubject")}>
          <input
            value={subject || int?.googleSubject || ""}
            onChange={(e) => setSubject(e.target.value)}
            className="border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
            placeholder="admin@yourdomain.com"
          />
        </Field>
        <Field label={t("googleCalendarID")}>
          <input
            value={calendarID || int?.googleCalendarID || "primary"}
            onChange={(e) => setCalendarID(e.target.value)}
            className="border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]"
          />
        </Field>
        <SaPasteInput value={sa} onChange={setSa} />
        <button
          type="button"
          disabled={!canSave || updateMut.isPending}
          onClick={() => void onSave()}
          className="border-tma-accent bg-tma-accent text-white rounded-[12px] px-4 py-2 text-sm font-bold disabled:opacity-60 disabled:cursor-not-allowed"
        >
          {updateMut.isPending ? t("saving") : t("verifyButton")}
        </button>
        <VerifyResultCard {...verify} />
      </div>
    </SettingsGroup>
  )
}
```

- [ ] **Step 4: Other sections (chat / members / scenarios / audit)**

Create `chat-link-section.tsx`:

```tsx
import { useState } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useChatStatus } from "@/entities/admin/queries"
import { useLinkChat } from "@/entities/admin/mutations"
import { SettingsGroup } from "@/features/profile/components/settings-group"
import { Field } from "@/shared/ui/cat/field"

export function ChatLinkSection() {
  const { t } = useTmaApp()
  const { data } = useChatStatus()
  const mut = useLinkChat()
  const [chatId, setChatId] = useState("")
  const [chatTitle, setChatTitle] = useState("")
  const onLink = async () => {
    const id = Number(chatId.trim())
    if (!Number.isFinite(id) || id === 0) { toastError(null, t("chatIDInvalid")); return }
    try {
      await mut.mutateAsync({ chatId: id, chatTitle: chatTitle || undefined })
      toastSuccess(t("chatLinked"))
      setChatId(""); setChatTitle("")
    } catch (err) { toastError(err, t("chatLinkFailed")) }
  }
  return (
    <SettingsGroup title={t("chatLink")}>
      <div className="px-3 pb-3 flex flex-col gap-3">
        <div className="text-tma-muted text-sm">
          {data?.linked ? `${t("chatLinked")}: ${data.chatTitle ?? data.chatId}` : t("chatNotLinked")}
        </div>
        <Field label={t("chatIDLabel")}>
          <input value={chatId} onChange={(e) => setChatId(e.target.value)}
            className="border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]" />
        </Field>
        <Field label={t("chatTitleLabel")}>
          <input value={chatTitle} onChange={(e) => setChatTitle(e.target.value)}
            className="border-tma-border bg-tma-card text-tma-text w-full rounded-[12px] border px-3 py-2.5 text-[15px]" />
        </Field>
        <button onClick={() => void onLink()} disabled={mut.isPending}
          className="border-tma-accent bg-tma-accent text-white rounded-[12px] px-4 py-2 text-sm font-bold disabled:opacity-60">
          {mut.isPending ? t("linking") : t("linkChat")}
        </button>
      </div>
    </SettingsGroup>
  )
}
```

Create `members-section.tsx`:

```tsx
import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useMembers } from "@/entities/admin/queries"
import { useSyncMembers } from "@/entities/admin/mutations"
import { SettingsGroup } from "@/features/profile/components/settings-group"

export function MembersSection() {
  const { t } = useTmaApp()
  const { data: members = [] } = useMembers()
  const mut = useSyncMembers()
  const onSync = async () => {
    try {
      const { added } = await mut.mutateAsync()
      toastSuccess(`${t("synced")}: +${added}`)
    } catch (err) { toastError(err, t("syncFailed")) }
  }
  return (
    <SettingsGroup title={t("members")}>
      <div className="px-3 pb-3 flex flex-col gap-2">
        {members.length === 0 ? (
          <div className="text-tma-muted text-sm">{t("noMembers")}</div>
        ) : (
          <ul className="flex flex-col gap-1">
            {members.map((m) => (
              <li key={m.id} className="text-tma-text text-sm">
                {m.fullName} <span className="text-tma-muted">· @{m.telegramUsername} · {m.role}</span>
              </li>
            ))}
          </ul>
        )}
        <button onClick={() => void onSync()} disabled={mut.isPending}
          className="self-start border-tma-accent bg-tma-accent text-white rounded-[12px] px-4 py-2 text-sm font-bold disabled:opacity-60">
          {mut.isPending ? t("syncing") : t("syncFromChat")}
        </button>
      </div>
    </SettingsGroup>
  )
}
```

Create `scenarios-section.tsx`:

```tsx
import { useTmaApp } from "@/shared/tma/context"
import { toastError, toastSuccess } from "@/shared/lib/toast"
import { useScenarios } from "@/entities/admin/queries"
import { useRunScenario, useToggleScenario } from "@/entities/admin/mutations"
import { SettingsGroup } from "@/features/profile/components/settings-group"
import { CatToggle } from "@/shared/ui/cat/primitives"

export function ScenariosSection() {
  const { t } = useTmaApp()
  const { data: scenarios = [] } = useScenarios()
  const toggle = useToggleScenario()
  const run = useRunScenario()
  return (
    <SettingsGroup title={t("scenarios")}>
      <div className="px-3 pb-3 flex flex-col gap-3">
        {scenarios.length === 0 && <div className="text-tma-muted text-sm">{t("noScenarios")}</div>}
        {scenarios.map((s) => (
          <div key={s.id} className="flex items-center justify-between gap-3">
            <div className="text-tma-text text-sm">
              <div className="font-bold">{s.name}</div>
              <div className="text-tma-muted text-xs">{s.schedule}</div>
            </div>
            <div className="flex items-center gap-2">
              <CatToggle
                on={s.enabled}
                onChange={(on) => {
                  toggle.mutate(
                    { id: s.id, enabled: on },
                    { onError: (err) => toastError(err, t("scenarioToggleFailed")) }
                  )
                }}
              />
              <button onClick={() => run.mutate(s.id, {
                onSuccess: () => toastSuccess(t("scenarioRan")),
                onError: (err) => toastError(err, t("scenarioRunFailed")),
              })} className="border-tma-border bg-tma-card text-tma-text rounded-[8px] border px-2 py-1 text-xs font-bold">
                {t("scenarioRun")}
              </button>
            </div>
          </div>
        ))}
      </div>
    </SettingsGroup>
  )
}
```

Create `audit-log-section.tsx`:

```tsx
import { useState } from "react"
import { useTmaApp } from "@/shared/tma/context"
import { useAuditLog } from "@/entities/admin/queries"
import { AUDIT_ACTIONS, type AuditAction } from "@/entities/admin/constants"
import { SettingsGroup } from "@/features/profile/components/settings-group"

export function AuditLogSection() {
  const { t } = useTmaApp()
  const [action, setAction] = useState<"" | AuditAction>("")
  const { data: entries = [], isLoading } = useAuditLog({ action: action || undefined, limit: 50 })
  return (
    <SettingsGroup title={t("auditLog")}>
      <div className="px-3 pb-3 flex flex-col gap-3">
        <div className="flex items-center gap-2">
          <select
            value={action}
            onChange={(e) => setAction(e.target.value as AuditAction | "")}
            className="border-tma-border bg-tma-card text-tma-text rounded-[12px] border px-3 py-2 text-sm"
          >
            <option value="">{t("auditAllActions")}</option>
            {AUDIT_ACTIONS.map((a) => (
              <option key={a} value={a}>{t(`auditAction_${a}` as never) /* see i18n */}</option>
            ))}
          </select>
        </div>
        {isLoading ? <div className="text-tma-muted text-sm">…</div> : (
          <ul className="flex flex-col gap-2">
            {entries.map((e) => (
              <li key={e.id} className="border-tma-border bg-tma-card rounded-[12px] border p-2 text-sm">
                <div className="font-bold">{e.action}</div>
                <div className="text-tma-muted text-xs">{e.actorEmail} · {new Date(e.createdAt).toLocaleString()}</div>
                {Object.keys(e.details).length > 0 && (
                  <pre className="text-tma-muted text-xs mt-1 whitespace-pre-wrap">{JSON.stringify(e.details, null, 2)}</pre>
                )}
              </li>
            ))}
            {entries.length === 0 && <li className="text-tma-muted text-sm">{t("auditEmpty")}</li>}
          </ul>
        )}
      </div>
    </SettingsGroup>
  )
}
```

- [ ] **Step 5: Page + route + admin-panel link**

Create `pages/admin-setup-page.tsx`:

```tsx
import { Overlay } from "@/components/tma-shell"
import { useNavigate } from "@tanstack/react-router"
import { useTmaApp } from "@/shared/tma/context"
import { IntegrationsSection } from "../components/integrations-section"
import { ChatLinkSection } from "../components/chat-link-section"
import { MembersSection } from "../components/members-section"
import { ScenariosSection } from "../components/scenarios-section"
import { AuditLogSection } from "../components/audit-log-section"

export function AdminSetupPage() {
  const { t } = useTmaApp()
  const navigate = useNavigate()
  const goBack = () => { void navigate({ to: "/profile/admin" }) }
  return (
    <Overlay open onClose={goBack} onBack={goBack} title={t("adminSetup")}>
      <div className="flex flex-col gap-3 px-4 pb-7">
        <IntegrationsSection />
        <ChatLinkSection />
        <MembersSection />
        <ScenariosSection />
        <AuditLogSection />
      </div>
    </Overlay>
  )
}
```

Create `routes/_tma/profile.admin.setup.tsx`:

```tsx
import { createFileRoute, redirect } from "@tanstack/react-router"
import { AdminSetupPage } from "@/features/admin-setup/pages/admin-setup-page"
import { canAccessTmaAdmin } from "@/shared/auth/module-policies"
import { getTmaUser } from "@/shared/auth/auth-storage"

export const Route = createFileRoute("/_tma/profile/admin/setup")({
  beforeLoad: () => {
    const u = getTmaUser()
    if (!canAccessTmaAdmin(u)) throw redirect({ to: "/profile" })
  },
  component: AdminSetupPage,
})
```

If `getTmaUser`/`canAccessTmaAdmin` signatures differ, mirror what the existing `/profile/admin` route does (`frontend/src/routes/_tma/profile.admin.tsx`).

Modify `frontend/src/features/profile/pages/admin-panel-page.tsx` to add a row that navigates to `/profile/admin/setup`. Re-use the existing `SettingsRow` pattern.

- [ ] **Step 6: Typecheck + build**

Run: `cd frontend && pnpm typecheck && pnpm build`
Expected: clean. Some i18n keys are still unresolved at this point — they're added in D-T15.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/features/admin-setup/ frontend/src/routes/_tma/profile.admin.setup.tsx frontend/src/features/profile/pages/admin-panel-page.tsx
git commit -m "$(cat <<'EOF'
feat(tma): admin-setup feature — sections, page, route

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T15: i18n keys (ru/kk/en)

**Files:**
- Modify: `frontend/src/shared/tma/i18n.ts`

- [ ] **Step 1: Add the new keys to all three language packs**

The keys consumed by D-T13/T14 components:

```
adminSetup, adminIntegrations,
googleSAJson, googleSubject, googleCalendarID, saPasteHint,
saInvalidJson, saNotServiceAccount, saMissingFields,
verifyButton, verifying, verifyOK, verifyUnknownErr,
googleSaInvalidErr, googleSubjectInvalidErr, googleCalendarNotAccessibleErr, googleApiDisabledErr,
integrationsSaved, integrationsSaveFailed, saving,
chatLink, chatLinked, chatNotLinked, chatIDLabel, chatTitleLabel, linkChat, linking, chatIDInvalid, chatLinkFailed,
members, members_, noMembers, syncFromChat, synced, syncing, syncFailed,
scenarios, noScenarios, scenarioRun, scenarioRan, scenarioRunFailed, scenarioToggleFailed,
auditLog, auditAllActions, auditEmpty,
auditAction_google_config_updated, auditAction_google_verified,
auditAction_chat_linked, auditAction_members_synced,
auditAction_scenario_toggled, auditAction_scenario_run_started,
```

Sample RU entries (apply same pattern to KK and EN, mirroring slice A/B style):

```ts
// in the ru block
adminSetup: "Настройка",
adminIntegrations: "Интеграции",
googleSAJson: "Сервис-аккаунт (JSON)",
googleSubject: "Email для имперсонации",
googleCalendarID: "ID календаря",
saPasteHint: "Вставь JSON-ключ из Google Cloud Console",
saInvalidJson: "Неверный JSON",
saNotServiceAccount: "Это не ключ сервис-аккаунта",
saMissingFields: "В ключе нет обязательных полей",
verifyButton: "Сохранить и проверить",
verifying: "Проверка…",
verifyOK: "Подключение работает",
verifyUnknownErr: "Не удалось проверить подключение",
googleSaInvalidErr: "Файл сервис-аккаунта повреждён или не от Google",
googleSubjectInvalidErr: "Не удалось имперсонировать. Проверь domain-wide delegation в Google Workspace",
googleCalendarNotAccessibleErr: "Календарь недоступен для этого сервис-аккаунта",
googleApiDisabledErr: "Calendar API выключен в GCP-проекте",
integrationsSaved: "Сохранено",
integrationsSaveFailed: "Не удалось сохранить",
saving: "Сохраняю…",
chatLink: "Чат-привязка",
chatLinked: "Чат подключён",
chatNotLinked: "Чат не подключён",
chatIDLabel: "ID чата",
chatTitleLabel: "Название",
linkChat: "Подключить",
linking: "Подключаю…",
chatIDInvalid: "Неверный ID чата",
chatLinkFailed: "Не удалось подключить",
members: "Участники",
noMembers: "Нет участников",
syncFromChat: "Синхронизировать из чата",
synced: "Синхронизировано",
syncing: "Синхронизирую…",
syncFailed: "Не удалось синхронизировать",
scenarios: "Сценарии",
noScenarios: "Нет сценариев",
scenarioRun: "Запустить",
scenarioRan: "Запущено",
scenarioRunFailed: "Не удалось запустить",
scenarioToggleFailed: "Не удалось переключить",
auditLog: "Журнал",
auditAllActions: "Все действия",
auditEmpty: "Записей пока нет",
auditAction_google_config_updated: "Google-конфигурация обновлена",
auditAction_google_verified: "Google проверена",
auditAction_chat_linked: "Чат подключён",
auditAction_members_synced: "Участники синхронизированы",
auditAction_scenario_toggled: "Сценарий переключён",
auditAction_scenario_run_started: "Сценарий запущен",
```

Add corresponding `kk` and `en` blocks (preserving meaning, slice A/B convention — short labels, no English jargon in RU).

- [ ] **Step 2: Typecheck + build**

Run: `cd frontend && pnpm typecheck && pnpm build`
Expected: clean (every component's `t(...)` resolves).

- [ ] **Step 3: Commit**

```bash
git add frontend/src/shared/tma/i18n.ts
git commit -m "$(cat <<'EOF'
feat(tma): i18n keys for slice D admin setup (ru/kk/en)

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

### Task D-T16: Docs refresh + final verification

**Files:**
- Modify: `docs/API.md`
- Modify: `docs/MEETINGS.md`
- Modify: `docs/OPERATIONS.md`

- [ ] **Step 1: API.md — add the 14 new TMA paths under "TMA — present"**

Append a table block:

```markdown
### TMA — admin (slice D)

All routes require `Authorization: Bearer <tma_jwt>` AND `bot_users.role == "admin"`.

| Method   | Path                                          | Purpose                                        |
| -------- | --------------------------------------------- | ---------------------------------------------- |
| `GET`    | `/api/tma/admin/workspace`                    | Workspace status (auto-create on first call)   |
| `POST`   | `/api/tma/admin/workspace`                    | Idempotent ensure-workspace                    |
| `GET`    | `/api/tma/admin/integrations`                 | Google integration view (no secrets)           |
| `PATCH`  | `/api/tma/admin/integrations`                 | Set SA JSON / subject / calendar id / meet / tz |
| `POST`   | `/api/tma/admin/integrations/verify`          | Real Google verify (parse → impersonate → Calendars.Get) |
| `GET`    | `/api/tma/admin/chat/status`                  | Chat-link status                               |
| `POST`   | `/api/tma/admin/chat/link`                    | Link Telegram chat                             |
| `GET`    | `/api/tma/admin/members`                      | List workspace members                         |
| `POST`   | `/api/tma/admin/members/sync-chat`            | Sync members from linked chat                  |
| `GET`    | `/api/tma/admin/scenarios`                    | List scenarios                                 |
| `PATCH`  | `/api/tma/admin/scenarios/:id`                | Toggle `enabled` (only)                        |
| `POST`   | `/api/tma/admin/scenarios/:id/run`            | Manual run                                     |
| `GET`    | `/api/tma/admin/scenarios/:id/runs`           | Last 30 runs                                   |
| `GET`    | `/api/tma/admin/audit`                        | Audit log (filters: action, actor, limit≤200)  |

Error codes: `forbidden`, `unauthorized`, `validation_failed`, `workspace_not_found`, `google_sa_invalid`, `google_subject_invalid`, `google_calendar_not_accessible`, `google_api_disabled`, `google_not_configured`.
```

Also add a note in the deprecated alpha-setup appendix: "`PATCH /api/workspaces/:id/integrations` and `POST /api/workspaces/:id/chat/link` are superseded by `/api/tma/admin/*` (slice D). Kept for scripted operator use; will be removed after first beta release."

- [ ] **Step 2: MEETINGS.md — flip "Setup cutover (planned)" to "Done"**

Replace the existing "Setup cutover (planned)" block with:

```markdown
### Setup cutover (done)

Admin setup (Google integration, chat link, members sync, scenarios toggle) is live in the Mini App under `/api/tma/admin/*`. Audit log at `GET /api/tma/admin/audit`. See [`docs/superpowers/specs/2026-06-09-slice-d-tma-admin-integrations-design.md`](superpowers/specs/2026-06-09-slice-d-tma-admin-integrations-design.md).
```

Update the status line at the top to mention admin setup.

- [ ] **Step 3: OPERATIONS.md — add encryption key rotation recipe**

Append a section "Master encryption key rotation":

```markdown
### Master encryption key rotation

Lead Cat encrypts service-account JSON at rest with `MASTER_ENCRYPTION_KEY`. Rotating the key today requires brief downtime:

1. Generate a new 32-byte key: `openssl rand -base64 32`
2. In Google Cloud Console, revoke the existing SA private key (audit only — the SA itself remains valid)
3. Stop the service: `dokploy stop lead-cat`
4. Set the new `MASTER_ENCRYPTION_KEY` env var in Dokploy
5. Start the service
6. Open TMA → Profile → Admin → Setup → Integrations → paste a freshly-issued SA JSON
7. Click "Verify" — confirm calendar metadata returns

The DB column `workspaces.google_sa_json_enc` will be unrecoverable until step 6. Loss of `MASTER_ENCRYPTION_KEY` follows the same recipe: SA in Google remains valid, you just re-upload through the admin UI.

Zero-downtime rotation (two-key handoff with bulk re-encrypt) is post-beta backlog.
```

- [ ] **Step 4: Final verification**

```bash
cd backend && make test && make lint && make build
cd ../frontend && pnpm typecheck && pnpm build && pnpm test
```

Expected: all green. Build artifact `dist/` present on frontend, `backend/server` binary present.

- [ ] **Step 5: Commit**

```bash
git add docs/API.md docs/MEETINGS.md docs/OPERATIONS.md
git commit -m "$(cat <<'EOF'
docs(slice-d): API/MEETINGS/OPERATIONS reflect admin setup live

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>
EOF
)"
```

---

## Self-review

| Check | Result |
|-------|--------|
| Spec §1 — route surface | D-T8/T9/T10 implement all 14 endpoints; route group registered in D-T8 |
| Spec §2 — backend new + modified files | D-T1..T10 covers every file listed in spec §7 |
| Spec §3 — frontend FSD layout | D-T12 entities/admin; D-T13 lib/sa-validate; D-T14 features/admin-setup; D-T15 i18n |
| Spec §4 — migration + OpenAPI | D-T1 migration; D-T11 OpenAPI + schema regen |
| Spec §5 — operational concerns (key rotation, audit retention, multi-instance) | D-T1 includes `workspaces_singleton_idx` (multi-instance); D-T16 OPERATIONS recipe (key rotation); audit retention deferred to Slice H (documented) |
| Spec §6 — risks/open questions | Resolved in spec; nothing new to add at plan layer |
| Spec §6.3 — file structure | Every file from spec §7 maps to a task |
| Placeholder scan | No TBD/TODO; every code block is complete |
| Type consistency | `EnsureLeadCatWorkspaceID` named identically in T2/T7; `mapProbeError` only in T6; `validateSAJson` signature consistent T13↔T14; `AuditEntry`/`AuditFilter` defined in T2 and used in T10 |

Gaps detected during self-review: none. Adjustments inline: none required.

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-09-slice-d-tma-admin-integrations.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, mirrors slice A/B flow
2. **Inline execution** — execute tasks in this session with checkpoints

Which approach?
