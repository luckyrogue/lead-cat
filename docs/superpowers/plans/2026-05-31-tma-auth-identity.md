# TMA Authentication & Identity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a Telegram Mini App user authenticate to the backend via `initData` (against `bot_users`) and see their own identity — the foundational slice for wiring the Mini App to the backend.

**Architecture:** A public `POST /api/auth/tma` exchanges `initData` for a short-lived TMA JWT (distinct from the platform `platform_users` JWT). A dedicated TMA middleware guards `/api/tma/*` and resolves the `bot_users` row. `GET /api/tma/me` is the first consumer. Frontend gets an auth bootstrap + gate that replaces the mock `ME` on the three identity-display screens.

**Tech Stack:** Go, Fiber, golang-jwt v5, pgx/Postgres, zap; React, axios, TanStack Router, Vite/pnpm.

**Spec:** `docs/superpowers/specs/2026-05-31-tma-auth-identity-design.md`

## Codebase facts (verified — rely on these, but confirm before editing)

- **Module path:** `github.com/Jaryq-Lab/notify-bot`.
- **Platform auth is GROUP-scoped, not global.** `app.go` does `ap := app.Group("/api", authMW.Middleware)` and `authPub := app.Group("/api/auth")` (no mw). So a new `app.Group("/api/tma", tmaMW)` is NOT covered by the platform middleware — **no skip-list edit needed** (the spec's "add /api/tma/ to the skip list" is unnecessary; ignore it).
- **`handlers.API` struct** (`internal/delivery/http/handlers/handlers.go`): fields `App *application.Services`, `Bot`, `RDB`, `Log`, `TMA *telegram.InitDataValidator`, `Version`. Constructed as a named-field literal in `app.go` (adding fields leaves others zero-valued — compiles). `a.App.Store` is the `*postgres.Store`.
- **initData validator** (`internal/infrastructure/telegram/initdata.go`): `InitDataValidator.Validate(initData) (InitDataUser, error)`; `InitDataUser{ID int64, Username string}`. Verifies HMAC. Does NOT parse/return `auth_date` — this plan adds `AuthDate int64`.
- **JWT** (`internal/platform/auth/jwt.go`): package `auth`, uses `github.com/golang-jwt/jwt/v5`, HS256, secret guard `len(secret) >= 16`. The TMA token is a parallel type in the same package reusing `cfg.JWTSecret`.
- **bot_users**: `GetBotUserByTelegramID(ctx, telegramID int64) (postgres.BotUser, error)`; `BotUser{ID uuid.UUID, TelegramID int64, FullName, Email, Role, ReminderMinutes string}`. `Role` is `"user"` or `"admin"` (`botreg/service.go:112-114`). Not-found → error.
- **Config** (`internal/platform/config/config.go`): `JWTSecret`, `JWTIssuer`, `AuthDevMode`, `BotToken` present. No new backend env.
- **app.go wiring points**: `jwtSvc, err := platformauth.NewJWT(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)` (~line 52); `authMW := middleware.NewAuth(...)` (~80); `api := &handlers.API{...}` (~82); `authPub := app.Group("/api/auth")` (~94); `ap := app.Group("/api", authMW.Middleware)` (~105). `platformauth` and `middleware` and `time` are already imported.
- **Frontend**: `src/shared/api/client.ts` exports `api` (axios, baseURL `/api`) and `setAuthToken(token|null)`. `TmaApp()` is exported from `src/features/tma/tma-app.tsx:308` and mounted at route `/` (`src/routes/index.tsx`). Mock `ME = EMPLOYEES[0]` in `src/shared/tma/mock-data.ts`; `ME` is used ONLY as `.name`/`.email`/`.role` in `home-screen.tsx:32`, `profile-screen.tsx:163/174/186/350`, `meetings-screen.tsx:179` (and in `create-wizard.tsx:274/346` + `meeting-utils.ts:82` — **those two stay on mock `ME`**, they're the create flow = sub-project 3). `window.Telegram?.WebApp?.initData` is already read in `link-telegram-banner.tsx`. No frontend test runner (no vitest); verify with `pnpm -C frontend typecheck` and `pnpm -C frontend build`.

## Conventions

- Backend: build/test/lint from repo root `make test && make lint && make build`; run Go as `env -u GOROOT go ...` from `backend/`. `make lint` includes gofmt. Pure logic unit-tested; handlers/middleware/wiring build-verified.
- Frontend: from repo root `pnpm -C frontend typecheck` (fast per-task check) and `pnpm -C frontend format` (prettier write) before committing; full `make build` builds the frontend.
- Commit messages end with a trailing `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>` line.
- Do NOT touch `frontend/vite.config.ts`.

## File structure (created/modified)

- Create `backend/internal/platform/auth/tma.go` + `tma_test.go` — TMA token mint/verify.
- Modify `backend/internal/infrastructure/telegram/initdata.go` + create `initdata_test.go` — `AuthDate` + `FreshAuthDate`.
- Create `backend/internal/delivery/http/handlers/tma_auth.go` — exchange handler; modify `handlers.go` (API fields).
- Create `backend/internal/delivery/http/middleware/tma_auth.go` — TMA middleware.
- Create `backend/internal/delivery/http/handlers/tma_me.go` — `/me` handler; modify `app.go` — wiring.
- Create `frontend/src/shared/tma/auth.ts` + `auth-context.tsx` — auth client + provider.
- Modify `frontend/src/features/tma/tma-app.tsx` (gate) + `screens/home-screen.tsx`, `screens/profile-screen.tsx`, `screens/meetings-screen.tsx` (ME→user).
- Modify `docs/MEETINGS.md`, `docs/REQUIREMENTS.md`, `deploy/.env.example` (frontend dev env vars).

---

## Task 1: TMA token (mint/verify)

**Files:**
- Create: `backend/internal/platform/auth/tma.go`
- Test: `backend/internal/platform/auth/tma_test.go`

Pure logic — full TDD. White-box test (same package `auth`) so it can sign with the unexported secret for the negative cases.

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/platform/auth/tma_test.go`:

```go
package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTMAToken_RoundTrip(t *testing.T) {
	tok, err := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	s, err := tok.Issue(42, "a@b.kz", "admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	claims, err := tok.Parse(s)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.TelegramID != 42 || claims.Email != "a@b.kz" || claims.Role != "admin" || claims.Typ != "tma" {
		t.Fatalf("bad claims: %+v", claims)
	}
}

func TestTMAToken_RejectsNonTMAType(t *testing.T) {
	tok, _ := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	// Sign a token with the same secret but a non-tma typ.
	bad := jwt.NewWithClaims(jwt.SigningMethodHS256, TMAClaims{TelegramID: 1, Typ: "platform"})
	s, _ := bad.SignedString(tok.secret)
	if _, err := tok.Parse(s); err == nil {
		t.Fatal("expected rejection of non-tma typ")
	}
}

func TestTMAToken_RejectsWrongSecret(t *testing.T) {
	a, _ := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	b, _ := NewTMAToken("ffffffffffffffffffff", "lead-cat", time.Hour)
	s, _ := a.Issue(1, "x@y.kz", "user")
	if _, err := b.Parse(s); err == nil {
		t.Fatal("expected rejection of wrong-secret token")
	}
}

func TestTMAToken_RejectsExpired(t *testing.T) {
	tok, _ := NewTMAToken("0123456789abcdef0123", "lead-cat", time.Hour)
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, TMAClaims{
		TelegramID: 1, Typ: "tma",
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
	})
	s, _ := expired.SignedString(tok.secret)
	if _, err := tok.Parse(s); err == nil {
		t.Fatal("expected rejection of expired token")
	}
}

func TestNewTMAToken_ShortSecret(t *testing.T) {
	if _, err := NewTMAToken("short", "lead-cat", time.Hour); err == nil {
		t.Fatal("expected error for short secret")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd backend && env -u GOROOT go test ./internal/platform/auth/ -run TMA -v`
Expected: FAIL — `undefined: NewTMAToken` / `TMAClaims`.

- [ ] **Step 3: Implement the token**

Create `backend/internal/platform/auth/tma.go`:

```go
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tmaTokenType = "tma"

// TMAClaims is a Telegram Mini App session token. Distinct from TokenClaims:
// a TMA user is a bot_users row keyed by telegram_id, not a platform_users UUID.
type TMAClaims struct {
	TelegramID int64  `json:"tg_id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Typ        string `json:"typ"`
	jwt.RegisteredClaims
}

// TMAToken mints and verifies TMA session JWTs. Reuses the platform JWT secret.
type TMAToken struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewTMAToken(secret, issuer string, ttl time.Duration) (*TMAToken, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT secret must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &TMAToken{secret: []byte(secret), ttl: ttl, issuer: issuer}, nil
}

func (t *TMAToken) Issue(telegramID int64, email, role string) (string, error) {
	now := time.Now()
	claims := TMAClaims{
		TelegramID: telegramID,
		Email:      email,
		Role:       role,
		Typ:        tmaTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *TMAToken) Parse(token string) (*TMAClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, &TMAClaims{}, func(tok *jwt.Token) (any, error) {
		if tok.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*TMAClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.Typ != tmaTokenType {
		return nil, fmt.Errorf("not a tma token")
	}
	return claims, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && env -u GOROOT go test ./internal/platform/auth/ -run TMA -v`
Expected: PASS (5 tests).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/platform/auth/tma.go backend/internal/platform/auth/tma_test.go
git commit -m "feat(tma): TMA session token mint/verify

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: initData `AuthDate` + `FreshAuthDate`

**Files:**
- Modify: `backend/internal/infrastructure/telegram/initdata.go`
- Test: `backend/internal/infrastructure/telegram/initdata_test.go`

`FreshAuthDate` is pure → unit-tested. The `AuthDate` parse is build-verified (a signed-initData test would require reproducing the HMAC; not worth it).

- [ ] **Step 1: Write the failing test for the pure helper**

Create `backend/internal/infrastructure/telegram/initdata_test.go`:

```go
package telegram

import (
	"testing"
	"time"
)

func TestFreshAuthDate(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	if !FreshAuthDate(now.Unix()-60, now, time.Hour) {
		t.Fatal("recent auth_date should be fresh")
	}
	if FreshAuthDate(now.Unix()-2*3600, now, time.Hour) {
		t.Fatal("2h-old auth_date should be stale for a 1h window")
	}
	if FreshAuthDate(0, now, time.Hour) {
		t.Fatal("absent auth_date (0) should be stale")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/telegram/ -run FreshAuthDate -v`
Expected: FAIL — `undefined: FreshAuthDate`.

- [ ] **Step 3: Add `AuthDate` field, parse it, add `FreshAuthDate`**

In `backend/internal/infrastructure/telegram/initdata.go`:

1. Add `time` and `strconv` to the import block.
2. Add the field to `InitDataUser`:

```go
type InitDataUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	AuthDate int64  `json:"-"` // from the top-level auth_date param, not the user JSON
}
```

3. In `Validate`, after the existing `if u.ID == 0 { ... }` check and before `return u, nil`, populate `AuthDate` from the query:

```go
	if ad := vals.Get("auth_date"); ad != "" {
		u.AuthDate, _ = strconv.ParseInt(ad, 10, 64)
	}
	return u, nil
```

4. Append the pure helper at the end of the file:

```go
// FreshAuthDate reports whether a Telegram auth_date (unix seconds) is within
// maxAge of now. An absent auth_date (0) is treated as stale.
func FreshAuthDate(authDate int64, now time.Time, maxAge time.Duration) bool {
	if authDate <= 0 {
		return false
	}
	return now.Sub(time.Unix(authDate, 0)) <= maxAge
}
```

- [ ] **Step 4: Run test + build**

Run: `cd backend && env -u GOROOT go test ./internal/infrastructure/telegram/ -run FreshAuthDate -v && env -u GOROOT go build ./...`
Expected: PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/infrastructure/telegram/initdata.go backend/internal/infrastructure/telegram/initdata_test.go
git commit -m "feat(tma): parse initData auth_date + FreshAuthDate freshness check

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: TMA exchange handler + API fields

**Files:**
- Modify: `backend/internal/delivery/http/handlers/handlers.go` (API struct fields)
- Create: `backend/internal/delivery/http/handlers/tma_auth.go`

Build-verified. The 401 results return an explicit JSON `{"code": ...}` so the frontend can distinguish `not_registered` from `invalid_init_data` (more robust than parsing the fiber error message).

- [ ] **Step 1: Add fields to the `API` struct**

In `backend/internal/delivery/http/handlers/handlers.go`, add this import to the existing import block (matching the `platformauth` alias the package already uses in `auth.go`):

```go
	platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
```

and add two fields to the `API` struct (after `TMA`):

```go
	TMAToken    *platformauth.TMAToken
	AuthDevMode bool
```

- [ ] **Step 2: Create the exchange handler**

Create `backend/internal/delivery/http/handlers/tma_auth.go`:

```go
package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/telegram"
)

type tmaAuthRequest struct {
	InitData string `json:"init_data"`
}

type tmaUser struct {
	TelegramID int64  `json:"telegram_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
}

// TMAAuth exchanges Telegram initData for a short-lived TMA JWT. Public route.
// 401 bodies carry a machine-readable {"code": ...} so the Mini App can tell
// not_registered (→ "register in the bot" screen) from invalid_init_data.
func (a *API) TMAAuth(c *fiber.Ctx) error {
	var req tmaAuthRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	var tgID int64
	if a.AuthDevMode {
		// Dev: no Telegram, no HMAC. init_data carries the dev telegram_id.
		id, err := strconv.ParseInt(strings.TrimSpace(req.InitData), 10, 64)
		if err != nil || id == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "dev init_data must be a telegram id")
		}
		tgID = id
	} else {
		u, err := a.TMA.Validate(req.InitData)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "invalid_init_data"})
		}
		if !telegram.FreshAuthDate(u.AuthDate, time.Now(), 24*time.Hour) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "invalid_init_data"})
		}
		tgID = u.ID
	}
	bu, err := a.App.Store.GetBotUserByTelegramID(c.Context(), tgID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": "not_registered"})
	}
	token, err := a.TMAToken.Issue(bu.TelegramID, bu.Email, bu.Role)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "token issue failed")
	}
	return c.JSON(fiber.Map{
		"token": token,
		"user":  tmaUser{TelegramID: bu.TelegramID, Name: bu.FullName, Email: bu.Email, Role: bu.Role},
	})
}
```

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/handlers/ && env -u GOROOT gofmt -l internal/delivery/http/handlers/`
Expected: build/vet clean; gofmt empty.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/handlers.go backend/internal/delivery/http/handlers/tma_auth.go
git commit -m "feat(tma): POST /api/auth/tma initData->JWT exchange handler

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: TMA auth middleware

