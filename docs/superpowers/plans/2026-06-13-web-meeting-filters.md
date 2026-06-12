# Web Meeting Filters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add server-side filtering (status, date range, department, organizer) to the web admin meetings list.

**Architecture:** A pure `MeetingFilter` value flows handler → application → repository. The HTTP handler parses query params into the filter (pure, testable). The application layer applies the existing owner/non-owner authorization (non-owners are forced to their own meetings regardless of requested organizer). The postgres layer builds a dynamic `WHERE` from the filter (pure builder, testable). The admin UI gains a filter bar above the table, wired through TanStack Query with the filter in the query key; mutations still invalidate by the `["orgs", orgId, "meetings"]` prefix so all filtered variants refetch.

**Tech Stack:** Go (Fiber, pgx), `model` value types + `Repository` port, OpenAPI → `@leadcat/api-client` (openapi-typescript), React Router v7 admin + TanStack Query + shadcn `@leadcat/ui` + axios.

**Prerequisite:** Start on top of the now-committed CQRS/`@leadcat/types` refactor (HEAD at or after `fac0ec7`). Run from repo root unless noted. Backend Go commands run from `apps/backend` (`env -u GOROOT go ...`).

---

## File Structure

**Backend (create/modify):**
- Modify `apps/backend/internal/application/model/model.go` — add `MeetingFilter` struct.
- Modify `apps/backend/internal/application/repository.go` — add `ListMeetingsFiltered` to the `Repository` port.
- Modify `apps/backend/internal/application/repository_unimpl_test.go` — implement the new method on the test stub.
- Modify `apps/backend/internal/application/meeting_service.go` — add `ListMeetingsFiltered`; make `ListMeetings` delegate to it.
- Create `apps/backend/internal/application/meeting_list_filter_test.go` — authorization-override test.
- Modify `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go` — add `meetingFilter` builder + `ListMeetingsFiltered`.
- Create `apps/backend/internal/infrastructure/persistence/postgres/meeting_filter_test.go` — builder table test.
- Modify `apps/backend/internal/delivery/http/handlers/web_meetings.go` — add `parseMeetingFilter`; wire `WebListMeetings`.
- Create `apps/backend/internal/delivery/http/handlers/web_meetings_filter_test.go` — parse table test.
- Modify `apps/backend/openapi/openapi.json` — add 5 query params to `GET /api/orgs/{id}/meetings`.

**Frontend (create/modify):**
- Modify `apps/admin/app/entities/meeting/types.ts` — add `MeetingListFilter` type.
- Modify `apps/admin/app/entities/meeting/api.ts` — `listMeetings(orgId, filter)`.
- Modify `apps/admin/app/entities/meeting/queries.ts` — `useMeetings(orgId, filter)` + `listFiltered` key.
- Create `apps/admin/app/shared/lib/use-debounced-value.ts` — debounce hook.
- Create `apps/admin/app/features/meetings/components/meetings-filter-bar.tsx` — filter bar.
- Modify `apps/admin/app/features/meetings/pages/meetings-page.tsx` — own filter state, render bar, pass to query.

---

## Task 1: Backend — MeetingFilter model + postgres builder + repo method

**Files:**
- Modify: `apps/backend/internal/application/model/model.go`
- Modify: `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go`
- Test: `apps/backend/internal/infrastructure/persistence/postgres/meeting_filter_test.go`

- [ ] **Step 1: Add the `MeetingFilter` struct to the model package**

Add to `apps/backend/internal/application/model/model.go` (near the `Meeting` struct). `Status` empty or `"all"` means no status predicate; `From` is an inclusive lower bound and `To` an exclusive upper bound on `starts_at`; `Dept` is a case-insensitive contains match; `Organizer` filters `organizer_user_id`.

```go
// MeetingFilter narrows a meetings query. Zero value = no filtering.
type MeetingFilter struct {
	Status    string     // "scheduled" | "cancelled"; "" or "all" = any
	From      *time.Time // inclusive lower bound on starts_at
	To        *time.Time // exclusive upper bound on starts_at
	Dept      string     // case-insensitive contains match when non-empty
	Organizer *uuid.UUID // organizer_user_id when non-nil
}
```

Verify `model.go` already imports `time` and `github.com/google/uuid` (the `Meeting` struct uses both). If not, add them.

- [ ] **Step 2: Write the failing builder test**

