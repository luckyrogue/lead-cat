# Increment A — bot-FSM meeting field editing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a registered user edit one of their upcoming meetings via a Telegram bot FSM — pick meeting → change fields (date/time, dept, type, host, description, recurrence) → apply, which patches the Google event, recomputes the name, persists, and DMs participants.

**Architecture:** A pure FSM package `meetingedit` (Redis session, keyboards) drives the conversation and calls an application command `Services.UpdateMeeting` (mirrors `CreateMeeting`: ACL → validate → Google `UpdateEvent` Patch → persist → enqueue `meeting:updated`). The notification reuses `meeting_notifier` + `meetingrecipients` from §5a. Bot identity → meetings is resolved via `platform_users.telegram_id`.

**Tech Stack:** Go, go-telegram/bot, asynq, pgx, zap, google/uuid, google.golang.org/api/calendar/v3, redis.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-edit-fields-design.md`

**Conventions:**

- Run Go commands from `backend/` with the `env -u GOROOT` prefix. Module: `github.com/Jaryq-Lab/notify-bot`.
- Build check: `env -u GOROOT go build ./...`

---

## Task 1: `CalendarService.UpdateEvent` (port + stub + Google adapter)

**Files:**

- Modify: `backend/internal/application/calendar.go`
- Modify: `backend/internal/infrastructure/calendar/stub/stub.go`
- Modify: `backend/internal/infrastructure/calendar/google/adapter.go`
- Test: `backend/internal/infrastructure/calendar/google/adapter_test.go`

- [ ] **Step 1: Add the method to the port.** In `backend/internal/application/calendar.go`, extend the `CalendarService` interface:

```go
type CalendarService interface {
	CreateEvent(ctx context.Context, e CalendarEvent) (CalendarResult, error)
	UpdateEvent(ctx context.Context, eventID string, e CalendarEvent) error
	DeleteEvent(ctx context.Context, eventID string) error
}
```

- [ ] **Step 2: Implement the stub.** In `backend/internal/infrastructure/calendar/stub/stub.go`, add (after `DeleteEvent`):

```go
func (s *Service) UpdateEvent(_ context.Context, _ string, _ application.CalendarEvent) error { return nil }
```

- [ ] **Step 3: Write the failing `buildPatch` test.** In `backend/internal/infrastructure/calendar/google/adapter_test.go`, add:

```go
func TestBuildPatch(t *testing.T) {
	e := application.CalendarEvent{
		Title:          "Разработка | Планёрка",
		Description:    "desc",
		Start:          time.Date(2026, 6, 1, 14, 0, 0, 0, time.UTC),
		End:            time.Date(2026, 6, 1, 15, 0, 0, 0, time.UTC),
		AttendeeEmails: []string{"a@x"},
	}
	ev := buildPatch(e)
	if ev.Summary != "Разработка | Планёрка" || ev.Description != "desc" {
		t.Fatalf("summary/description not set: %+v", ev)
	}
	if ev.Start == nil || ev.End == nil {
		t.Fatal("start/end must be set")
	}
	// Patch must NOT touch attendees or conference data (preserves guests + Meet link).
	if ev.Attendees != nil {
		t.Fatalf("patch must not set attendees, got %+v", ev.Attendees)
	}
	if ev.ConferenceData != nil {
		t.Fatal("patch must not set conference data")
	}
}
```

(The file already imports `testing`, `time`, and `application` for `TestBuildEvent`. Confirm; add any missing import.)

- [ ] **Step 4: Run it, verify it fails.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/infrastructure/calendar/google/ -run TestBuildPatch -v`
      Expected: FAIL — `undefined: buildPatch`.

- [ ] **Step 5: Implement `buildPatch` + `UpdateEvent`.** In `backend/internal/infrastructure/calendar/google/adapter.go`, add (after `DeleteEvent`):

```go
// buildPatch maps a CalendarEvent to a partial Google event for Events.Patch.
// It sets only the fields edited here; omitting Attendees and ConferenceData
// leaves the guest list and the Meet link untouched.
func buildPatch(e application.CalendarEvent) *calendar.Event {
	return &calendar.Event{
		Summary:     e.Title,
		Description: e.Description,
		Start:       &calendar.EventDateTime{DateTime: e.Start.Format(time.RFC3339)},
		End:         &calendar.EventDateTime{DateTime: e.End.Format(time.RFC3339)},
	}
}

func (a *adapter) UpdateEvent(ctx context.Context, eventID string, e application.CalendarEvent) error {
	_, err := a.svc.Events.
		Patch(a.calendarID, eventID, buildPatch(e)).
		SendUpdates("all").
		Context(ctx).
		Do()
	return err
}
```

- [ ] **Step 6: Run tests + build.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/infrastructure/calendar/... -v && env -u GOROOT go build ./...`
      Expected: `TestBuildPatch`, `TestBuildEvent`, stub tests PASS; build OK.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/calendar.go backend/internal/infrastructure/calendar/ && git commit -m "feat(meetings): CalendarService.UpdateEvent (Google Patch)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: repo — `UpdateMeeting`, `MeetingWithTZ`, `ListMeetingsByOrganizerTelegram`

**Files:**

- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Add a qualified column list + the `MeetingWithTZ` type + the two queries.** In `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`, after the existing `meetingCols` const (line ~11), add:

```go
// meetingColsM is meetingCols qualified with the `m` alias for joins.
const meetingColsM = `m.id, m.workspace_id, m.organizer_user_id, m.dept, m.type, m.host,
	m.starts_at, m.ends_at, m.recurrence, m.name, m.description, m.google_event_id, m.meet_link, m.status`

