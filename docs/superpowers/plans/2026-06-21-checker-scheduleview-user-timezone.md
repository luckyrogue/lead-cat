# checker / scheduleview — user timezone Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Parse date input and render times in `checker` and `scheduleview` using the user's stored timezone (`bot_users.timezone`) instead of hardcoded `Asia/Almaty`.

**Architecture:** A new dispatcher helper `resolveLangLoc` resolves the user's language + `*time.Location` in one lookup. The two FSMs gain a trailing `loc *time.Location` on the date-parsing/display methods (alongside the `lang` added in C1); the `almaty()` call-site arguments become the threaded `loc`. The parse/display helpers already take `*time.Location` and are unchanged.

**Tech Stack:** Go, `go-telegram/bot`, existing `boti18n.Resolve`.

## Global Constraints

- **Location fallback chain:** `time.LoadLocation(bot_users.timezone)`, falling back to `Asia/Almaty`, then `time.UTC` on load error — exactly what the existing `almaty()` provides for the unset case.
- **`loc` is a trailing parameter** threaded only into methods that parse or display dates; `Start` (prompt-only) stays on `resolveLang` (lang only).
- **Helpers unchanged:** `parseRange`, `parseDate`, `dayWindow`, `formatSlots`, `scheduleText` already take `*time.Location` — do not change their signatures; only the `almaty()` arguments at call sites become `loc`.
- **`almaty()` retained** as the resolver's fallback default (no longer called from the parse/display flow).
- **Out of scope:** weekday-name localization (`ruWeekday`/`dayLabel` stay Russian — separate i18n tail); all other surfaces.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: `resolveLangLoc` helper + localize `checker` parsing/display