Create `apps/backend/internal/infrastructure/persistence/postgres/meeting_filter_test.go`:

```go
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
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestMeetingFilter`
Expected: FAIL — `undefined: meetingFilter`.

- [ ] **Step 4: Implement the builder + repo method**

Add to `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go`. Confirm the file imports `fmt` and `model` (`github.com/luckyrogue/lead-cat/internal/application/model`); add them to the import block if missing.

```go
// meetingFilter builds the WHERE clause and ordered args for a filtered
// meetings query. $1 is always organization_id.
func meetingFilter(organizationID uuid.UUID, f model.MeetingFilter) (string, []any) {
	args := []any{organizationID}
	where := "organization_id = $1"
	add := func(expr string, val any) {
		args = append(args, val)
		where += fmt.Sprintf(" AND %s $%d", expr, len(args))
	}
	switch f.Status {
	case "scheduled", "cancelled":
		add("status =", f.Status)
	}
	if f.From != nil {
		add("starts_at >=", *f.From)
	}
	if f.To != nil {
		add("starts_at <", *f.To)
	}
	if f.Dept != "" {
		add("dept ILIKE", "%"+f.Dept+"%")
	}
	if f.Organizer != nil {
		add("organizer_user_id =", *f.Organizer)
	}
	return where, args
}

// ListMeetingsFiltered returns the organization's meetings matching f, newest first.
func (s *Store) ListMeetingsFiltered(ctx context.Context, organizationID uuid.UUID, f model.MeetingFilter) ([]Meeting, error) {
	where, args := meetingFilter(organizationID, f)
	return s.queryMeetings(ctx, `SELECT `+meetingCols+` FROM meetings WHERE `+where+` ORDER BY starts_at DESC`, args...)
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd apps/backend && env -u GOROOT go test ./internal/infrastructure/persistence/postgres/ -run TestMeetingFilter`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/application/model/model.go \
  apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go \
  apps/backend/internal/infrastructure/persistence/postgres/meeting_filter_test.go
git commit -m "feat(meetings): MeetingFilter model + postgres filtered query

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: Backend — Repository port + application method with auth override

**Files:**
- Modify: `apps/backend/internal/application/repository.go:57-58`
- Modify: `apps/backend/internal/application/repository_unimpl_test.go`
- Modify: `apps/backend/internal/application/meeting_service.go:20-29`
- Test: `apps/backend/internal/application/meeting_list_filter_test.go`

- [ ] **Step 1: Add the method to the Repository port**

In `apps/backend/internal/application/repository.go`, immediately after the existing `ListMeetingsByOrganizer` line (around line 58), add:

```go
	ListMeetingsFiltered(ctx context.Context, organizationID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error)
```

- [ ] **Step 2: Implement it on the test stub**

In `apps/backend/internal/application/repository_unimpl_test.go`, add a method on `unimplementedRepo` mirroring the existing stub style (these panic/return-zero). Match the surrounding pattern exactly; if the existing stubs `panic("unimplemented")`, do the same:

```go
func (unimplementedRepo) ListMeetingsFiltered(context.Context, uuid.UUID, model.MeetingFilter) ([]model.Meeting, error) {
	panic("unimplemented")
}
```

