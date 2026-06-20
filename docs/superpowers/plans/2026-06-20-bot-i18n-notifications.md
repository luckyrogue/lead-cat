# Bot i18n — Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Localize the two bot notification surfaces (meeting notifier + Telegram reminders) per recipient, and render meeting-notifier times in each recipient's timezone instead of the organization's.

**Architecture:** New `boti18n` catalog files supply ru/en/kk strings. `meeting_notifier` builders take a `lang` (header resolved from the catalog); the notifier moves text construction into the per-recipient loop, keyed on `r.Language`/`r.Timezone` (org TZ as fallback). `reminder_scheduler`'s `message`/`offsetLabel` take a `lang` and the send site passes the target's language.

**Tech Stack:** Go, `boti18n` (from sub-project A), `meetingrecipients.Resolve`, zap.

## Global Constraints

- **`boti18n.T(lang, key, args...)`** for every localized string; ru | en | kk; default ru. `%[1]s`/`%[1]d` explicit-index verbs for params.
- **`ru` catalog values are the current hardcoded strings verbatim** (e.g. `"📅 Новая встреча"`, `"⏰ Напоминание: встреча через %[1]s!"`, `"1 час"`); en/kk are new.
- **Per-recipient timezone** (meeting notifier only): `time.LoadLocation(cmp.Or(recipientTZ, orgTZ, "Asia/Almaty"))`, warn + `time.UTC` on error — same fallback chain as today, recipient first. Reminders contain no date/time → no timezone change.
- **Only header/label text is localized**; dates, `«»`, `🗓`/`🔗`/`⏰` emoji, and `tzLabel` stay language-neutral.
- **`buildEventMessage` stays a pure formatter taking a resolved header string** — the wrappers resolve the catalog key. Do not push the key into `buildEventMessage`.
- **Every new catalog key must have ru/en/kk** — the existing `boti18n` `TestCatalog_AllKeysHaveAllLangs` enforces it.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: `meeting_notifier` — localized, per-recipient language + timezone

**Files:**
- Create: `apps/backend/internal/platform/boti18n/catalog_notifier.go`
- Modify: `apps/backend/internal/platform/meeting_notifier/message.go`
- Modify: `apps/backend/internal/platform/meeting_notifier/notifier.go`
- Test: `apps/backend/internal/platform/meeting_notifier/message_test.go` (update existing calls + add a localized case)
- Test: `apps/backend/internal/platform/meeting_notifier/notifier_test.go` (add a per-recipient lang/tz test)

**Interfaces:**
- Consumes (sub-project A): `boti18n.T`.
- Produces: `buildMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string` and the analogous `buildUpdatedMessage`/`buildRemovedMessage(lang, name, startsAt, loc)`/`buildCancelledMessage(lang, name, startsAt, loc)`; a `(*Notifier).recipientLoc(recipientTZ, orgTZ string) *time.Location` helper.

- [ ] **Step 1: Create `catalog_notifier.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		"notif.created":   {"ru": "📅 Новая встреча", "en": "📅 New meeting", "kk": "📅 Жаңа кездесу"},
		"notif.updated":   {"ru": "✏️ Встреча изменена", "en": "✏️ Meeting updated", "kk": "✏️ Кездесу өзгертілді"},
		"notif.removed":   {"ru": "➖ Вас удалили из встречи", "en": "➖ You were removed from a meeting", "kk": "➖ Сіз кездесуден шығарылдыңыз"},
		"notif.cancelled": {"ru": "❌ Встреча отменена", "en": "❌ Meeting cancelled", "kk": "❌ Кездесу болдырылмады"},
		"notif.added":     {"ru": "➕ Вас добавили на встречу", "en": "➕ You were added to a meeting", "kk": "➕ Сіз кездесуге қосылдыңыз"},
	})
}
```

- [ ] **Step 2: Update `message_test.go` to the new signatures + add a localized assertion**

Replace the two existing builder-call tests so they pass `"ru"` (ru assertions unchanged), and add an English case:

```go
func TestBuildMessage_WithAndWithoutLink(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC) // 10:00 Almaty
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)  // 10:30
	with := buildMessage("ru", "Sync", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"📅 Новая встреча", "«Sync»", "01.06.2026", "10:00–10:30", "UTC+5", "🔗 https://meet.google.com/abc"} {
		if !strings.Contains(with, want) {
			t.Fatalf("missing %q in:\n%s", want, with)
		}
	}
	without := buildMessage("ru", "Sync", "", start, end, loc)
	if strings.Contains(without, "🔗") {
		t.Fatalf("no link icon expected without meet link:\n%s", without)
	}
}

func TestBuildUpdatedRemovedCancelled(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)
	if !strings.Contains(buildUpdatedMessage("ru", "S", "", start, end, loc), "✏️ Встреча изменена") {
		t.Fatal("updated header")
	}
	if !strings.Contains(buildRemovedMessage("ru", "S", start, loc), "➖") {
		t.Fatal("removed header")
	}
	if !strings.Contains(buildCancelledMessage("ru", "S", start, loc), "❌ Встреча отменена") {
		t.Fatal("cancelled header")
	}
}

func TestBuildMessage_Localized(t *testing.T) {
	loc := almaty(t)
	start := time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC)
	en := buildMessage("en", "Sync", "", start, end, loc)
	if !strings.Contains(en, "📅 New meeting") || strings.Contains(en, "Новая встреча") {
		t.Fatalf("expected English header, got:\n%s", en)
	}
	// Non-header content stays neutral.
	if !strings.Contains(en, "«Sync»") || !strings.Contains(en, "01.06.2026") {
		t.Fatalf("neutral content missing:\n%s", en)
	}
}
```

> Keep the rest of `TestBuildUpdatedRemovedCancelled` (if it asserts more lines) intact; only the builder calls gained the leading `"ru"`.

- [ ] **Step 3: Run the message tests to verify they fail**

Run: `cd apps/backend && go test ./internal/platform/meeting_notifier/ -run 'TestBuildMessage|TestBuildUpdated' 2>&1 | head`
Expected: FAIL — builders don't yet take a `lang` argument.

- [ ] **Step 4: Update `message.go`**

Add the import and thread `lang` through the wrappers (keep `buildEventMessage` and `tzLabel` unchanged):

```go
package meeting_notifier

import (
	"fmt"
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

func buildEventMessage(header, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	e := endsAt.In(loc)
	msg := fmt.Sprintf("%s\n«%s»\n🗓 %s, %s–%s (%s)",
		header,
		name,
		s.Format("02.01.2006"),
		s.Format("15:04"),
		e.Format("15:04"),
		tzLabel(s))
	if meetLink != "" {
		msg += "\n🔗 " + meetLink
	}
	return msg
}

func buildMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage(boti18n.T(lang, "notif.created"), name, meetLink, startsAt, endsAt, loc)
}

func buildUpdatedMessage(lang, name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage(boti18n.T(lang, "notif.updated"), name, meetLink, startsAt, endsAt, loc)
}

func buildRemovedMessage(lang, name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("%s\n«%s»\n🗓 %s (%s)", boti18n.T(lang, "notif.removed"), name, s.Format("02.01.2006"), tzLabel(s))
}

func buildCancelledMessage(lang, name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("%s\n«%s»\n🗓 %s (%s)", boti18n.T(lang, "notif.cancelled"), name, s.Format("02.01.2006"), tzLabel(s))
}

func tzLabel(t time.Time) string {
	_, off := t.Zone()
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	h, m := off/3600, (off%3600)/60
	if m == 0 {
		return fmt.Sprintf("UTC%s%d", sign, h)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, h, m)
}
```

- [ ] **Step 5: Run the message tests to verify they pass**

Run: `cd apps/backend && go test ./internal/platform/meeting_notifier/ -run 'TestBuildMessage|TestBuildUpdated' -v`
Expected: PASS (incl. `TestBuildMessage_Localized`).

- [ ] **Step 6: Add the per-recipient test, then run it red**

Append to `notifier_test.go`:

```go
func TestHandleCreated_PerRecipientLangAndTZ(t *testing.T) {
	fs := baseStore()
	// Two participants in different languages and timezones.
	fs.participants = []postgres.MeetingParticipant{{Email: "ru@x.io"}, {Email: "en@x.io"}}
	fs.byEmail = map[string]postgres.BotUser{
		"ru@x.io": {TelegramID: 601, Email: "ru@x.io", Language: "ru", Timezone: "Asia/Almaty"},
		"en@x.io": {TelegramID: 602, Email: "en@x.io", Language: "en", Timezone: "Europe/London"},
	}
	fs.meeting.OrganizerUserID = nil // only the two participants
	fs.claimed = map[int64]bool{601: true, 602: true}
	snd := &fakeSender{}
	if err := newNotifier(fs, snd).HandleCreated(context.Background(), uuid.New(), fs.meeting.ID); err != nil {
		t.Fatalf("created: %v", err)
	}
	if len(snd.sent) != 2 {
		t.Fatalf("want 2 sends, got %d: %+v", len(snd.sent), snd.sent)
	}
	byChat := map[int64]string{}
	for _, m := range snd.sent {
		byChat[m.ChatID] = m.Text
	}
	// 10:00 Almaty (UTC+5) for ru; 06:00 London (UTC+1, BST) for en.
	if !strings.Contains(byChat[601], "📅 Новая встреча") || !strings.Contains(byChat[601], "10:00") {
		t.Fatalf("ru recipient text wrong: %q", byChat[601])
	}
	if !strings.Contains(byChat[602], "📅 New meeting") || !strings.Contains(byChat[602], "06:00") {
		t.Fatalf("en recipient text wrong: %q", byChat[602])
	}
}
```

Run: `cd apps/backend && go test ./internal/platform/meeting_notifier/ -run TestHandleCreated_PerRecipientLangAndTZ 2>&1 | head`
Expected: FAIL — the notifier still builds one org-TZ Russian text for all recipients (and `buildMessage` arity changed, so it won't compile until Step 7).

- [ ] **Step 7: Update `notifier.go` — per-recipient loop + `recipientLoc` helper**

Add the helper (near `New`) and rework the three loop handlers + `notifyParticipant`. The `cmp`, `time`, `zap` imports are already present.

```go
// recipientLoc resolves a recipient's display timezone, falling back to the
// organization TZ then Asia/Almaty; on a load error it warns and uses UTC.
func (n *Notifier) recipientLoc(recipientTZ, orgTZ string) *time.Location {
	tz := cmp.Or(recipientTZ, orgTZ, "Asia/Almaty")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		n.log.Warn("load location", zap.String("tz", tz), zap.Error(err))
		return time.UTC
	}
	return loc
}
```

In `HandleCreated`: remove the pre-loop `loc, err := time.LoadLocation(...)` block and the `text := buildMessage(...)` line; build inside the loop. The loop body becomes:

```go
	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		claimed, err := n.store.TryClaimReminder(ctx, m.ID, r.TelegramID, postgres.ReminderOffsetCreated)
		if err != nil {
			return fmt.Errorf("claim reminder: %w", err)
		}
		if !claimed {
			continue
		}
		loc := n.recipientLoc(r.Timezone, w.TZ)
		text := buildMessage(r.Language, m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting created",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
```

In `HandleUpdated`: same shape — drop the pre-loop `loc`/`text`, and inside the loop add `loc := n.recipientLoc(r.Timezone, w.TZ)` then `text := buildUpdatedMessage(r.Language, m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)` before the send.

In `HandleCancelled`: same — inside the loop add `loc := n.recipientLoc(r.Timezone, w.TZ)` then `text := buildCancelledMessage(r.Language, m.Name, m.StartsAt, loc)` before the send.

In `notifyParticipant`: replace the pre-existing `loc, err := time.LoadLocation(...)` block with `loc := n.recipientLoc(u.Timezone, w.TZ)` (placed after the `u, err := n.store.GetBotUserByEmail(...)` lookup so `u.Timezone` is known), and change the text construction to:

```go
	var text string
	if added {
		text = buildEventMessage(boti18n.T(u.Language, "notif.added"), m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
	} else {
		text = buildRemovedMessage(u.Language, m.Name, m.StartsAt, loc)
	}
```

Add the boti18n import to `notifier.go`:

```go
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
```

> Each handler still loads `w` (the organization) — keep that; only its `loc` derivation moves to `recipientLoc(..., w.TZ)`. After removing the per-handler `time.LoadLocation` blocks, confirm `time` and `cmp` are still used (they are: `recipientLoc` uses both, and `m.StartsAt` etc. are `time.Time`).

- [ ] **Step 8: Run the full meeting_notifier package + vet**

Run: `cd apps/backend && go test ./internal/platform/meeting_notifier/ -v && go vet ./internal/platform/meeting_notifier/`
Expected: PASS — existing tests (recipients default to empty language → ru header, so unchanged) plus `TestBuildMessage_Localized` and `TestHandleCreated_PerRecipientLangAndTZ`; vet clean.

- [ ] **Step 9: Run the boti18n coverage test + commit**

```bash
cd apps/backend && go test ./internal/platform/boti18n/ -run TestCatalog_AllKeysHaveAllLangs && cd /Users/temirlan/Workspace/in-house/lead-cat
git add apps/backend/internal/platform/boti18n/catalog_notifier.go apps/backend/internal/platform/meeting_notifier/message.go apps/backend/internal/platform/meeting_notifier/notifier.go apps/backend/internal/platform/meeting_notifier/message_test.go apps/backend/internal/platform/meeting_notifier/notifier_test.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize meeting notifications per-recipient (lang + timezone)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: `reminder_scheduler` — localized Telegram reminders

**Files:**
- Create: `apps/backend/internal/platform/boti18n/catalog_reminder.go`
- Modify: `apps/backend/internal/platform/reminder_scheduler/reminder.go`
- Modify: `apps/backend/internal/platform/reminder_scheduler/scheduler.go` (send site)
- Test: `apps/backend/internal/platform/reminder_scheduler/reminder_test.go` (new)

**Interfaces:**
- Consumes (sub-project A): `boti18n.T`. (existing) `reminderTarget.Language` at the send site.
- Produces: `offsetLabel(min int, lang string) string`, `message(name, meetLink string, offset int, lang string) string`.

- [ ] **Step 1: Create `catalog_reminder.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		"reminder.telegram":     {"ru": "⏰ Напоминание: встреча через %[1]s!", "en": "⏰ Reminder: meeting in %[1]s!", "kk": "⏰ Еске салу: кездесу %[1]s кейін!"},
		"reminder.offset.10m":   {"ru": "10 минут", "en": "10 minutes", "kk": "10 минут"},
		"reminder.offset.15m":   {"ru": "15 минут", "en": "15 minutes", "kk": "15 минут"},
		"reminder.offset.30m":   {"ru": "30 минут", "en": "30 minutes", "kk": "30 минут"},
		"reminder.offset.1h":    {"ru": "1 час", "en": "1 hour", "kk": "1 сағат"},
		"reminder.offset.2h":    {"ru": "2 часа", "en": "2 hours", "kk": "2 сағат"},
		"reminder.offset.1d":    {"ru": "1 день", "en": "1 day", "kk": "1 күн"},
		"reminder.offset.n_min": {"ru": "%[1]d мин", "en": "%[1]d min", "kk": "%[1]d мин"},
	})
}
```

- [ ] **Step 2: Write the failing test `reminder_test.go`**

```go
package reminder_scheduler