// MeetingWithTZ is a meeting plus its workspace timezone (for bot rendering).
type MeetingWithTZ struct {
	Meeting
	TZ string
}
```

Then add these methods (anywhere after `GetMeeting`):

```go
// UpdateMeeting overwrites the editable fields of a scheduled meeting.
func (s *Store) UpdateMeeting(ctx context.Context, workspaceID, id uuid.UUID, m Meeting) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE meetings SET dept=$3, type=$4, host=$5, starts_at=$6, ends_at=$7,
			recurrence=$8, name=$9, description=$10, updated_at=now()
		WHERE id=$1 AND workspace_id=$2 AND status='scheduled'`,
		id, workspaceID, m.Dept, m.Type, m.Host, m.StartsAt, m.EndsAt, m.Recurrence, m.Name, m.Description)
	return err
}

// ListMeetingsByOrganizerTelegram returns the upcoming scheduled meetings
// organized by the platform user linked to telegramID, each with its workspace TZ.
func (s *Store) ListMeetingsByOrganizerTelegram(ctx context.Context, telegramID int64) ([]MeetingWithTZ, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+meetingColsM+`, w.tz
		FROM meetings m
		JOIN platform_users pu ON pu.id = m.organizer_user_id
		JOIN workspaces w ON w.id = m.workspace_id
		WHERE pu.telegram_id = $1 AND m.status = 'scheduled' AND m.starts_at > now()
		ORDER BY m.starts_at`, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MeetingWithTZ
	for rows.Next() {
		var mt MeetingWithTZ
		if err := rows.Scan(&mt.ID, &mt.WorkspaceID, &mt.OrganizerUserID, &mt.Dept, &mt.Type, &mt.Host,
			&mt.StartsAt, &mt.EndsAt, &mt.Recurrence, &mt.Name, &mt.Description, &mt.GoogleEventID, &mt.MeetLink, &mt.Status,
			&mt.TZ); err != nil {
			return nil, err
		}
		out = append(out, mt)
	}
	return out, rows.Err()
}
```

- [ ] **Step 2: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/`
      Expected: clean. (No DB harness in the postgres package — build/vet is the gate, consistent with the rest of the repo.)

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go && git commit -m "feat(meetings): repo UpdateMeeting + ListMeetingsByOrganizerTelegram

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: queue — `meeting:updated` task

**Files:**

- Modify: `backend/internal/infrastructure/queue/asynq/queue.go`

- [ ] **Step 1: Add the task type, payload, enqueue, and parser.** In `backend/internal/infrastructure/queue/asynq/queue.go`, after the `meeting:created` block (after `ParseMeetingCreated`), add:

```go
const TaskMeetingUpdated = "meeting:updated"