**Files:**
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` (add `resolveLangLoc`; wire 2 checker sites)
- Modify: `apps/backend/internal/platform/checker/service.go` (`OnText`/`OnCallback`/`setRange`/`duration` gain `loc`)
- Test: `apps/backend/internal/platform/checker/service_test.go`

**Interfaces:**
- Produces: `func (h *MultiHandler) resolveLangLoc(ctx, from *models.User) (string, *time.Location)` (consumed by Task 2 too); checker `OnText(...,lang,loc)`, `OnCallback(...,lang,loc)`, `setRange(...,lang,loc)`, `duration(...,lang,loc)`.

- [ ] **Step 1: Add `resolveLangLoc` to `multitenant.go`**

Confirm `time` is imported in `multitenant.go` (add it to the import block if absent). Add next to `resolveLang`:

```go
// resolveLangLoc resolves the acting user's language and display/parse location in a
// single store lookup. The location is the user's stored timezone, falling back to
// Asia/Almaty, then UTC on load error.
func (h *MultiHandler) resolveLangLoc(ctx context.Context, from *models.User) (string, *time.Location) {
	var storedLang, tz string
	if u, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err == nil {
		storedLang, tz = u.Language, u.Timezone
	}
	if tz == "" {
		tz = "Asia/Almaty"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return boti18n.Resolve(storedLang, from.LanguageCode), loc
}
```

- [ ] **Step 2: Write the failing test (`checker/service_test.go`), run red**

`setRange`/`duration` are methods, but `parseRange` and `formatSlots` are pure and loc-parameterized — assert the timezone behavior directly through them (they already exist) plus a method-level check that `setRange` stores the range parsed under the passed `loc`. Add:

```go
func TestChecker_SetRange_UsesLoc(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	svc := New(&fakeBackend{}, newFakeSessions())
	ctx := context.Background()
	// seed a session in the range step
	_ = svc.Start(ctx, 1, "ru")
	// drive to range step: add a participant then "done" — or set the session directly
	// via the fake if exposed. Simplest: call setRange through OnText after forcing step.
	// Assert the parsed range reflects London midnight, not Almaty.
	from, _, err := parseRange("2026-06-15..2026-06-15", london)
	if err != nil {
		t.Fatal(err)
	}
	fromAlmaty, _, _ := parseRange("2026-06-15..2026-06-15", almaty())
	if from.Equal(fromAlmaty) {
		t.Fatal("expected London and Almaty parse of the same date to differ in absolute time")
	}
	if from.UTC().Hour() != 23 { // 2026-06-15 00:00 BST == 2026-06-14 23:00 UTC
		t.Errorf("London 2026-06-15 midnight should be 23:00 prev-day UTC, got %v", from.UTC())
	}
}
```

> This pins the helper's tz behavior (the core of the fix). Run: `cd apps/backend && go test ./internal/platform/checker/ -run TestChecker_SetRange_UsesLoc 2>&1 | head` — Expected: PASS already for the pure-helper assertions (they don't need the threading). The threading itself is verified by build + the existing flow tests once methods take `loc`. If you prefer a method-level assertion, drive a full OnText→setRange with a London `loc` and assert the stored `st.From`/`st.To`; either is acceptable. The mandatory red→green signal for this task is the compile error in Step 4 when call sites pass `loc` before the methods accept it.

- [ ] **Step 3: Thread `loc` through `checker/service.go`**

- `OnText(ctx, telegramID int64, text, lang string, loc *time.Location) (Reply, bool)`: in the `stepRange` case, call `s.setRange(ctx, telegramID, st, text, lang, loc)`. (The `stepParticipants` → `s.search(...)` path is unchanged — no `loc`.)
- `OnCallback(ctx, telegramID int64, data, lang string, loc *time.Location) (Reply, bool)`: in the `chk:dur:` case, call `s.duration(ctx, telegramID, st, ..., lang, loc)`. (Other cases unchanged.)
- `setRange(ctx, telegramID int64, st *State, text, lang string, loc *time.Location) Reply`: change `parseRange(text, almaty())` → `parseRange(text, loc)`.
- `duration(ctx, telegramID int64, st *State, durStr, lang string, loc *time.Location) Reply`: replace `loc := almaty()` with the passed-in `loc` (delete the local `loc := almaty()` line; the rest — `time.ParseInLocation(..., loc)`, `formatSlots(slots, n, loc, lang)` — already uses `loc`).

> `Start` is unchanged (prompt only, no date). `almaty()` stays defined (it's the resolver fallback and still referenced by the test).

- [ ] **Step 4: Wire the 2 checker dispatcher sites in `multitenant.go`**

- OnText (~line 93):
  ```go
  lang, loc := h.resolveLangLoc(ctx, from)
  if reply, handled := h.checker.OnText(ctx, from.ID, text, lang, loc); handled {
  ```
  (Place the `lang, loc :=` line just before the checker `OnText` call. If `h.schedule.OnText` on the line above still uses `h.resolveLang(ctx, from)`, leave it for Task 2 — or reuse `lang` there; do NOT touch schedule in this task.)
- OnCallback (~line 208):
  ```go
  lang, loc := h.resolveLangLoc(ctx, &cq.From)
  if reply, handled := h.checker.OnCallback(ctx, cq.From.ID, cq.Data, lang, loc); handled && cq.Message.Message != nil {
  ```
- The checker `Start` site (~line 146) stays `h.checker.Start(ctx, from.ID, h.resolveLang(ctx, from))`.

> If the `lang, loc :=` declaration collides with an existing `lang`/`loc` in that scope, use distinct names or restructure minimally; the build will flag any shadowing.

- [ ] **Step 5: Build + vet + test**

Run: `cd apps/backend && go build ./... && go vet ./internal/platform/checker/ ./internal/infrastructure/telegram/ && go test ./internal/platform/checker/`
Expected: build clean (scheduleview still uses `resolveLang` — untouched), vet clean, checker tests PASS. Update any existing checker test that calls `OnText`/`OnCallback` to pass a `loc` (e.g. `almaty()` or a test `loc`) and keep its assertions.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/infrastructure/telegram/multitenant.go apps/backend/internal/platform/checker/service.go apps/backend/internal/platform/checker/service_test.go
git commit -m "$(cat <<'EOF'
fix(tz): parse + display checker dates in the user's timezone

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: localize `scheduleview` parsing/display

**Files:**
- Modify: `apps/backend/internal/platform/scheduleview/service.go` (`OnText`/`OnCallback`/`period`/`list` gain `loc`)
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` (wire 2 schedule sites)
- Test: `apps/backend/internal/platform/scheduleview/service_test.go`

**Interfaces:**
- Consumes (Task 1): `h.resolveLangLoc`.
- Produces: scheduleview `OnText(...,lang,loc)`, `OnCallback(...,lang,loc)`, `period(...,lang,loc)`, `list(...,lang,loc)`.

- [ ] **Step 1: Write the failing test (`scheduleview/service_test.go`), run red**

`dayWindow`/`parseDate`/`parseRange` are pure + loc-parameterized — assert tz behavior directly:

```go
func TestSchedule_DayWindow_UsesLoc(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	almatyLoc, _ := time.LoadLocation("Asia/Almaty")
	now := time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC) // 07:00 Almaty, 03:00 London
	fL, _, ok := dayWindow(now, "today", london)
	if !ok {
		t.Fatal("today window")
	}
	fA, _, _ := dayWindow(now, "today", almatyLoc)
	if fL.Equal(fA) {
		t.Fatal("today start should differ between London and Almaty for the same instant")
	}
}

func TestSchedule_ParseDate_UsesLoc(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	d, err := parseDate("2026-06-15", london)
	if err != nil {
		t.Fatal(err)
	}
	if d.UTC().Hour() != 23 {
		t.Errorf("London 2026-06-15 midnight should be 23:00 prev-day UTC, got %v", d.UTC())
	}
}
```

Run: `cd apps/backend && go test ./internal/platform/scheduleview/ -run 'TestSchedule_DayWindow_UsesLoc|TestSchedule_ParseDate_UsesLoc' 2>&1 | head` — these pure-helper tests PASS immediately (helpers already loc-aware); they guard the behavior. The threading red→green is the compile error in Step 3 when dispatcher passes `loc`.

- [ ] **Step 2: Thread `loc` through `scheduleview/service.go`**

- `OnText(ctx, telegramID int64, text, lang string, loc *time.Location) (Reply, bool)`:
  - `awaitDate`: `parseDate(text, loc)`; then `s.list(ctx, st, d, d.AddDate(0,0,1), text, false, lang, loc)`.
  - `awaitRange`: `parseRange(text, loc)`; then `s.list(ctx, st, from, to, text, false, lang, loc)`.
  - `awaitSearch` → `s.search(...)` unchanged (no loc).
- `OnCallback(ctx, telegramID int64, data, lang string, loc *time.Location) (Reply, bool)`:
  - `sched:d:` → `s.period(ctx, telegramID, ..., lang, loc)`.
  - `sched:pick:` → `s.pick(...)`, `sched:periods` → `s.periods(...)`, `sched:back` → `s.Start(ctx, telegramID, lang)` — all unchanged (no loc).
- `period(ctx, telegramID int64, kind, lang string, loc *time.Location) Reply`:
  - the `date`/`range` cases return prompts (unchanged); the default computes
    `from, to, ok := dayWindow(time.Now(), kind, loc)` (was `almaty()`) and returns
    `s.list(ctx, st, from, to, periodLabel(kind, lang), true, lang, loc)`.
- `list(ctx, st *State, from, to time.Time, period string, edit bool, lang string, loc *time.Location) Reply`:
  - change `scheduleText(st.EmployeeEmail, period, ms, time.Now(), almaty(), lang)` →
    `scheduleText(st.EmployeeEmail, period, ms, time.Now(), loc, lang)`.

> `Start`, `search`, `pick`, `periods`, `periodReply`, `periodLabel` are unchanged (no date parse/display). `almaty()` stays defined as the resolver fallback.

- [ ] **Step 3: Wire the 2 schedule dispatcher sites in `multitenant.go`**

- OnText (~line 89):
  ```go
  lang, loc := h.resolveLangLoc(ctx, from)
  if reply, handled := h.schedule.OnText(ctx, from.ID, text, lang, loc); handled {
  ```
- OnCallback (~line 203):
  ```go
  lang, loc := h.resolveLangLoc(ctx, &cq.From)
  if reply, handled := h.schedule.OnCallback(ctx, cq.From.ID, cq.Data, lang, loc); handled && cq.Message.Message != nil {
  ```
- The schedule `Start` site (~line 138) stays `h.schedule.Start(ctx, from.ID, h.resolveLang(ctx, from))`.

> If a `lang`/`loc` from the checker block is already in scope at a shared point, ensure each FSM's call uses its own freshly-resolved pair (the checker and schedule callback blocks are separate `if strings.HasPrefix(...)` branches, so `lang, loc :=` in each branch is independent).

- [ ] **Step 4: Build + vet + test**

Run: `cd apps/backend && go build ./... && go vet ./internal/platform/scheduleview/ ./internal/infrastructure/telegram/ && go test ./internal/platform/scheduleview/`
Expected: build clean, vet clean, tests PASS. Update any existing scheduleview test calling `OnText`/`OnCallback`/`period`/`list` to pass a `loc` and keep assertions.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/platform/scheduleview/service.go apps/backend/internal/infrastructure/telegram/multitenant.go apps/backend/internal/platform/scheduleview/service_test.go
git commit -m "$(cat <<'EOF'
fix(tz): parse + display scheduleview dates in the user's timezone

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- `resolveLangLoc` (one lookup → lang + loc, Almaty→UTC fallback) → Task 1 Step 1. ✓
- checker parse (`setRange`/`parseRange`) + display (`duration`/`formatSlots`) use user `loc` → Task 1. ✓
- scheduleview parse (`parseDate`/`parseRange`), `dayWindow` (today/tomorrow/upcoming), display (`scheduleText`) use user `loc` → Task 2. ✓
- Helpers unchanged (already loc-parameterized); only `almaty()` call-site args become `loc` → Tasks 1 & 2. ✓
- `Start` stays `resolveLang`; dispatcher OnText/OnCallback sites use `resolveLangLoc` → Tasks 1 & 2 Steps 4/3. ✓
- `almaty()` retained as fallback; weekday names out of scope → constraints + Task notes. ✓
- Each commit builds repo-wide (Task 1 adds resolveLangLoc + checker only; scheduleview still on resolveLang until Task 2) → task ordering. ✓

**Placeholder scan:** No TBD/TODO. Per-method instructions name the exact call-site change (signature + the `almaty()`→`loc` line); the methods are otherwise unchanged from their post-C1 state, so targeted instructions are clearer than re-pasting full bodies. Tests show complete code.

**Type consistency:** `resolveLangLoc(ctx, *models.User) (string, *time.Location)` defined in Task 1, used in Tasks 1 & 2. checker `OnText`/`OnCallback`/`setRange`/`duration` and scheduleview `OnText`/`OnCallback`/`period`/`list` all gain a trailing `loc *time.Location` consistently between service.go, their tests, and the dispatcher call sites. `formatSlots`/`scheduleText`/`parseRange`/`parseDate`/`dayWindow` signatures are untouched (already `*time.Location`).