(If the file's existing stubs return `(nil, nil)` instead of panicking, follow that style instead.)

- [ ] **Step 3: Write the failing authorization test**

Create `apps/backend/internal/application/meeting_list_filter_test.go`:

```go
package application_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type filterFakeRepo struct {
	unimplementedRepo
	owner     uuid.UUID
	gotFilter model.MeetingFilter
}

func (r *filterFakeRepo) GetOrganization(context.Context, uuid.UUID) (model.Organization, error) {
	owner := r.owner
	return model.Organization{OwnerUserID: &owner}, nil
}

func (r *filterFakeRepo) ListMeetingsFiltered(_ context.Context, _ uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error) {
	r.gotFilter = f
	return nil, nil
}

func TestListMeetingsFiltered_NonOwnerForcedToSelf(t *testing.T) {
	userID := uuid.New()
	requested := uuid.New()
	repo := &filterFakeRepo{owner: uuid.New()} // owner != userID
	s := &application.Services{Store: repo}

	_, err := s.ListMeetingsFiltered(context.Background(), uuid.New(), userID, model.MeetingFilter{Organizer: &requested})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotFilter.Organizer == nil || *repo.gotFilter.Organizer != userID {
		t.Fatalf("non-owner organizer = %v, want %v", repo.gotFilter.Organizer, userID)
	}
}

func TestListMeetingsFiltered_OwnerKeepsRequestedOrganizer(t *testing.T) {
	ownerID := uuid.New()
	requested := uuid.New()
	repo := &filterFakeRepo{owner: ownerID}
	s := &application.Services{Store: repo}

	_, err := s.ListMeetingsFiltered(context.Background(), uuid.New(), ownerID, model.MeetingFilter{Organizer: &requested})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotFilter.Organizer == nil || *repo.gotFilter.Organizer != requested {
		t.Fatalf("owner organizer = %v, want %v", repo.gotFilter.Organizer, requested)
	}
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/ -run TestListMeetingsFiltered`
Expected: FAIL — `s.ListMeetingsFiltered undefined`.

- [ ] **Step 5: Implement the application method**

In `apps/backend/internal/application/meeting_service.go`, replace the existing `ListMeetings` (lines 20-29) with a thin delegate plus the new filtered method:

```go
func (s *Services) ListMeetings(ctx context.Context, organizationID, userID uuid.UUID) ([]model.Meeting, error) {
	return s.ListMeetingsFiltered(ctx, organizationID, userID, model.MeetingFilter{})
}

// ListMeetingsFiltered lists an organization's meetings matching f. Organization
// owners see all meetings; non-owners are restricted to the ones they organize
// (the requested organizer filter is overridden for them).
func (s *Services) ListMeetingsFiltered(ctx context.Context, organizationID, userID uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error) {
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if w.OwnerUserID == nil || *w.OwnerUserID != userID {
		f.Organizer = &userID
	}
	return s.Store.ListMeetingsFiltered(ctx, organizationID, f)
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd apps/backend && env -u GOROOT go test ./internal/application/ -run TestListMeetingsFiltered`
Expected: PASS.

- [ ] **Step 7: Verify the whole backend still builds (interface satisfied everywhere)**

Run: `cd apps/backend && env -u GOROOT go build ./...`
Expected: no output (success). If a non-stub fake repo elsewhere fails to compile, add the `ListMeetingsFiltered` method to it the same way.

- [ ] **Step 8: Commit**

```bash
git add apps/backend/internal/application/repository.go \
  apps/backend/internal/application/repository_unimpl_test.go \
  apps/backend/internal/application/meeting_service.go \
  apps/backend/internal/application/meeting_list_filter_test.go
git commit -m "feat(meetings): application ListMeetingsFiltered with owner/organizer scoping

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: Backend — parse query params + wire the handler

**Files:**
- Modify: `apps/backend/internal/delivery/http/handlers/web_meetings.go:92-110`
- Test: `apps/backend/internal/delivery/http/handlers/web_meetings_filter_test.go`

- [ ] **Step 1: Write the failing parser test**

Create `apps/backend/internal/delivery/http/handlers/web_meetings_filter_test.go`:

```go
package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseMeetingFilter(t *testing.T) {
	org := uuid.New()

	t.Run("empty is zero filter", func(t *testing.T) {
		f, err := parseMeetingFilter("", "", "", "", "")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if f.Status != "" || f.From != nil || f.To != nil || f.Dept != "" || f.Organizer != nil {
			t.Fatalf("filter not zero: %+v", f)
		}
	})

	t.Run("all maps to no status", func(t *testing.T) {
		f, err := parseMeetingFilter("all", "", "", "", "")
		if err != nil || f.Status != "" {
			t.Fatalf("f=%+v err=%v", f, err)
		}
	})

	t.Run("invalid status errors", func(t *testing.T) {
		if _, err := parseMeetingFilter("bogus", "", "", "", ""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("dates parse; to is exclusive next day", func(t *testing.T) {
		f, err := parseMeetingFilter("scheduled", "2026-06-01", "2026-06-30", "Eng", org.String())
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if f.Status != "scheduled" || f.Dept != "Eng" {
			t.Fatalf("f=%+v", f)
		}
		wantFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		wantTo := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		if f.From == nil || !f.From.Equal(wantFrom) || f.To == nil || !f.To.Equal(wantTo) {
			t.Fatalf("from=%v to=%v", f.From, f.To)
		}
		if f.Organizer == nil || *f.Organizer != org {
			t.Fatalf("organizer=%v", f.Organizer)
		}
	})

	t.Run("bad date errors", func(t *testing.T) {
		if _, err := parseMeetingFilter("", "nope", "", "", ""); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("bad organizer errors", func(t *testing.T) {
		if _, err := parseMeetingFilter("", "", "", "", "not-a-uuid"); err == nil {
			t.Fatal("expected error")
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd apps/backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run TestParseMeetingFilter`
Expected: FAIL — `undefined: parseMeetingFilter`.

- [ ] **Step 3: Implement the parser and wire the handler**

In `apps/backend/internal/delivery/http/handlers/web_meetings.go`: confirm the import block has `fmt`, `strings`, `time`, and `model` (`github.com/luckyrogue/lead-cat/internal/application/model`); add any missing. Add the parser:

```go
// parseMeetingFilter turns raw query values into a model.MeetingFilter.
// status "" or "all" means no status filter; from/to are YYYY-MM-DD (to is an
// inclusive day, stored as the exclusive next-day bound); organizer is a UUID.
func parseMeetingFilter(status, from, to, dept, organizer string) (model.MeetingFilter, error) {
	f := model.MeetingFilter{Dept: strings.TrimSpace(dept)}
	switch status {
	case "", "all":
	case "scheduled", "cancelled":
		f.Status = status
	default:
		return f, fmt.Errorf("invalid status")
	}
	if from != "" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			return f, fmt.Errorf("invalid from")
		}
		f.From = &t
	}
	if to != "" {
		t, err := time.Parse("2006-01-02", to)
		if err != nil {
			return f, fmt.Errorf("invalid to")
		}
		end := t.AddDate(0, 0, 1)
		f.To = &end
	}
	if organizer != "" {
		id, err := uuid.Parse(organizer)
		if err != nil {
			return f, fmt.Errorf("invalid organizer")
		}
		f.Organizer = &id
	}
	return f, nil
}
```

Then replace the body of `WebListMeetings` (lines 92-110) so it parses the filter and calls the filtered method:

```go
func (a *API) WebListMeetings(c *fiber.Ctx) error {
	user, ok := webUser(c)
	if !ok {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	orgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_org_id")
	}
	filter, err := parseMeetingFilter(c.Query("status"), c.Query("from"), c.Query("to"), c.Query("dept"), c.Query("organizer"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	ms, err := a.App.ListMeetingsFiltered(c.UserContext(), orgID, user.ID, filter)
	if err != nil {
		a.Log.Error("web_meetings_list_failed", zap.String("org_id", orgID.String()), zap.Error(err))
		return fiber.NewError(fiber.StatusInternalServerError, "list_failed")
	}
	if ms == nil {
		ms = []model.Meeting{}
	}
	return c.JSON(fiber.Map{"meetings": ms})
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd apps/backend && env -u GOROOT go test ./internal/delivery/http/handlers/ -run TestParseMeetingFilter`
Expected: PASS.

- [ ] **Step 5: Run the full backend test suite + lint**

Run: `cd apps/backend && env -u GOROOT go test ./... && env -u GOROOT golangci-lint run --config ../../config/.golangci.yml ./internal/application/... ./internal/delivery/... ./internal/infrastructure/persistence/...`
Expected: all `ok`; `0 issues`. (Pre-existing unrelated lint in other packages may exist — only your touched packages must be clean.)

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/delivery/http/handlers/web_meetings.go \
  apps/backend/internal/delivery/http/handlers/web_meetings_filter_test.go
git commit -m "feat(meetings): parse meeting list filters in WebListMeetings

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: Backend — OpenAPI params + regenerate api-client

**Files:**
- Modify: `apps/backend/openapi/openapi.json` (the `GET /api/orgs/{id}/meetings` parameters array, ~line 1751-1758)
- Regenerate: `packages/api-client/src/generated/schema.ts`

- [ ] **Step 1: Add the query parameters to the GET operation**

In `apps/backend/openapi/openapi.json`, find the `"/api/orgs/{id}/meetings"` → `"get"` → `"parameters"` array. It currently contains only the `id` path param. Replace that array with the path param plus the five query params (keep the compact one-line `schema` style used elsewhere in this file — do NOT run prettier on this file):

```json
        "parameters": [
          {
            "name": "id",
            "in": "path",
            "required": true,
            "schema": { "type": "string", "format": "uuid" }
          },
          {
            "name": "status",
            "in": "query",
            "required": false,
            "schema": { "type": "string", "enum": ["scheduled", "cancelled", "all"] }
          },
          {
            "name": "from",
            "in": "query",
            "required": false,
            "schema": { "type": "string", "format": "date" }
          },
          {
            "name": "to",
            "in": "query",
            "required": false,
            "schema": { "type": "string", "format": "date" }
          },
          {
            "name": "dept",
            "in": "query",
            "required": false,
            "schema": { "type": "string" }
          },
          {
            "name": "organizer",
            "in": "query",
            "required": false,
            "schema": { "type": "string", "format": "uuid" }
          }
        ],
```

- [ ] **Step 2: Validate the JSON and regenerate the client**

Run: `python3 -c "import json; json.load(open('apps/backend/openapi/openapi.json')); print('valid')" && pnpm openapi:generate`
Expected: `valid`, then openapi-typescript writes `packages/api-client/src/generated/schema.ts` with no error.

- [ ] **Step 3: Typecheck the api-client package**

Run: `pnpm --filter @leadcat/api-client typecheck`
Expected: no errors. (If this reports pre-existing missing `@types/node`/`vite/client` errors unrelated to the schema, note it and continue — the generation itself is the deliverable here.)

- [ ] **Step 4: Commit**

```bash
git add apps/backend/openapi/openapi.json packages/api-client/src/generated/schema.ts
git commit -m "feat(meetings): document meeting list filter query params + regen api-client

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: Admin — filter type + api + query plumbing

**Files:**
- Modify: `apps/admin/app/entities/meeting/types.ts`
- Modify: `apps/admin/app/entities/meeting/api.ts:9-14`
- Modify: `apps/admin/app/entities/meeting/queries.ts`

- [ ] **Step 1: Add the `MeetingListFilter` type**

Append to `apps/admin/app/entities/meeting/types.ts`:

```typescript
export type MeetingStatusFilter = "scheduled" | "cancelled" | "all"

export type MeetingListFilter = {
  status?: MeetingStatusFilter
  from?: string
  to?: string
  dept?: string
  organizer?: string
}
```

- [ ] **Step 2: Make `listMeetings` accept the filter**

In `apps/admin/app/entities/meeting/api.ts`, update the imports to include `MeetingListFilter` and replace the `listMeetings` function:

```typescript
import type {
  CreateMeetingInput,
  Meeting,
  MeetingListFilter,
  MeetingScope,
  UpdateMeetingInput,
} from "~/entities/meeting/types"

export async function listMeetings(
  orgId: string,
  filter: MeetingListFilter = {}
): Promise<Meeting[]> {
  const params: Record<string, string> = {}
  if (filter.status && filter.status !== "all") {
    params.status = filter.status
  }
  if (filter.from) {
    params.from = filter.from
  }
  if (filter.to) {
    params.to = filter.to
  }
  if (filter.dept) {
    params.dept = filter.dept
  }
  if (filter.organizer) {
    params.organizer = filter.organizer
  }
  const { data } = await api.get<{ meetings: Meeting[] }>(
    `/api/orgs/${orgId}/meetings`,
    { params }
  )
  return data.meetings ?? []
}
```

- [ ] **Step 3: Thread the filter through the query hook**

Replace `apps/admin/app/entities/meeting/queries.ts` with:

```typescript
import { useQuery } from "@tanstack/react-query"

import { getMeeting, listMeetings } from "~/entities/meeting/api"
import type { MeetingListFilter } from "~/entities/meeting/types"

export const meetingKeys = {
  list: (orgId: string) => ["orgs", orgId, "meetings"] as const,
  listFiltered: (orgId: string, filter: MeetingListFilter) =>
    ["orgs", orgId, "meetings", filter] as const,
  detail: (orgId: string, meetingId: string) =>
    ["orgs", orgId, "meetings", meetingId] as const,
}

export function useMeetings(orgId: string | null, filter: MeetingListFilter = {}) {
  return useQuery({
    queryKey: meetingKeys.listFiltered(orgId ?? "", filter),
    queryFn: () => listMeetings(orgId as string, filter),
    enabled: Boolean(orgId),
  })
}

export function useMeeting(orgId: string, meetingId: string | null) {
  return useQuery({
    queryKey: meetingKeys.detail(orgId, meetingId ?? ""),
    queryFn: () => getMeeting(orgId, meetingId as string),
    enabled: Boolean(orgId) && Boolean(meetingId),
  })
}
```

Note: `meetingKeys.list(orgId)` is unchanged, so the existing mutation invalidations (which use the `["orgs", orgId, "meetings"]` prefix) still match every filtered variant.

- [ ] **Step 4: Typecheck**

Run: `make typecheck` (or `cd apps/admin && pnpm typecheck`)
Expected: no errors. (The `meetings-page.tsx` call `useMeetings(activeOrgId)` still compiles because `filter` defaults to `{}`.)

- [ ] **Step 5: Commit**

```bash
git add apps/admin/app/entities/meeting/types.ts \
  apps/admin/app/entities/meeting/api.ts \
  apps/admin/app/entities/meeting/queries.ts
git commit -m "feat(admin): plumb meeting list filter through api + query

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: Admin — debounce hook, filter bar, wire the page

**Files:**
- Create: `apps/admin/app/shared/lib/use-debounced-value.ts`
- Create: `apps/admin/app/features/meetings/components/meetings-filter-bar.tsx`
- Modify: `apps/admin/app/features/meetings/pages/meetings-page.tsx`

- [ ] **Step 1: Create the debounce hook**

Create `apps/admin/app/shared/lib/use-debounced-value.ts`:

```typescript
import { useEffect, useState } from "react"

export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const handle = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(handle)
  }, [value, delayMs])
  return debounced
}
```

- [ ] **Step 2: Create the filter bar**

Create `apps/admin/app/features/meetings/components/meetings-filter-bar.tsx`. It renders status/organizer selects, a date range, and a department text input. Status/organizer/date commit immediately via `onFilterChange`; the department string is owned by the page and debounced there, so the bar takes `dept` + `onDeptChange` separately.

```typescript
import {
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@leadcat/ui"

import type { MeetingListFilter } from "~/entities/meeting/types"

export type OrganizerOption = { id: string; label: string }

type MeetingsFilterBarProps = {
  filter: MeetingListFilter
  dept: string
  organizers: OrganizerOption[]
  onFilterChange: (patch: Partial<MeetingListFilter>) => void
  onDeptChange: (value: string) => void
}

export function MeetingsFilterBar({
  filter,
  dept,
  organizers,
  onFilterChange,
  onDeptChange,
}: MeetingsFilterBarProps) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <Field label="Status">
        <Select
          value={filter.status ?? "all"}
          onValueChange={(value) =>
            onFilterChange({
              status: value === "all" ? undefined : (value as "scheduled" | "cancelled"),
            })
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="scheduled">Scheduled</SelectItem>
            <SelectItem value="cancelled">Cancelled</SelectItem>
          </SelectContent>
        </Select>
      </Field>

      <Field label="Organizer">
        <Select
          value={filter.organizer ?? "all"}
          onValueChange={(value) =>
            onFilterChange({ organizer: value === "all" ? undefined : value })
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Anyone</SelectItem>
            {organizers.map((organizer) => (
              <SelectItem key={organizer.id} value={organizer.id}>
                {organizer.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </Field>

      <Field label="From">
        <Input
          type="date"
          value={filter.from ?? ""}
          onChange={(event) =>
            onFilterChange({ from: event.target.value || undefined })
          }
        />
      </Field>

      <Field label="To">
        <Input
          type="date"
          value={filter.to ?? ""}
          onChange={(event) =>
            onFilterChange({ to: event.target.value || undefined })
          }
        />
      </Field>

      <Field label="Department">
        <Input
          placeholder="Filter by department"
          value={dept}
          onChange={(event) => onDeptChange(event.target.value)}
        />
      </Field>
    </div>
  )
}

function Field({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      {children}
    </div>
  )
}
```

- [ ] **Step 3: Wire the bar into the page**

In `apps/admin/app/features/meetings/pages/meetings-page.tsx`, add imports and filter state, derive organizer options from `useMembers`, and pass the effective filter to `useMeetings`. Apply these edits:

Add imports (with the existing import group):

```typescript
import { useMembers } from "~/entities/org/queries"
import type { MeetingListFilter } from "~/entities/meeting/types"
import {
  MeetingsFilterBar,
  type OrganizerOption,
} from "~/features/meetings/components/meetings-filter-bar"
import { useDebouncedValue } from "~/shared/lib/use-debounced-value"
```

Inside `MeetingsPage`, immediately after the `const deleteMeeting = ...` line, add:

```typescript
  const [filter, setFilter] = useState<MeetingListFilter>({})
  const [deptInput, setDeptInput] = useState("")
  const debouncedDept = useDebouncedValue(deptInput, 300)
  const effectiveFilter: MeetingListFilter = {
    ...filter,
    dept: debouncedDept || undefined,
  }
  const members = useMembers(activeOrgId)
  const organizers: OrganizerOption[] = (members.data ?? [])
    .filter((member) => member.user_id)
    .map((member) => ({
      id: member.user_id as string,
      label: member.name || member.email,
    }))
```

Change the meetings query line from `const meetings = useMeetings(activeOrgId)` to:

```typescript
  const meetings = useMeetings(activeOrgId, effectiveFilter)
```

Render the filter bar inside `<ListPageShell>`, directly above `<MeetingsTable ...>`:

```typescript
        <MeetingsFilterBar
          filter={filter}
          dept={deptInput}
          organizers={organizers}
          onFilterChange={(patch) => setFilter((current) => ({ ...current, ...patch }))}
          onDeptChange={setDeptInput}
        />
```

Note: leave `ListPageShell`'s `isEmpty`/`emptyState` as-is — an empty filtered result will show the existing empty state, which is acceptable for v1.

- [ ] **Step 4: Verify the org members query export exists**

Run: `grep -n "export function useMembers" apps/admin/app/entities/org/queries.ts`
Expected: one match. (If the hook lives elsewhere, adjust the import path in Step 3 accordingly — it returns `OrgMember[]` with `user_id`, `name`, `email`.)

- [ ] **Step 5: Typecheck, lint, build the admin app**

Run: `cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`
Expected: all succeed, no errors.

- [ ] **Step 6: Format only the touched frontend files (correct per-app config)**

Run: `npx prettier --write --config apps/admin/config/prettier.config.mjs apps/admin/app/shared/lib/use-debounced-value.ts apps/admin/app/features/meetings/components/meetings-filter-bar.tsx apps/admin/app/features/meetings/pages/meetings-page.tsx`
Expected: the three files reported. (Do NOT run the app-wide `pnpm format` — it reformats unrelated files. Do NOT use a config-less `npx prettier`, which defaults to `semi: true` and corrupts the no-semicolon style.)

- [ ] **Step 7: Commit**

```bash
git add apps/admin/app/shared/lib/use-debounced-value.ts \
  apps/admin/app/features/meetings/components/meetings-filter-bar.tsx \
  apps/admin/app/features/meetings/pages/meetings-page.tsx
git commit -m "feat(admin): meeting list filter bar (status, organizer, date range, department)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: Final verification

- [ ] **Step 1: Backend full gate**

Run: `cd apps/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./...`
Expected: build clean; all packages `ok`.

- [ ] **Step 2: Admin full gate**

Run: `cd apps/admin && pnpm typecheck && pnpm lint && pnpm build`
Expected: all succeed.

- [ ] **Step 3: Manual smoke (optional, requires running stack)**

With the backend + admin running and signed in as an org owner, open the Meetings page and confirm: changing Status/Organizer/From/To refetches and narrows the list; typing a department filters after a ~300 ms pause; clearing all controls restores the full list. As a non-owner member, confirm only your own meetings show regardless of the Organizer selection.

- [ ] **Step 4: Confirm the diff is scoped and the working tree is clean**

Run: `git status --short`
Expected: no unexpected modified files (only the files this plan created/changed are committed; `.gitignore` and build artifacts untouched).

---

## Notes & decisions

- **Department match is case-insensitive `ILIKE '%…%'`** so the free-text input is forgiving; a dedicated dept dropdown can come later if a "distinct departments" endpoint is added.
- **`to` is an inclusive calendar day**, implemented as the exclusive next-day bound — selecting From=To still returns that day's meetings.
- **Date filtering is in UTC** against `starts_at`. This is a v1 simplification; once per-user timezone lands (a later phase), revisit whether the range should be interpreted in the user's timezone.
- **Authorization is unchanged**: owners filter freely (including by organizer); non-owners are always scoped to their own meetings, so a non-owner cannot use `organizer` to see others' meetings.
- **No new participant loading**: the list endpoint still returns meetings without participants (unchanged behavior); filtering does not depend on participants.