type MeetingUpdatedPayload struct {
	WorkspaceID string `json:"workspace_id"`
	MeetingID   string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingUpdated(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingUpdatedPayload{
		WorkspaceID: workspaceID.String(),
		MeetingID:   meetingID.String(),
	})
	task := asynq.NewTask(TaskMeetingUpdated, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseMeetingUpdated(t *asynq.Task) (MeetingUpdatedPayload, error) {
	var p MeetingUpdatedPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
```

- [ ] **Step 2: Build.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...`
      Expected: OK. (The handler is registered in Task 9.)

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/queue/asynq/queue.go && git commit -m "feat(queue): meeting:updated task

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: notifier — `buildUpdatedMessage` + `HandleUpdated`

**Files:**

- Modify: `backend/internal/platform/meeting_notifier/message.go`
- Modify: `backend/internal/platform/meeting_notifier/message_test.go`
- Modify: `backend/internal/platform/meeting_notifier/notifier.go`

- [ ] **Step 1: Write the failing test.** In `backend/internal/platform/meeting_notifier/message_test.go`, add:

```go
func TestBuildUpdatedMessage(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	end := time.Date(2026, 5, 31, 15, 0, 0, 0, loc)

	m := buildUpdatedMessage("Разработка | Планёрка", "https://meet.google.com/abc", start, end, loc)
	for _, want := range []string{"изменена", "Разработка | Планёрка", "31.05.2026", "14:00–15:00", "UTC+5", "https://meet.google.com/abc"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}
	if strings.Contains(buildUpdatedMessage("X", "", start, end, loc), "🔗") {
		t.Fatal("no link line when meet link empty")
	}
}
```

- [ ] **Step 2: Run it, verify it fails.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -run TestBuildUpdatedMessage -v`
      Expected: FAIL — `undefined: buildUpdatedMessage`.

- [ ] **Step 3: Refactor the builder + add the updated variant.** In `backend/internal/platform/meeting_notifier/message.go`, replace the existing `buildMessage` function with a shared builder plus two thin wrappers (DRY — only the header differs):

```go
// buildEventMessage renders an event DM with the given header line. Times are
// converted to loc; the link line is omitted when meetLink is empty.
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

func buildMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage("📅 Новая встреча", name, meetLink, startsAt, endsAt, loc)
}

func buildUpdatedMessage(name, meetLink string, startsAt, endsAt time.Time, loc *time.Location) string {
	return buildEventMessage("✏️ Встреча изменена", name, meetLink, startsAt, endsAt, loc)
}
```

Leave `tzLabel` unchanged.

- [ ] **Step 4: Run tests, verify they pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v`
      Expected: `TestBuildMessage`, `TestTZLabel`, `TestBuildUpdatedMessage` PASS.

- [ ] **Step 5: Add `HandleUpdated`.** In `backend/internal/platform/meeting_notifier/notifier.go`, add (after `HandleCreated`):

```go
// HandleUpdated DMs the meeting's recipients that it changed. Like HandleCreated
// it returns an error only on read failures (asynq retries before any send);
// sends are best-effort. No dedup: each edit is its own notification.
func (n *Notifier) HandleUpdated(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
	m, err := n.store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return fmt.Errorf("get meeting: %w", err)
	}
	w, err := n.store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}
	loc, err := time.LoadLocation(cmp.Or(w.TZ, "Asia/Almaty"))
	if err != nil {
		n.log.Warn("load location", zap.String("tz", w.TZ), zap.Error(err))
		loc = time.UTC
	}
	text := buildUpdatedMessage(m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting updated",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}
```

(`cmp`, `fmt`, `time`, the bot/uuid/zap imports, `postgres`, `meetingrecipients` are already imported in `notifier.go` from §5a. Confirm `cmp` is present.)

- [ ] **Step 6: Build + test.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v`
      Expected: build OK; all message tests PASS.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meeting_notifier/ && git commit -m "feat(meetings): meeting-updated DM notification

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: application — `applyMeetingUpdate` pure helper

**Files:**

- Create: `backend/internal/application/meeting_update.go`
- Test: `backend/internal/application/meeting_update_test.go`

- [ ] **Step 1: Write the failing test.** Create `backend/internal/application/meeting_update_test.go`:

```go
package application

import (
	"errors"
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

func strp(s string) *string { return &s }

func baseMeeting() postgres.Meeting {
	loc, _ := time.LoadLocation("Asia/Almaty")
	return postgres.Meeting{
		Dept: "Разработка", Type: "Планёрка", Host: "Иванов А.А.",
		StartsAt:   time.Date(2026, 6, 1, 14, 0, 0, 0, loc).UTC(),
		EndsAt:     time.Date(2026, 6, 1, 15, 0, 0, 0, loc).UTC(),
		Recurrence: "once", Description: "old", Name: "old name",
	}
}

func TestApplyMeetingUpdate_DeptOnly(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applyMeetingUpdate(baseMeeting(), UpdateMeetingInput{Dept: strp("Маркетинг")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Dept != "Маркетинг" {
		t.Fatalf("dept = %q", out.Dept)
	}
	// name recomputed; local date is 2026-06-01.
	if out.Name != "Маркетинг | Планёрка | Иванов А.А. | 2026-06-01" {
		t.Fatalf("name = %q", out.Name)
	}
	if !out.StartsAt.Equal(baseMeeting().StartsAt) {
		t.Fatalf("start changed unexpectedly")
	}
}

func TestApplyMeetingUpdate_DateTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applyMeetingUpdate(baseMeeting(), UpdateMeetingInput{
		Date: strp("2026-06-02"), Start: strp("10:00"), End: strp("11:00"),
	}, loc)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 6, 2, 10, 0, 0, 0, loc).UTC()
	if !out.StartsAt.Equal(wantStart) {
		t.Fatalf("start = %v want %v", out.StartsAt, wantStart)
	}
	if out.Name != "Разработка | Планёрка | Иванов А.А. | 2026-06-02" {
		t.Fatalf("name = %q", out.Name)
	}
}

func TestApplyMeetingUpdate_EndBeforeStart(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	_, err := applyMeetingUpdate(baseMeeting(), UpdateMeetingInput{
		Date: strp("2026-06-02"), Start: strp("11:00"), End: strp("10:00"),
	}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestApplyMeetingUpdate_BadRecurrence(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	_, err := applyMeetingUpdate(baseMeeting(), UpdateMeetingInput{Recurrence: strp("hourly")}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestApplyMeetingUpdate_RecurrenceLabelInName(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applyMeetingUpdate(baseMeeting(), UpdateMeetingInput{Recurrence: strp("weekly")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Разработка | Планёрка | Иванов А.А. | 2026-06-01 | Еженедельно" {
		t.Fatalf("name = %q", out.Name)
	}
}
```

- [ ] **Step 2: Run it, verify it fails.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run TestApplyMeetingUpdate -v`
      Expected: FAIL — `undefined: applyMeetingUpdate` / `UpdateMeetingInput`.

- [ ] **Step 3: Implement the helper.** Create `backend/internal/application/meeting_update.go`:

```go
package application

import (
	"fmt"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// UpdateMeetingInput carries optional field overrides (nil = leave unchanged).
// Date/Start/End must be supplied together to change the time.
type UpdateMeetingInput struct {
	Dept        *string
	Type        *string
	Host        *string
	Date        *string // YYYY-MM-DD
	Start       *string // HH:MM
	End         *string // HH:MM
	Recurrence  *string
	Description *string
}

// applyMeetingUpdate applies non-nil overrides to cur, parsing date/time in loc,
// validating via the domain, and recomputing the name (from the LOCAL start, as
// CreateMeeting does). Pure; returns a patched copy or ErrInvalidInput.
func applyMeetingUpdate(cur postgres.Meeting, in UpdateMeetingInput, loc *time.Location) (postgres.Meeting, error) {
	dept := orStr(in.Dept, cur.Dept)
	typ := orStr(in.Type, cur.Type)
	host := orStr(in.Host, cur.Host)
	desc := orStr(in.Description, cur.Description)
	rec := meeting.Recurrence(orStr(in.Recurrence, cur.Recurrence))

	startLocal := cur.StartsAt.In(loc)
	startsAt := cur.StartsAt
	endsAt := cur.EndsAt
	if in.Date != nil && in.Start != nil && in.End != nil {
		s, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.Start, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
		}
		e, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.End, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
		}
		startLocal = s
		startsAt = s.UTC()
		endsAt = e.UTC()
	}

	dom := meeting.Input{Dept: dept, Type: typ, Host: host, StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: desc}
	if err := dom.Validate(); err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out := cur
	out.Dept, out.Type, out.Host = dept, typ, host
	out.Description = desc
	out.Recurrence = string(rec)
	out.StartsAt, out.EndsAt = startsAt, endsAt
	out.Name = meeting.GenerateName(dept, typ, host, startLocal, rec)
	return out, nil
}

func orStr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}
```

- [ ] **Step 4: Run tests, verify they pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run TestApplyMeetingUpdate -v`
      Expected: all five PASS.

- [ ] **Step 5: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/meeting_update.go backend/internal/application/meeting_update_test.go && git commit -m "feat(meetings): applyMeetingUpdate pure helper

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: application — `Services.UpdateMeeting` + `ListEditableMeetings`

**Files:**

- Modify: `backend/internal/application/meeting_service.go`

- [ ] **Step 1: Add the command + query.** In `backend/internal/application/meeting_service.go`, add (after `CreateMeeting`, before `CancelMeeting`):

```go
// UpdateMeeting applies field overrides to a meeting (organizer or workspace
// owner only): validates, recomputes the name, patches the Google event, persists,
// and enqueues a change notification. Mirrors CreateMeeting.
func (s *Services) UpdateMeeting(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in UpdateMeetingInput) (postgres.Meeting, error) {
	cur, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return postgres.Meeting{}, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, err
	}
	isOwner := w.OwnerUserID != nil && *w.OwnerUserID == userID
	isOrganizer := cur.OrganizerUserID != nil && *cur.OrganizerUserID == userID
	if !isOwner && !isOrganizer {
		return postgres.Meeting{}, ErrForbidden
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		return postgres.Meeting{}, fmt.Errorf("bad timezone: %w", err)
	}
	updated, err := applyMeetingUpdate(cur, in, loc)
	if err != nil {
		return postgres.Meeting{}, err
	}
	if updated.GoogleEventID != "" {
		calSvc, err := s.Calendar.For(ctx, workspaceID)
		if err != nil {
			return postgres.Meeting{}, err
		}
		if err := calSvc.UpdateEvent(ctx, updated.GoogleEventID, CalendarEvent{
			Title: updated.Name, Description: updated.Description,
			Start: updated.StartsAt, End: updated.EndsAt,
		}); err != nil {
			return postgres.Meeting{}, fmt.Errorf("calendar: %w", err)
		}
	}
	if err := s.Store.UpdateMeeting(ctx, workspaceID, meetingID, updated); err != nil {
		return postgres.Meeting{}, err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueMeetingUpdated(ctx, workspaceID, meetingID); err != nil && s.Log != nil {
			s.Log.Warn("enqueue meeting updated",
				zap.String("workspace_id", workspaceID.String()),
				zap.String("meeting_id", meetingID.String()),
				zap.Error(err))
		}
	}
	return updated, nil
}

// ListEditableMeetings returns the upcoming meetings the Telegram user organizes,
// each with its workspace timezone (for the bot edit FSM).
func (s *Services) ListEditableMeetings(ctx context.Context, telegramID int64) ([]postgres.MeetingWithTZ, error) {
	return s.Store.ListMeetingsByOrganizerTelegram(ctx, telegramID)
}
```

(`fmt`, `time`, `uuid`, `zap`, `postgres` are already imported in `meeting_service.go`. `orDefault` already exists in this file.)

- [ ] **Step 2: Build + run application tests.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/application/ -v`
      Expected: build OK; `TestApplyMeetingUpdate*` PASS. (The orchestration's storage path is build-verified — no DB harness, consistent with `CreateMeeting`.)

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/meeting_service.go && git commit -m "feat(meetings): Services.UpdateMeeting + ListEditableMeetings

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: `meetingedit` — date/time parser

**Files:**

- Create: `backend/internal/platform/meetingedit/parse.go`
- Test: `backend/internal/platform/meetingedit/parse_test.go`

- [ ] **Step 1: Write the failing test.** Create `backend/internal/platform/meetingedit/parse_test.go`:

```go
package meetingedit

import "testing"

func TestParseDateTime_OK(t *testing.T) {
	for _, in := range []string{"2026-06-01 14:00–15:00", "2026-06-01 14:00-15:00"} {
		d, s, e, err := parseDateTime(in)
		if err != nil {
			t.Fatalf("%q: unexpected error %v", in, err)
		}
		if d != "2026-06-01" || s != "14:00" || e != "15:00" {
			t.Fatalf("%q -> %q %q %q", in, d, s, e)
		}
	}
}

func TestParseDateTime_Errors(t *testing.T) {
	bad := []string{
		"",                        // empty
		"2026-06-01",              // no time range
		"2026/06/01 14:00-15:00",  // bad date
		"2026-06-01 14:00",        // single time
		"2026-06-01 9:00-10:00",   // not HH:MM
		"2026-06-01 15:00-14:00",  // end before start
		"2026-06-01 14:00-14:00",  // equal
	}
	for _, in := range bad {
		if _, _, _, err := parseDateTime(in); err == nil {
			t.Fatalf("%q: expected error", in)
		}
	}
}
```

- [ ] **Step 2: Run it, verify it fails.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -v`
      Expected: FAIL — `undefined: parseDateTime`.

- [ ] **Step 3: Implement the parser.** Create `backend/internal/platform/meetingedit/parse.go`:

```go
// Package meetingedit drives the Telegram bot FSM for editing a meeting's fields.
package meetingedit

import (
	"fmt"
	"strings"
	"time"
)

// parseDateTime parses "YYYY-MM-DD HH:MM–HH:MM" (en dash or hyphen) into the
// override strings. It validates the formats and that end is after start.
func parseDateTime(text string) (date, start, end string, err error) {
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return "", "", "", fmt.Errorf("формат: ГГГГ-ММ-ДД ЧЧ:ММ–ЧЧ:ММ")
	}
	d, err := time.Parse("2006-01-02", fields[0])
	if err != nil {
		return "", "", "", fmt.Errorf("неверная дата (нужно ГГГГ-ММ-ДД)")
	}
	rng := strings.NewReplacer("–", "-", "—", "-").Replace(fields[1])
	parts := strings.SplitN(rng, "-", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("нужно время начала и конца: ЧЧ:ММ–ЧЧ:ММ")
	}
	st, err := time.Parse("15:04", strings.TrimSpace(parts[0]))
	if err != nil {
		return "", "", "", fmt.Errorf("неверное время начала (ЧЧ:ММ)")
	}
	en, err := time.Parse("15:04", strings.TrimSpace(parts[1]))
	if err != nil {
		return "", "", "", fmt.Errorf("неверное время конца (ЧЧ:ММ)")
	}
	if !en.After(st) {
		return "", "", "", fmt.Errorf("конец должен быть позже начала")
	}
	return d.Format("2006-01-02"), st.Format("15:04"), en.Format("15:04"), nil
}
```

- [ ] **Step 4: Run tests, verify they pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -v`
      Expected: `TestParseDateTime_OK`, `TestParseDateTime_Errors` PASS.

- [ ] **Step 5: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meetingedit/ && git commit -m "feat(meetings): meetingedit date/time parser

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: `meetingedit` — FSM Service, state, sessions, renderers

**Files:**

- Create: `backend/internal/platform/meetingedit/state.go`
- Create: `backend/internal/platform/meetingedit/redis_sessions.go`
- Create: `backend/internal/platform/meetingedit/service.go`
- Test: `backend/internal/platform/meetingedit/service_test.go`

- [ ] **Step 1: Create the state + types.** Create `backend/internal/platform/meetingedit/state.go`:

```go
package meetingedit

// State is the per-user FSM state (stored in Redis between messages).
type State struct {
	Step          string            `json:"step"` // menu | awaiting
	MeetingID     string            `json:"meeting_id"`
	WorkspaceID   string            `json:"workspace_id"`
	UserID        string            `json:"user_id"`
	AwaitingField string            `json:"awaiting_field,omitempty"`
	Cur           map[string]string `json:"cur"`       // current display values
	Overrides     map[string]string `json:"overrides"` // pending edits
}

const (
	stepMenu     = "menu"
	stepAwaiting = "awaiting"
)

// Button is one inline-keyboard button.
type Button struct {
	Text string
	Data string
}

// Reply is what the FSM returns for the handler to send. Edit=true means edit
// the existing message instead of sending a new one.
type Reply struct {
	Text     string
	Keyboard [][]Button
	Edit     bool
}
```

- [ ] **Step 2: Create the Redis sessions (mirror botreg).** Create `backend/internal/platform/meetingedit/redis_sessions.go`:

```go
package meetingedit

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
	return "meetedit:" + strconv.FormatInt(telegramID, 10)
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

- [ ] **Step 3: Write the failing Service test.** Create `backend/internal/platform/meetingedit/service_test.go`:

```go
package meetingedit

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type fakeBackend struct {
	meetings []postgres.MeetingWithTZ
	gotIn    application.UpdateMeetingInput
	gotWS    uuid.UUID
	gotUser  uuid.UUID
	gotMID   uuid.UUID
	applied  postgres.Meeting
}

func (f *fakeBackend) ListEditableMeetings(_ context.Context, _ int64) ([]postgres.MeetingWithTZ, error) {
	return f.meetings, nil
}
func (f *fakeBackend) UpdateMeeting(_ context.Context, ws, user, mid uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error) {
	f.gotWS, f.gotUser, f.gotMID, f.gotIn = ws, user, mid, in
	return f.applied, nil
}

type memSessions struct{ m map[int64]*State }

func newMemSessions() *memSessions { return &memSessions{m: map[int64]*State{}} }
func (s *memSessions) Get(_ context.Context, tg int64) (*State, error) { return s.m[tg], nil }
func (s *memSessions) Set(_ context.Context, tg int64, st State) error { c := st; s.m[tg] = &c; return nil }
func (s *memSessions) Del(_ context.Context, tg int64) error          { delete(s.m, tg); return nil }

func sampleMeeting() postgres.MeetingWithTZ {
	loc, _ := time.LoadLocation("Asia/Almaty")
	org := uuid.New()
	return postgres.MeetingWithTZ{
		Meeting: postgres.Meeting{
			ID: uuid.New(), WorkspaceID: uuid.New(), OrganizerUserID: &org,
			Dept: "Разработка", Type: "Планёрка", Host: "Иванов",
			StartsAt:   time.Date(2026, 6, 1, 14, 0, 0, 0, loc).UTC(),
			EndsAt:     time.Date(2026, 6, 1, 15, 0, 0, 0, loc).UTC(),
			Recurrence: "once", Description: "d", Name: "Разработка | Планёрка | Иванов | 2026-06-01",
			MeetLink: "https://meet.google.com/x",
		},
		TZ: "Asia/Almaty",
	}
}

func TestEditFlow_TextField(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(42)

	// /edit -> list with a pick button
	start := svc.Start(ctx, tg)
	if len(start.Keyboard) != 1 || start.Keyboard[0][0].Data != "medit:pick:"+m.ID.String() {
		t.Fatalf("bad start keyboard: %+v", start.Keyboard)
	}
	// pick -> menu
	if r, ok := svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String()); !ok || !strings.Contains(r.Text, "Редактирование") {
		t.Fatalf("pick reply: %+v ok=%v", r, ok)
	}
	// choose dept -> awaiting
	if _, ok := svc.OnCallback(ctx, tg, "medit:field:dept"); !ok {
		t.Fatal("field:dept not handled")
	}
	// type new value -> back to menu
	if r, ok := svc.OnText(ctx, tg, "Маркетинг"); !ok || !strings.Contains(r.Text, "★") {
		t.Fatalf("ontext reply: %+v ok=%v", r, ok)
	}
	// apply -> backend called with Dept override
	if _, ok := svc.OnCallback(ctx, tg, "medit:apply"); !ok {
		t.Fatal("apply not handled")
	}
	if be.gotIn.Dept == nil || *be.gotIn.Dept != "Маркетинг" {
		t.Fatalf("apply did not pass dept override: %+v", be.gotIn)
	}
	if be.gotWS != m.WorkspaceID || be.gotUser != *m.OrganizerUserID || be.gotMID != m.ID {
		t.Fatal("apply passed wrong ids")
	}
}

func TestEditFlow_NoChanges(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(7)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	r, ok := svc.OnCallback(ctx, tg, "medit:apply")
	if !ok || !strings.Contains(r.Text, "Нет изменений") {
		t.Fatalf("expected no-changes reply, got %+v", r)
	}
}

func TestEditFlow_DateTime(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(9)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:field:datetime")
	if r, ok := svc.OnText(ctx, tg, "bad"); !ok || !strings.Contains(r.Text, "формат") {
		t.Fatalf("expected format error, got %+v", r)
	}
	svc.OnText(ctx, tg, "2026-06-02 10:00-11:00")
	svc.OnCallback(ctx, tg, "medit:apply")
	if be.gotIn.Date == nil || *be.gotIn.Date != "2026-06-02" || be.gotIn.Start == nil || *be.gotIn.Start != "10:00" {
		t.Fatalf("datetime override not passed: %+v", be.gotIn)
	}
}
```

- [ ] **Step 4: Run it, verify it fails.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -run TestEditFlow -v`
      Expected: FAIL — `undefined: New` / `Service`.

- [ ] **Step 5: Implement the Service.** Create `backend/internal/platform/meetingedit/service.go`:

```go
package meetingedit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/domain/meeting"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Backend is the application surface the FSM needs (satisfied by *application.Services).
type Backend interface {
	ListEditableMeetings(ctx context.Context, telegramID int64) ([]postgres.MeetingWithTZ, error)
	UpdateMeeting(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error)
}

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	backend  Backend
	sessions sessions
}

func New(backend Backend, sess sessions) *Service {
	return &Service{backend: backend, sessions: sess}
}

// Start handles /edit: lists the user's upcoming meetings as a pick keyboard.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	ms, err := s.backend.ListEditableMeetings(ctx, telegramID)
	if err != nil {
		return Reply{Text: "Не удалось получить встречи, попробуй позже."}
	}
	if len(ms) == 0 {
		return Reply{Text: "Нет предстоящих встреч для редактирования.\n(Убедись, что Telegram привязан в приложении.)"}
	}
	var rows [][]Button
	for _, m := range ms {
		rows = append(rows, []Button{{Text: m.Name, Data: "medit:pick:" + m.ID.String()}})
	}
	return Reply{Text: "Выбери встречу для редактирования:", Keyboard: rows}
}

// OnCallback handles medit:* inline-button taps. The bool is false when data is
// not an medit callback.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	switch {
	case strings.HasPrefix(data, "medit:pick:"):
		return s.pick(ctx, telegramID, strings.TrimPrefix(data, "medit:pick:")), true
	case strings.HasPrefix(data, "medit:field:"):
		return s.field(ctx, telegramID, strings.TrimPrefix(data, "medit:field:")), true
	case strings.HasPrefix(data, "medit:set:rec:"):
		return s.setRec(ctx, telegramID, strings.TrimPrefix(data, "medit:set:rec:")), true
	case data == "medit:apply":
		return s.apply(ctx, telegramID), true
	case data == "medit:cancel":
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: "Редактирование отменено.", Edit: true}, true
	}
	return Reply{}, false
}

// OnText feeds a free-text value into an active "awaiting field" session. The
// bool is false when there is no awaiting session (so other handlers can run).
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.Step != stepAwaiting {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	if st.AwaitingField == "datetime" {
		d, start, end, perr := parseDateTime(text)
		if perr != nil {
			return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
		}
		st.Overrides["date"] = d
		st.Overrides["start"] = start
		st.Overrides["end"] = end
	} else {
		if text == "" {
			return Reply{Text: "Пусто. Введи значение:"}, true
		}
		st.Overrides[st.AwaitingField] = text
	}
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, false), true
}

func (s *Service) pick(ctx context.Context, telegramID int64, idStr string) Reply {
	mid, err := uuid.Parse(idStr)
	if err != nil {
		return Reply{Text: "Неизвестная встреча."}
	}
	ms, err := s.backend.ListEditableMeetings(ctx, telegramID)
	if err != nil {
		return Reply{Text: "Не удалось получить встречу."}
	}
	var found *postgres.MeetingWithTZ
	for i := range ms {
		if ms[i].ID == mid {
			found = &ms[i]
			break
		}
	}
	if found == nil || found.OrganizerUserID == nil {
		return Reply{Text: "Эта встреча недоступна для редактирования."}
	}
	loc := loadLoc(found.TZ)
	st := State{
		Step:        stepMenu,
		MeetingID:   mid.String(),
		WorkspaceID: found.WorkspaceID.String(),
		UserID:      found.OrganizerUserID.String(),
		Cur:         snapshot(found.Meeting, loc),
		Overrides:   map[string]string{},
	}
	_ = s.sessions.Set(ctx, telegramID, st)
	return menuReply(st, true)
}

func (s *Service) field(ctx context.Context, telegramID int64, f string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	if f == "rec" {
		return recReply()
	}
	prompt, ok := fieldPrompts[f]
	if !ok {
		return Reply{}
	}
	st.Step = stepAwaiting
	st.AwaitingField = f
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: prompt}
}

func (s *Service) setRec(ctx context.Context, telegramID int64, val string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	if !meeting.Recurrence(val).Valid() {
		return Reply{}
	}
	st.Overrides["recurrence"] = val
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, true)
}

func (s *Service) apply(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	if len(st.Overrides) == 0 {
		return Reply{Text: "Нет изменений. Выбери поле или нажми «Отмена».", Keyboard: menuKeyboard(), Edit: true}
	}
	ws, _ := uuid.Parse(st.WorkspaceID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	m, err := s.backend.UpdateMeeting(ctx, ws, uid, mid, toInput(st.Overrides))
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			return Reply{Text: "Неверные данные. Поправь поле и попробуй снова."}
		case errors.Is(err, application.ErrForbidden):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: "Нет доступа к этой встрече."}
		default:
			return Reply{Text: "Не удалось обновить встречу, попробуй позже."}
		}
	}
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: "Готово ✏️\n" + summary(m)}
}

var fieldPrompts = map[string]string{
	"dept":        "Введи новый отдел:",
	"type":        "Введи новый тип встречи:",
	"host":        "Введи нового ведущего:",
	"description": "Введи новое описание:",
	"datetime":    "Введи дату и время: ГГГГ-ММ-ДД ЧЧ:ММ–ЧЧ:ММ\n(например: 2026-06-01 14:00–15:00)",
}

func menuKeyboard() [][]Button {
	return [][]Button{
		{{Text: "📅 Дата/время", Data: "medit:field:datetime"}},
		{{Text: "🏢 Отдел", Data: "medit:field:dept"}, {Text: "🏷 Тип", Data: "medit:field:type"}},
		{{Text: "🎤 Ведущий", Data: "medit:field:host"}, {Text: "📝 Описание", Data: "medit:field:description"}},
		{{Text: "🔁 Частота", Data: "medit:field:rec"}},
		{{Text: "✅ Применить", Data: "medit:apply"}, {Text: "✖ Отмена", Data: "medit:cancel"}},
	}
}

func recReply() Reply {
	return Reply{Text: "Выбери частоту:", Edit: true, Keyboard: [][]Button{
		{{Text: "Однократно", Data: "medit:set:rec:once"}},
		{{Text: "Ежедневно", Data: "medit:set:rec:daily"}},
		{{Text: "Еженедельно", Data: "medit:set:rec:weekly"}},
		{{Text: "Раз в 2 недели", Data: "medit:set:rec:biweekly"}},
		{{Text: "Ежемесячно", Data: "medit:set:rec:monthly"}},
	}}
}

func menuReply(st State, edit bool) Reply {
	return Reply{Text: menuText(st), Keyboard: menuKeyboard(), Edit: edit}
}

func menuText(st State) string {
	eff := func(k string) string {
		if v, ok := st.Overrides[k]; ok {
			return v
		}
		return st.Cur[k]
	}
	mark := func(k string) string {
		if _, ok := st.Overrides[k]; ok {
			return " ★"
		}
		return ""
	}
	var b strings.Builder
	b.WriteString("Редактирование встречи (★ — изменено):\n")
	dmark := ""
	if _, ok := st.Overrides["date"]; ok {
		dmark = " ★"
	}
	fmt.Fprintf(&b, "• Дата/время: %s %s–%s%s\n", eff("date"), eff("start"), eff("end"), dmark)
	fmt.Fprintf(&b, "• Отдел: %s%s\n", eff("dept"), mark("dept"))
	fmt.Fprintf(&b, "• Тип: %s%s\n", eff("type"), mark("type"))
	fmt.Fprintf(&b, "• Ведущий: %s%s\n", eff("host"), mark("host"))
	fmt.Fprintf(&b, "• Описание: %s%s\n", eff("description"), mark("description"))
	fmt.Fprintf(&b, "• Частота: %s%s\n", recLabel(eff("recurrence")), mark("recurrence"))
	return b.String()
}

func recLabel(v string) string {
	if v == "once" || v == "" {
		return "Однократно"
	}
	return meeting.Recurrence(v).Label()
}

func snapshot(m postgres.Meeting, loc *time.Location) map[string]string {
	s := m.StartsAt.In(loc)
	e := m.EndsAt.In(loc)
	return map[string]string{
		"dept": m.Dept, "type": m.Type, "host": m.Host,
		"description": m.Description, "recurrence": m.Recurrence,
		"date": s.Format("2006-01-02"), "start": s.Format("15:04"), "end": e.Format("15:04"),
	}
}

func toInput(ov map[string]string) application.UpdateMeetingInput {
	var in application.UpdateMeetingInput
	set := func(p **string, k string) {
		if v, ok := ov[k]; ok {
			vv := v
			*p = &vv
		}
	}
	set(&in.Dept, "dept")
	set(&in.Type, "type")
	set(&in.Host, "host")
	set(&in.Description, "description")
	set(&in.Recurrence, "recurrence")
	set(&in.Date, "date")
	set(&in.Start, "start")
	set(&in.End, "end")
	return in
}

func summary(m postgres.Meeting) string {
	s := "«" + m.Name + "»"
	if m.MeetLink != "" {
		s += "\n🔗 " + m.MeetLink
	}
	return s
}

func loadLoc(tz string) *time.Location {
	if tz == "" {
		tz = "Asia/Almaty"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}
```

- [ ] **Step 6: Run tests, verify they pass.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -v`
      Expected: `TestParseDateTime*`, `TestEditFlow_TextField`, `TestEditFlow_NoChanges`, `TestEditFlow_DateTime` PASS.

- [ ] **Step 7: Build + commit.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...` then:

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meetingedit/ && git commit -m "feat(meetings): meetingedit FSM service

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 9: wiring — MultiHandler routing + centralized Services + asynq handler

**Files:**

- Modify: `backend/internal/infrastructure/telegram/multitenant.go`
- Modify: `backend/internal/delivery/http/app.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add the editor to MultiHandler.** In `backend/internal/infrastructure/telegram/multitenant.go`:

(a) Add the import `"github.com/Jaryq-Lab/notify-bot/internal/platform/meetingedit"`. (The editor backend is passed as the `meetingedit.Backend` interface type, so this file needs neither `application` nor `uuid`.)

(b) Add a field to the struct:

```go
type MultiHandler struct {
	store     *postgres.Store
	executor  *scenario_executor.Executor
	registrar *botreg.Service
	settings  *botsettings.Service
	editor    *meetingedit.Service
	log       *zap.Logger
}
```

(c) Change `NewMultiHandler` to accept the editor backend and build the editor (the `*application.Services` passed in satisfies `meetingedit.Backend`):

```go
func NewMultiHandler(store *postgres.Store, cipher *crypto.TokenCipher, b *bot.Bot, rdb *redis.Client, adminIDs []int64, otpLog bool, editorBackend meetingedit.Backend, log *zap.Logger) *MultiHandler {
	otp := platformauth.NewOTP(rdb, log, otpLog)
	registrar := botreg.New(store, otp, botreg.NewRedisSessions(rdb), adminIDs)
	settings := botsettings.New(store)
	editor := meetingedit.New(editorBackend, meetingedit.NewRedisSessions(rdb))
	return &MultiHandler{
		store:     store,
		executor:  scenario_executor.New(store, cipher, b, log),
		registrar: registrar,
		settings:  settings,
		editor:    editor,
		log:       log,
	}
}
```

(d) Route free text to the editor after the registrar declines. Replace the no-command branch in `Handle`:

```go
	cmd, ok := parseCommand(text)
	if !ok {
		if isPrivate {
			if reply, handled := h.registrar.OnText(ctx, from.ID, text); handled {
				h.reply(ctx, b, update.Message, reply)
				return
			}
			if reply, handled := h.editor.OnText(ctx, from.ID, text); handled {
				h.sendEditorReply(ctx, b, chatID, 0, reply)
			}
		}
		return
	}
```

(e) Add an `/edit` command case in the `switch cmd` block:

```go
	case "/edit":
		if isPrivate {
			if _, err := h.store.GetBotUserByTelegramID(ctx, from.ID); err != nil {
				h.reply(ctx, b, update.Message, "Сначала зарегистрируйся: /start")
				return
			}
			h.sendEditorReply(ctx, b, chatID, 0, h.editor.Start(ctx, from.ID))
		}
```

(f) Add an `medit:` branch in `handleCallback` (before the final `AnswerCallbackQuery`):

```go
	if strings.HasPrefix(cq.Data, "medit:") {
		if reply, handled := h.editor.OnCallback(ctx, cq.From.ID, cq.Data); handled && cq.Message.Message != nil {
			h.sendEditorReply(ctx, b, cq.Message.Message.Chat.ID, cq.Message.Message.ID, reply)
		}
	}
```

(g) Add the send helper + the markup converter (next to `toInlineMarkup`):

```go
func (h *MultiHandler) sendEditorReply(ctx context.Context, b *bot.Bot, chatID int64, msgID int, reply meetingedit.Reply) {
	var markup models.ReplyMarkup
	if len(reply.Keyboard) > 0 {
		markup = toMeditMarkup(reply.Keyboard)
	}
	if reply.Edit && msgID != 0 {
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID: chatID, MessageID: msgID, Text: reply.Text, ReplyMarkup: markup,
		})
		return
	}
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: reply.Text, ReplyMarkup: markup})
}

func toMeditMarkup(rows [][]meetingedit.Button) models.InlineKeyboardMarkup {
	var kb [][]models.InlineKeyboardButton
	for _, row := range rows {
		var r []models.InlineKeyboardButton
		for _, btn := range row {
			r = append(r, models.InlineKeyboardButton{Text: btn.Text, CallbackData: btn.Data})
		}
		kb = append(kb, r)
	}
	return models.InlineKeyboardMarkup{InlineKeyboard: kb}
}
```

Note: this file only needs the new `meetingedit` import (plus its existing imports). Remove any import the build reports as unused.

- [ ] **Step 2: Centralize `Services` construction in `NewApp`.** In `backend/internal/delivery/http/app.go`, change the signature to accept a prebuilt `*application.Services` and use it, dropping the internal calendar-provider + Services build:

Change the signature:

```go
func NewApp(cfg config.Config, store *postgres.Store, cipher *crypto.TokenCipher, rdb *redis.Client, tg *bot.Bot, log *zap.Logger, services *application.Services) (*fiber.App, error) {
```

Remove the `calProvider` block (the `var calProvider … if cfg.CalendarStub … else …`) and change the `api` construction to use the passed-in services:

```go
	api := &handlers.API{
		App:     services,
		Bot:     tg,
		RDB:     rdb,
		Log:     log,
		TMA:     telegram.NewInitDataValidator(cfg.BotToken),
		Version: os.Getenv("APP_VERSION"),
	}
```

Remove now-unused imports from `app.go`: the `queue` parameter is gone, and `calendarstub` / `calendargoogle` are no longer referenced here. Remove the `calendarstub`/`calendargoogle` imports and the `queue` param. Keep `application` (used for the `*application.Services` param type). Run the build to find any other now-unused import and remove it.

- [ ] **Step 3: Build the provider + Services once in `main.go` and wire everything.** In `backend/cmd/server/main.go`:

(a) Add imports: `"github.com/Jaryq-Lab/notify-bot/internal/application"`, `calendargoogle "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/calendar/google"`, `calendarstub "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/calendar/stub"`.

(b) After `store`, `cipher`, `queueClient` are created (and before the bot/tgHandler block), build the calendar provider and the shared `Services`:

```go
	var calProvider application.CalendarProvider
	if cfg.CalendarStub {
		calProvider = calendarstub.NewProvider()
	} else {
		calProvider = calendargoogle.NewProvider(store, cipher)
	}
	services := &application.Services{Store: store, Cipher: cipher, Queue: queueClient, Calendar: calProvider, Log: logger}
```

(c) Pass `services` to `NewMultiHandler` (the new `editorBackend` param, before `logger`):

```go
		tgHandler = telegram.NewMultiHandler(store, cipher, tg, rdb, cfg.BotAdminTelegramIDs, cfg.AuthOTPLog, services, logger)
```

(d) Add the `meeting:updated` worker handler next to `meetingCreatedHandler`:

```go
	meetingUpdatedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseMeetingUpdated(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.WorkspaceID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleUpdated(c, wid, mid)
	}
```

and register it in the `NewServer` map:

```go
	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, map[string]asynq.HandlerFunc{
		asynqqueue.TaskRunScenario:    asynqHandler,
		asynqqueue.TaskMeetingCreated: meetingCreatedHandler,
		asynqqueue.TaskMeetingUpdated: meetingUpdatedHandler,
	})
```

(e) Update the `NewApp` call to the new signature (drop `queueClient`, append `services`):

```go
	app, err := deliveryhttp.NewApp(cfg, store, cipher, rdb, tg, logger, services)
```

- [ ] **Step 4: Build + vet.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./...`
      Expected: clean. Fix any unused-import errors the compiler reports (see notes in Steps 1–2).

- [ ] **Step 5: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/telegram/multitenant.go backend/internal/delivery/http/app.go backend/cmd/server/main.go && git commit -m "feat(meetings): wire meeting-edit FSM + meeting:updated handler

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 10: full verification + docs

**Files:**

- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Run the full suite.** Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && make test && make lint && make build`
      Expected: all green. (Fallback if `make` is unavailable: `cd backend && env -u GOROOT go test ./... && env -u GOROOT go vet ./... && env -u GOROOT go build ./...`.) If anything fails, STOP and report it.

- [ ] **Step 2: Document the feature.** In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list, after the "Meeting-created notification (§5a, done)" line, add:

```markdown
> **Meeting field editing (§4.4.1, done):** a `/edit` bot FSM (Redis session) lets an organizer edit their upcoming meeting's fields (date/time, dept, type, host, description, recurrence); on apply the name is recomputed, the Google event is patched (`SendUpdates=all` emails attendees), the row is persisted, and a `meeting:updated` asynq job DMs participants + organizer. Meetings are resolved via `platform_users.telegram_id`. Participant management (§4.3), recurring-series edit (§4.4.2), bot admin-edit of others' meetings, and conflict warnings (§4.7) remain planned.
```

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document §4.4.1 meeting field editing

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** identity/scope (Tasks 2,6,9) · layering: application command + FSM package (Tasks 5,6,8) · FSM states/UX/input (Tasks 7,8) · `UpdateEvent` Patch + repo + command (Tasks 1,2,6) · `meeting:updated` notification (Tasks 3,4,9) · testing (Tasks 1,4,5,7,8,10) · docs (Task 10). Out-of-scope items (participants, series, admin-edit, conflicts) are recorded in the spec and Task 10 doc note. All covered.
- **Type consistency:** `UpdateMeetingInput{Dept,Type,Host,Date,Start,End,Recurrence,Description *string}` defined in Task 5, used identically in Tasks 6, 8. `applyMeetingUpdate(cur, in, loc)` (Task 5) called by `UpdateMeeting` (Task 6). `MeetingWithTZ{Meeting; TZ}` (Task 2) returned by `ListMeetingsByOrganizerTelegram`/`ListEditableMeetings` (Tasks 2,6) and consumed by `meetingedit` (Task 8). `meetingedit.Backend{ListEditableMeetings, UpdateMeeting}` (Task 8) satisfied by `*application.Services` (Task 6) and injected via `NewMultiHandler` (Task 9). `UpdateEvent(ctx, eventID, CalendarEvent)` (Task 1) called by `UpdateMeeting` (Task 6). `EnqueueMeetingUpdated`/`ParseMeetingUpdated`/`TaskMeetingUpdated` (Task 3) used in Tasks 6, 9. `Reply`/`Button` (Task 8) consumed by `toMeditMarkup`/`sendEditorReply` (Task 9).
- **No placeholders:** every code/command step is concrete. The only conditional instructions are the "remove unused imports if the compiler reports them" notes in Task 9, which are explicit and build-checked.

```

```