**Files:**
- Create: `backend/internal/delivery/http/middleware/tma_auth.go`

Build-verified. Reuses the `context` import pattern of the existing `auth.go` in this package.

- [ ] **Step 1: Create the middleware**

Create `backend/internal/delivery/http/middleware/tma_auth.go`:

```go
package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
)

type tmaStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
}

// TMAAuth guards /api/tma/* with a TMA session JWT and resolves the bot_users
// row each request (so role/email changes and de-registration take effect).
type TMAAuth struct {
	token *auth.TMAToken
	store tmaStore
}

func NewTMAAuth(token *auth.TMAToken, store *postgres.Store) *TMAAuth {
	return &TMAAuth{token: token, store: store}
}

func (m *TMAAuth) Middleware(c *fiber.Ctx) error {
	hdr := c.Get("Authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	claims, err := m.token.Parse(strings.TrimPrefix(hdr, "Bearer "))
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	bu, err := m.store.GetBotUserByTelegramID(c.UserContext(), claims.TelegramID)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	c.Locals("bot_user", bu)
	return c.Next()
}
```

- [ ] **Step 2: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/middleware/ && env -u GOROOT gofmt -l internal/delivery/http/middleware/`
Expected: build/vet clean; gofmt empty.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/delivery/http/middleware/tma_auth.go
git commit -m "feat(tma): TMA auth middleware resolving bot_user from token

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: `/api/tma/me` handler + app.go wiring

**Files:**
- Create: `backend/internal/delivery/http/handlers/tma_me.go`
- Modify: `backend/internal/delivery/http/app.go`

Build-verified. This makes the whole slice live.

- [ ] **Step 1: Create the `/me` handler**

Create `backend/internal/delivery/http/handlers/tma_me.go`:

```go
package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// TMAMe returns the authenticated Telegram Mini App user's identity.
func (a *API) TMAMe(c *fiber.Ctx) error {
	bu, ok := c.Locals("bot_user").(postgres.BotUser)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return c.JSON(tmaUser{TelegramID: bu.TelegramID, Name: bu.FullName, Email: bu.Email, Role: bu.Role})
}
```

> `tmaUser` is defined in `tma_auth.go` (Task 3), same package — do not redefine it.

- [ ] **Step 2: Wire into `app.go`**

In `backend/internal/delivery/http/app.go`:

1. After `jwtSvc, err := platformauth.NewJWT(...)` (and its `if err != nil` block), add:

```go
	tmaToken, err := platformauth.NewTMAToken(cfg.JWTSecret, cfg.JWTIssuer, 24*time.Hour)
	if err != nil {
		return nil, err
	}