import (
	"strings"
	"testing"
)

func TestOffsetLabel_Localized(t *testing.T) {
	if got := offsetLabel(60, "ru"); got != "1 час" {
		t.Errorf("ru 60 = %q", got)
	}
	if got := offsetLabel(60, "en"); got != "1 hour" {
		t.Errorf("en 60 = %q", got)
	}
	if got := offsetLabel(1440, "kk"); got != "1 күн" {
		t.Errorf("kk 1440 = %q", got)
	}
	// off-list value uses the n_min default
	if got := offsetLabel(7, "en"); got != "7 min" {
		t.Errorf("en 7 = %q", got)
	}
}

func TestMessage_Localized(t *testing.T) {
	en := message("Sync", "https://meet.google.com/abc", 60, "en")
	if !strings.Contains(en, "in 1 hour") || !strings.Contains(en, "«Sync»") || !strings.Contains(en, "🔗 https://meet.google.com/abc") {
		t.Fatalf("en message wrong:\n%s", en)
	}
	ru := message("Sync", "", 30, "ru")
	if !strings.Contains(ru, "через 30 минут") || strings.Contains(ru, "🔗") {
		t.Fatalf("ru message wrong:\n%s", ru)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd apps/backend && go test ./internal/platform/reminder_scheduler/ -run 'TestOffsetLabel|TestMessage' 2>&1 | head`
Expected: FAIL — `offsetLabel`/`message` don't take a `lang` argument.

- [ ] **Step 4: Update `reminder.go`**

Add the import and thread `lang` (leave `dueOffsets` unchanged):

```go
package reminder_scheduler

import (
	"fmt"
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

func dueOffsets(now, startsAt time.Time, offsets []int) []int {
	var due []int
	for _, off := range offsets {
		threshold := startsAt.Add(-time.Duration(off) * time.Minute)
		if !now.Before(threshold) && now.Before(startsAt) {
			due = append(due, off)
		}
	}
	return due
}

func offsetLabel(min int, lang string) string {
	switch min {
	case 10:
		return boti18n.T(lang, "reminder.offset.10m")
	case 15:
		return boti18n.T(lang, "reminder.offset.15m")
	case 30:
		return boti18n.T(lang, "reminder.offset.30m")
	case 60:
		return boti18n.T(lang, "reminder.offset.1h")
	case 120:
		return boti18n.T(lang, "reminder.offset.2h")
	case 1440:
		return boti18n.T(lang, "reminder.offset.1d")
	default:
		return boti18n.T(lang, "reminder.offset.n_min", min)
	}
}

func message(name, meetLink string, offset int, lang string) string {
	m := fmt.Sprintf("%s\n«%s»", boti18n.T(lang, "reminder.telegram", offsetLabel(offset, lang)), name)
	if meetLink != "" {
		m += "\n🔗 " + meetLink
	}
	return m
}
```

> `strconv` is no longer used in `reminder.go` (the `n_min` default now formats via `boti18n.T`); drop the `strconv` import.

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd apps/backend && go test ./internal/platform/reminder_scheduler/ -run 'TestOffsetLabel|TestMessage' -v`
Expected: PASS.

- [ ] **Step 6: Update the send site in `scheduler.go`**

Change the Telegram reminder send (currently `Text: message(m.Name, m.MeetLink, off)`) to pass the target's language — `t` is the `reminderTarget` in scope at that loop:

```go
					Text:   message(m.Name, m.MeetLink, off, t.Language),
```

- [ ] **Step 7: Build + vet + coverage**

Run: `cd apps/backend && go build ./... && go vet ./internal/platform/reminder_scheduler/ && go test ./internal/platform/boti18n/ -run TestCatalog_AllKeysHaveAllLangs`
Expected: build clean, vet clean, coverage test PASS (reminder.* keys all have ru/en/kk).

- [ ] **Step 8: Commit**

```bash
git add apps/backend/internal/platform/boti18n/catalog_reminder.go apps/backend/internal/platform/reminder_scheduler/reminder.go apps/backend/internal/platform/reminder_scheduler/scheduler.go apps/backend/internal/platform/reminder_scheduler/reminder_test.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize Telegram reminders (offset labels + message)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- meeting_notifier headers localized via `notif.*` catalog → Task 1 (catalog + `message.go`). ✓
- Per-recipient language (build inside loop, `r.Language`; participant `u.Language`) → Task 1 notifier rework. ✓
- Per-recipient timezone (`recipientLoc(r.Timezone, w.TZ)`, org fallback, UTC on error) → Task 1 helper. ✓
- reminder Telegram localized (`offsetLabel`/`message` + `reminder.*`), send passes `t.Language` → Task 2. ✓
- ru values verbatim; en/kk new → Tasks 1 & 2 catalogs. ✓
- `buildEventMessage` kept as a pure header-string formatter → Task 1 Step 4. ✓
- Tests: builder localization, two-recipient lang+tz notifier test, offsetLabel/message tests, coverage test → Tasks 1 & 2. ✓
- Out of scope (FSMs C, NL agent D, email reminders, plurals) → untouched. ✓

**Placeholder scan:** No TBD/TODO. Every code step shows complete code. The `> Note` blocks flag real mechanics (keep `w` load, drop `strconv`, confirm `time`/`cmp` still used) — not deferred work.

**Type consistency:** `buildMessage`/`buildUpdatedMessage` take `(lang, name, meetLink, startsAt, endsAt, loc)`; `buildRemovedMessage`/`buildCancelledMessage` take `(lang, name, startsAt, loc)` — consistent between `message.go`, `message_test.go`, and the `notifier.go` call sites. `recipientLoc(recipientTZ, orgTZ string) *time.Location` defined and called consistently. `offsetLabel(min int, lang string)` and `message(name, meetLink string, offset int, lang string)` consistent between `reminder.go`, `reminder_test.go`, and the `scheduler.go` send site. Catalog keys referenced in code (`notif.*`, `reminder.*`) all exist in the catalog files.