```

2. In the `api := &handlers.API{...}` literal, add two fields:

```go
		TMAToken:    tmaToken,
		AuthDevMode: cfg.AuthDevMode,
```

3. Register the public exchange in the `authPub` group (after the existing `authPub.*` lines):

```go
	authPub.Post("/tma", api.TMAAuth)
```

4. After the `ap := app.Group("/api", authMW.Middleware)` block (and its routes), add the TMA group:

```go
	tmaAuth := middleware.NewTMAAuth(tmaToken, store)
	tma := app.Group("/api/tma", tmaAuth.Middleware)
	tma.Get("/me", api.TMAMe)
```

> `time`, `platformauth`, and `middleware` are already imported in `app.go`. Route `/api/auth/tma` is public (the `authPub` group has no middleware); `/api/tma/*` uses only `tmaAuth.Middleware` (the platform `authMW` is group-scoped to `ap` and does not apply here).

- [ ] **Step 3: Build + vet + gofmt**

Run: `cd backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/delivery/http/... && env -u GOROOT gofmt -l internal/delivery/http/`
Expected: build/vet clean; gofmt empty.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/delivery/http/handlers/tma_me.go backend/internal/delivery/http/app.go
git commit -m "feat(tma): wire /api/auth/tma + /api/tma/me routes

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Frontend auth client + provider

**Files:**
- Create: `frontend/src/shared/tma/auth.ts`
- Create: `frontend/src/shared/tma/auth-context.tsx`

Typecheck/build-verified (no frontend test runner).

- [ ] **Step 1: Create the auth client**

Create `frontend/src/shared/tma/auth.ts`:

```ts
import { api, setAuthToken } from "@/shared/api/client"

export type TmaUser = {
  telegramId: number
  name: string
  email: string
  role: "user" | "admin"
}

type TmaAuthResponse = {
  token: string
  user: { telegram_id: number; name: string; email: string; role: "user" | "admin" }
}

// tmaLogin exchanges Telegram initData for a TMA JWT and sets it as the axios
// bearer token. Returns the authenticated user.
export async function tmaLogin(initData: string): Promise<TmaUser> {
  const res = await api.post<TmaAuthResponse>("/auth/tma", { init_data: initData })
  const { token, user } = res.data
  setAuthToken(token)
  return {
    telegramId: user.telegram_id,
    name: user.name,
    email: user.email,
    role: user.role,
  }
}

// getInitData returns the Telegram WebApp initData string, or — in dev mode —
// the configured dev telegram id (the backend dev path treats init_data as the id).
export function getInitData(): string {
  if (import.meta.env.VITE_AUTH_DEV_MODE === "true") {
    return import.meta.env.VITE_TMA_DEV_TG_ID ?? ""
  }
  const tg = (window as unknown as { Telegram?: { WebApp?: { initData?: string } } }).Telegram
  return tg?.WebApp?.initData ?? ""
}
```

- [ ] **Step 2: Create the auth provider/context**

Create `frontend/src/shared/tma/auth-context.tsx`:

```tsx
import { createContext, useContext, useEffect, useState, type ReactNode } from "react"
import { isAxiosError } from "axios"
import { getInitData, tmaLogin, type TmaUser } from "./auth"

export type TmaAuthStatus = "loading" | "authed" | "not_registered" | "error"

type TmaAuthValue = {
  status: TmaAuthStatus
  user: TmaUser | null
  retry: () => void
}

const TmaAuthContext = createContext<TmaAuthValue | null>(null)

export function TmaAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<TmaAuthStatus>("loading")
  const [user, setUser] = useState<TmaUser | null>(null)

  function run() {
    setStatus("loading")
    const initData = getInitData()
    if (!initData) {
      setStatus("error")
      return
    }
    tmaLogin(initData)
      .then((u) => {
        setUser(u)
        setStatus("authed")
      })
      .catch((e) => {
        const code = isAxiosError(e) ? (e.response?.data as { code?: string })?.code : undefined
        setStatus(code === "not_registered" ? "not_registered" : "error")
      })
  }

  useEffect(() => {
    run()
    // run once on mount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return (
    <TmaAuthContext.Provider value={{ status, user, retry: run }}>
      {children}
    </TmaAuthContext.Provider>
  )
}

export function useTmaAuth(): TmaAuthValue {
  const ctx = useContext(TmaAuthContext)
  if (!ctx) throw new Error("useTmaAuth must be used within TmaAuthProvider")
  return ctx
}
```

- [ ] **Step 3: Typecheck + format**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format`
Expected: typecheck passes; prettier writes (no errors).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/shared/tma/auth.ts frontend/src/shared/tma/auth-context.tsx
git commit -m "feat(tma): frontend auth client + provider (initData exchange)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Auth gate + identity swap in screens

**Files:**
- Modify: `frontend/src/features/tma/tma-app.tsx`
- Modify: `frontend/src/features/tma/screens/home-screen.tsx`
- Modify: `frontend/src/features/tma/screens/profile-screen.tsx`
- Modify: `frontend/src/features/tma/screens/meetings-screen.tsx`

Typecheck/build-verified. The current exported `TmaApp()` becomes the inner app; a new `TmaApp()` wraps it in the auth provider + gate.

- [ ] **Step 1: Wrap `TmaApp` with the provider + gate**

In `frontend/src/features/tma/tma-app.tsx`:

1. Add imports near the top:

```tsx
import { TmaAuthProvider, useTmaAuth } from "@/shared/tma/auth-context"
```

2. Rename the existing `export function TmaApp() {` (line ~308) to `function TmaAppInner() {` (remove `export`).

3. Add the new exported wrapper + gate at the end of the file:

```tsx
export function TmaApp() {
  return (
    <TmaAuthProvider>
      <TmaAuthGate />
    </TmaAuthProvider>
  )
}

function TmaAuthGate() {
  const { status, retry } = useTmaAuth()
  if (status === "authed") return <TmaAppInner />
  const botUsername = import.meta.env.VITE_BOT_USERNAME ?? ""
  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        gap: 16,
        padding: 24,
        textAlign: "center",
      }}
    >
      {status === "loading" && <p>Загрузка…</p>}
      {status === "not_registered" && (
        <>
          <p>Сначала зарегистрируйтесь в боте командой /start.</p>
          {botUsername && (
            <a href={`https://t.me/${botUsername}?start`} target="_blank" rel="noreferrer">
              Открыть бота
            </a>
          )}
        </>
      )}
      {status === "error" && (
        <>
          <p>Не удалось войти. Откройте приложение из Telegram.</p>
          <button onClick={retry}>Повторить</button>
        </>
      )}
    </div>
  )
}
```

> `TmaAppInner` keeps its existing body (palette/i18n provider, tabs, overlays — all still mock-backed). Only the export wrapper changed.

- [ ] **Step 2: Swap `ME` → `useTmaAuth().user` in `home-screen.tsx`**

In `frontend/src/features/tma/screens/home-screen.tsx`:
- Remove the `ME` import from `@/shared/tma/mock-data` (keep any other imports from that module).
- Add `import { useTmaAuth } from "@/shared/tma/auth-context"`.
- Inside the component, add `const { user } = useTmaAuth()`.
- Replace `const firstName = ME.name.split(" ")[0]` with:

```tsx
  const firstName = (user?.name ?? "").split(" ")[0]
```

- [ ] **Step 3: Swap `ME` → `useTmaAuth().user` in `profile-screen.tsx`**

In `frontend/src/features/tma/screens/profile-screen.tsx`:
- Remove the `ME` import from `@/shared/tma/mock-data` (keep `EMPLOYEES` and others still used).
- Add `import { useTmaAuth } from "@/shared/tma/auth-context"`.
- Inside the profile component (the one using `ME` at lines ~163/174/186/350), add `const { user } = useTmaAuth()`.
- Replace usages:
  - `<Avatar name={ME.name} size={62} />` → `<Avatar name={user?.name ?? ""} size={62} />`
  - `{ME.name}` → `{user?.name ?? ""}`
  - `{ME.email}` → `{user?.email ?? ""}`
  - `{ME.role === "admin" && (` → `{user?.role === "admin" && (`

> If `EMPLOYEES`/`ME` are referenced in the colleague-schedule or admin sub-components further down (those read meeting/employee lists), leave those on mocks — they belong to sub-projects 2 and 4. Only swap the four current-user identity usages above.

- [ ] **Step 4: Swap `ME` → `useTmaAuth().user` in `meetings-screen.tsx`**

In `frontend/src/features/tma/screens/meetings-screen.tsx`:
- Remove the `ME` import from `@/shared/tma/mock-data` (keep others).
- Add `import { useTmaAuth } from "@/shared/tma/auth-context"`.
- In the component containing line ~179, add `const { user } = useTmaAuth()`.
- Replace `const canManage = m.organizer === ME.email || ME.role === "admin"` with:

```tsx
  const canManage = m.organizer === user?.email || user?.role === "admin"
```

- [ ] **Step 5: Typecheck + format + build**

Run: `pnpm -C frontend typecheck && pnpm -C frontend format && pnpm -C frontend build`
Expected: typecheck clean; prettier writes; production build succeeds.

> If typecheck flags that `ME` is still imported but unused in any file, remove the dangling import. If a screen reads `ME` somewhere not listed above (current-user identity only), swap it the same way; if it's list data (employees/meetings), leave it on mocks.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/features/tma/tma-app.tsx frontend/src/features/tma/screens/home-screen.tsx frontend/src/features/tma/screens/profile-screen.tsx frontend/src/features/tma/screens/meetings-screen.tsx
git commit -m "feat(tma): auth gate + real identity on home/profile/meetings screens

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: Docs + env + final verification

**Files:**
- Modify: `docs/MEETINGS.md`
- Modify: `docs/REQUIREMENTS.md`
- Modify: `deploy/.env.example`

- [ ] **Step 1: Document the dev env vars in `deploy/.env.example`**

Append to `deploy/.env.example` (find the frontend `VITE_` section; if none, add at the end):

```
# Telegram Mini App local dev (only used when VITE_AUTH_DEV_MODE=true):
# the backend dev path treats init_data as a raw telegram_id, so set this to a
# telegram id that already has a bot_users row (registered via the bot once).
VITE_TMA_DEV_TG_ID=
# Bot username (without @) for the "register in the bot" deep link in the Mini App.
VITE_BOT_USERNAME=
```

- [ ] **Step 2: Update `docs/MEETINGS.md`**

In the Backend (planned) block, after the last `> **...(done):**` line, add:

```markdown
> **Mini App auth (frontend integration #1, done):** the Mini App authenticates Telegram-natively — `POST /api/auth/tma` validates `initData` (HMAC + `auth_date` freshness ≤ 24h, dev mode bypasses HMAC and treats `init_data` as a telegram id) and exchanges it for a short-lived **TMA JWT** (`tg_id`/`email`/`role`, `typ:tma`, 24h) resolved against `bot_users`; unregistered → `401 {code:not_registered}`. A dedicated `TMAAuth` middleware guards `/api/tma/*` (re-resolves the `bot_users` row per request) and `GET /api/tma/me` returns the identity. Registration stays owned by the bot `/start` flow. Frontend: an auth provider/gate (loading / authed / not_registered / error) replaces the mock current user on the home/profile/meetings screens. Meetings/employees/availability wiring is the next sub-project (still mock-backed).
```

- [ ] **Step 3: Update `docs/REQUIREMENTS.md`**

Read `docs/REQUIREMENTS.md`. Find the Mini App / frontend prerequisites area (search `Mini App` or the `VITE_` env listing). Add a bullet (matching the file's list style):

```markdown
- **Mini App auth:** Telegram-native — no separate login. The frontend exchanges Telegram `initData` at `POST /api/auth/tma` for a TMA JWT (`bot_users`-backed). Local dev (`VITE_AUTH_DEV_MODE=true`) uses `VITE_TMA_DEV_TG_ID` (a registered telegram id) instead of real `initData`; `VITE_BOT_USERNAME` powers the "register in the bot" deep link.
```

- [ ] **Step 4: Full verification from repo root**

Run: `make test && make lint && make build`
Expected: all green. If `make lint` flags gofmt, run `cd backend && env -u GOROOT gofmt -w ./internal/...` and re-run. If the frontend build flags a prettier/format issue, run `pnpm -C frontend format` and re-run.

- [ ] **Step 5: Commit**

```bash
git add docs/MEETINGS.md docs/REQUIREMENTS.md deploy/.env.example
git commit -m "docs(tma): document Mini App auth + dev env vars

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes (for the executor)

- **Spec coverage:** `POST /api/auth/tma` exchange (initData validate + freshness + bot_users + mint) → Tasks 2,3,5; TMA JWT distinct type with `typ` guard → Task 1; freshness ≤24h → Task 2; `not_registered` vs `invalid_init_data` codes → Task 3; TMA middleware re-resolving bot_user → Task 4; `GET /api/tma/me` → Task 5; route wiring (public exchange + guarded group, no skip-list edit) → Task 5; frontend bootstrap/gate state machine + 401-handling → Tasks 6,7; replace mock `ME` (identity screens only) → Task 7; dev mode (HMAC bypass, `VITE_TMA_DEV_TG_ID`) → Tasks 3,6,8; docs/env → Task 8. Out-of-scope items (meetings/employees wiring, refresh tokens, in-app registration, admin views) are intentionally absent.
- **Type consistency:** backend `TMAToken`/`TMAClaims{TelegramID,Email,Role,Typ}` (Task 1) used by handler (Task 3) and middleware (Task 4); `tmaUser{TelegramID,Name,Email,Role}` defined once in `tma_auth.go` (Task 3), reused by `tma_me.go` (Task 5); `c.Locals("bot_user")` set in Task 4, read in Task 5. Frontend `TmaUser{telegramId,name,email,role}` (Task 6) consumed by the gate + screens (Task 7); response code `not_registered` produced in Task 3, matched in Task 6.
- **Known approximations:** TMA token and platform JWT share `JWT_SECRET` (acceptable; `typ` prevents cross-use). The frontend uses the shared axios `setAuthToken`, which is fine because the Mini App has no competing platform JWT; `LinkTelegramBanner` (platform-authed) is out of scope. `me` identity is exposed via `useTmaAuth()` and read with optional chaining in screens (no null guards needed since screens render only when `authed`). Per-request `GetBotUserByTelegramID` in the middleware is a cheap single lookup.
```
