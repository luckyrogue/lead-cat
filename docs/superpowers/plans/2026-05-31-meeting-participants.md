# Increment B — bot-FSM participant management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** From the `/edit` bot FSM, let an organizer add/remove meeting guests (by email or employee-directory search), syncing the Google attendee list (`SendUpdates=all`) and DMing the affected person via asynq.

**Architecture:** Extends the `meetingedit` FSM with a "Participants" sub-screen (immediate ops, separate from field accumulate-then-apply). New `application` commands `AddParticipant`/`RemoveParticipant` (mirror `UpdateMeeting`: ACL → DB write → Google attendee sync → enqueue notify) and a `SearchEmployees` query. New `participant:added`/`participant:removed` asynq tasks handled by `meeting_notifier`.

**Tech Stack:** Go, go-telegram/bot, asynq, pgx, zap, google/uuid, google.golang.org/api/calendar/v3, net/mail, redis.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-participants-design.md`

**Conventions:** Run Go from `backend/` with `env -u GOROOT`. Module `github.com/Jaryq-Lab/notify-bot`. Build check: `env -u GOROOT go build ./...`.

---

## Task 1: `CalendarService.UpdateAttendees` (port + stub + Google adapter)

**Files:**
- Modify: `backend/internal/application/calendar.go`
- Modify: `backend/internal/infrastructure/calendar/stub/stub.go`
- Modify: `backend/internal/infrastructure/calendar/google/adapter.go`
- Test: `backend/internal/infrastructure/calendar/google/adapter_test.go`

- [ ] **Step 1: Add the method to the port.** In `backend/internal/application/calendar.go`, extend `CalendarService`:

```go
type CalendarService interface {
	CreateEvent(ctx context.Context, e CalendarEvent) (CalendarResult, error)
	UpdateEvent(ctx context.Context, eventID string, e CalendarEvent) error
	UpdateAttendees(ctx context.Context, eventID string, emails []string) error
	DeleteEvent(ctx context.Context, eventID string) error
}
```

- [ ] **Step 2: Stub.** In `backend/internal/infrastructure/calendar/stub/stub.go`, add (after `UpdateEvent`):

```go
func (s *Service) UpdateAttendees(_ context.Context, _ string, _ []string) error { return nil }
```

- [ ] **Step 3: Failing test.** In `backend/internal/infrastructure/calendar/google/adapter_test.go`, add:

```go
func TestAttendeeList(t *testing.T) {
	got := attendeeList([]string{"a@x", "b@y"})
	if len(got) != 2 || got[0].Email != "a@x" || got[1].Email != "b@y" {
		t.Fatalf("bad attendees: %+v", got)
	}
	if attendeeList(nil) != nil {
		t.Fatal("empty input must yield nil slice")
	}
}
```

- [ ] **Step 4: Run, verify fail.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/infrastructure/calendar/google/ -run TestAttendeeList -v` → FAIL (undefined attendeeList).

- [ ] **Step 5: Implement.** In `backend/internal/infrastructure/calendar/google/adapter.go`, add (after `UpdateEvent`):

```go
func attendeeList(emails []string) []*calendar.EventAttendee {
	var as []*calendar.EventAttendee
	for _, e := range emails {
		as = append(as, &calendar.EventAttendee{Email: e})
	}
	return as
}

// UpdateAttendees replaces the event's guest list with emails. SendUpdates("all")
// emails invites to newly-added guests and cancellations to removed ones.
// ForceSendFields ensures an empty list actually clears the attendees.
func (a *adapter) UpdateAttendees(ctx context.Context, eventID string, emails []string) error {
	ev := &calendar.Event{Attendees: attendeeList(emails), ForceSendFields: []string{"Attendees"}}
	_, err := a.svc.Events.Patch(a.calendarID, eventID, ev).SendUpdates("all").Context(ctx).Do()
	return err
}
```

- [ ] **Step 6: Run tests + build.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/infrastructure/calendar/... -v && env -u GOROOT go build ./...` → PASS, build OK.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/calendar.go backend/internal/infrastructure/calendar/ && git commit -m "feat(meetings): CalendarService.UpdateAttendees (Google attendee sync)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: repo — `RemoveParticipant`

**Files:**
- Modify: `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`

- [ ] **Step 1: Add the method** (after `ListParticipants`):

```go
// RemoveParticipant deletes a participant by email from a meeting.
func (s *Store) RemoveParticipant(ctx context.Context, meetingID uuid.UUID, email string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM meeting_participants WHERE meeting_id = $1 AND email = $2`, meetingID, email)
	return err
}
```

- [ ] **Step 2: Build + vet.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/` → clean. (No DB harness — build/vet is the gate, per repo convention.)

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go && git commit -m "feat(meetings): repo RemoveParticipant

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: queue — `participant:added` / `participant:removed` tasks

**Files:**
- Modify: `backend/internal/infrastructure/queue/asynq/queue.go`

- [ ] **Step 1: Add tasks, shared payload, enqueues, parser** (after the `meeting:updated` block):

```go
const (
	TaskParticipantAdded   = "participant:added"
	TaskParticipantRemoved = "participant:removed"
)

type ParticipantPayload struct {
	WorkspaceID string `json:"workspace_id"`
	MeetingID   string `json:"meeting_id"`
	Email       string `json:"email"`
}

func (c *Client) EnqueueParticipantAdded(ctx context.Context, workspaceID, meetingID uuid.UUID, email string) error {
	return c.enqueueParticipant(ctx, TaskParticipantAdded, workspaceID, meetingID, email)
}

func (c *Client) EnqueueParticipantRemoved(ctx context.Context, workspaceID, meetingID uuid.UUID, email string) error {
	return c.enqueueParticipant(ctx, TaskParticipantRemoved, workspaceID, meetingID, email)
}

func (c *Client) enqueueParticipant(ctx context.Context, taskType string, workspaceID, meetingID uuid.UUID, email string) error {
	p, _ := json.Marshal(ParticipantPayload{
		WorkspaceID: workspaceID.String(),
		MeetingID:   meetingID.String(),
		Email:       email,
	})
	task := asynq.NewTask(taskType, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseParticipant(t *asynq.Task) (ParticipantPayload, error) {
	var p ParticipantPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
```

- [ ] **Step 2: Build.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...` → OK.

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/queue/asynq/queue.go && git commit -m "feat(queue): participant added/removed tasks

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: notifier — `buildRemovedMessage` + `HandleParticipantAdded`/`HandleParticipantRemoved`

**Files:**
- Modify: `backend/internal/platform/meeting_notifier/message.go`
- Modify: `backend/internal/platform/meeting_notifier/message_test.go`
- Modify: `backend/internal/platform/meeting_notifier/notifier.go`

- [ ] **Step 1: Failing test.** In `backend/internal/platform/meeting_notifier/message_test.go`, add:

```go
func TestBuildRemovedMessage(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	m := buildRemovedMessage("Разработка | Планёрка", start, loc)
	for _, want := range []string{"удалили", "Разработка | Планёрка", "31.05.2026", "UTC+5"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}
	if strings.Contains(m, "🔗") {
		t.Fatal("removed message has no meet link")
	}
}
```

- [ ] **Step 2: Run, verify fail.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -run TestBuildRemovedMessage -v` → FAIL (undefined buildRemovedMessage).

- [ ] **Step 3: Implement the message.** In `backend/internal/platform/meeting_notifier/message.go`, add (after `buildUpdatedMessage`):

```go
func buildRemovedMessage(name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("➖ Вас удалили из встречи\n«%s»\n🗓 %s (%s)", name, s.Format("02.01.2006"), tzLabel(s))
}
```

- [ ] **Step 4: Run, verify pass.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v` → all message tests PASS.

- [ ] **Step 5: Add the handlers.** In `backend/internal/platform/meeting_notifier/notifier.go`, add (after `HandleUpdated`):

```go
// HandleParticipantAdded DMs a newly-added participant (if they have a bot account).
func (n *Notifier) HandleParticipantAdded(ctx context.Context, workspaceID, meetingID uuid.UUID, email string) error {
	return n.notifyParticipant(ctx, workspaceID, meetingID, email, true)
}

// HandleParticipantRemoved DMs a removed participant (if they have a bot account).
func (n *Notifier) HandleParticipantRemoved(ctx context.Context, workspaceID, meetingID uuid.UUID, email string) error {
	return n.notifyParticipant(ctx, workspaceID, meetingID, email, false)
}

func (n *Notifier) notifyParticipant(ctx context.Context, workspaceID, meetingID uuid.UUID, email string, added bool) error {
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
	u, err := n.store.GetBotUserByEmail(ctx, email)
	if err != nil {
		return nil // not a bot user — the Google email invitation/cancellation covers them
	}
	var text string
	if added {
		text = buildEventMessage("➕ Вас добавили на встречу", m.Name, m.MeetLink, m.StartsAt, m.EndsAt, loc)
	} else {
		text = buildRemovedMessage(m.Name, m.StartsAt, loc)
	}
	if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: u.TelegramID, Text: text}); err != nil {
		n.log.Warn("send participant notice",
			zap.Int64("telegram_id", u.TelegramID),
			zap.String("meeting_id", m.ID.String()),
			zap.Bool("added", added),
			zap.Error(err))
	}
	return nil
}
```

(`cmp`, `fmt`, `time`, bot/uuid/zap, `postgres` are already imported in `notifier.go`. `GetBotUserByEmail` exists on the store.)

- [ ] **Step 6: Build + test.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v` → build OK; tests PASS.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meeting_notifier/ && git commit -m "feat(meetings): participant added/removed DM notifications

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: application — commands, search, ACL helper, email normalize

**Files:**
- Create: `backend/internal/application/participants.go`
- Create: `backend/internal/application/participants_test.go`
- Modify: `backend/internal/application/meeting_service.go` (use the new `ownerOrOrganizer` helper)

- [ ] **Step 1: Failing test for the pure helpers.** Create `backend/internal/application/participants_test.go`:

```go
package application

import (
	"testing"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

func TestNormalizeEmail(t *testing.T) {
	got, err := normalizeEmail("  Ivan@Corp.KZ ")
	if err != nil || got != "ivan@corp.kz" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := normalizeEmail("not-an-email"); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestFilterEmployees(t *testing.T) {
	all := []postgres.Employee{
		{FullName: "Иван Иванов", Email: "ivan@corp.kz"},
		{FullName: "Пётр Петров", Email: "petr@corp.kz"},
		{FullName: "Анна Сидорова", Email: "anna@corp.kz"},
	}
	// by name substring (case-insensitive)
	if got := filterEmployees(all, "иван"); len(got) != 1 || got[0].Email != "ivan@corp.kz" {
		t.Fatalf("name search: %+v", got)
	}
	// by email substring
	if got := filterEmployees(all, "PETR@"); len(got) != 1 || got[0].FullName != "Пётр Петров" {
		t.Fatalf("email search: %+v", got)
	}
	// empty query → nil
	if got := filterEmployees(all, "   "); got != nil {
		t.Fatalf("empty query must yield nil, got %+v", got)
	}
	// no match → empty
	if got := filterEmployees(all, "zzz"); len(got) != 0 {
		t.Fatalf("no match must yield none, got %+v", got)
	}
}
```

- [ ] **Step 2: Run, verify fail.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run 'TestNormalizeEmail|TestFilterEmployees' -v` → FAIL (undefined normalizeEmail/filterEmployees).

- [ ] **Step 3: Implement commands + helpers.** Create `backend/internal/application/participants.go`:

```go
package application

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// ownerOrOrganizer reports whether userID is the workspace owner or the meeting's organizer.
func ownerOrOrganizer(w postgres.Workspace, organizerUserID *uuid.UUID, userID uuid.UUID) bool {
	if w.OwnerUserID != nil && *w.OwnerUserID == userID {
		return true
	}
	return organizerUserID != nil && *organizerUserID == userID
}

func normalizeEmail(s string) (string, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(s))
	if err != nil {
		return "", err
	}
	return strings.ToLower(addr.Address), nil
}

func filterEmployees(all []postgres.Employee, query string) []postgres.Employee {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []postgres.Employee
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.FullName), q) || strings.Contains(strings.ToLower(e.Email), q) {
			out = append(out, e)
		}
	}
	return out
}

// SearchEmployees returns directory entries whose name or email contains query.
func (s *Services) SearchEmployees(ctx context.Context, workspaceID uuid.UUID, query string) ([]postgres.Employee, error) {
	all, err := s.Store.ListEmployees(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return filterEmployees(all, query), nil
}

// ListParticipants returns a meeting's participants (for the bot FSM).
func (s *Services) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return s.Store.ListParticipants(ctx, meetingID)
}

// AddParticipant adds a guest by email (organizer or owner only): persists, syncs
// the Google attendee list, and enqueues a notification.
func (s *Services) AddParticipant(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := s.loadForParticipantOp(ctx, workspaceID, meetingID, userID)
	if err != nil {
		return err
	}
	email, err = normalizeEmail(email)
	if err != nil {
		return fmt.Errorf("%w: bad email", ErrInvalidInput)
	}
	parts, err := s.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return err
	}
	for _, p := range parts {
		if p.Email == email {
			return fmt.Errorf("%w: already a participant", ErrInvalidInput)
		}
	}
	if err := s.Store.AddParticipants(ctx, meetingID, []postgres.MeetingParticipant{{Email: email}}); err != nil {
		return err
	}
	if err := s.syncAttendees(ctx, workspaceID, m.GoogleEventID, meetingID); err != nil {
		return err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueParticipantAdded(ctx, workspaceID, meetingID, email); err != nil && s.Log != nil {
			s.Log.Warn("enqueue participant added", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

// RemoveParticipant removes a guest by email (organizer or owner only): persists,
// syncs the Google attendee list, and enqueues a notification.
func (s *Services) RemoveParticipant(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, email string) error {
	m, _, err := s.loadForParticipantOp(ctx, workspaceID, meetingID, userID)
	if err != nil {
		return err
	}
	if err := s.Store.RemoveParticipant(ctx, meetingID, email); err != nil {
		return err
	}
	if err := s.syncAttendees(ctx, workspaceID, m.GoogleEventID, meetingID); err != nil {
		return err
	}
	if s.Queue != nil {
		if err := s.Queue.EnqueueParticipantRemoved(ctx, workspaceID, meetingID, email); err != nil && s.Log != nil {
			s.Log.Warn("enqueue participant removed", zap.String("meeting_id", meetingID.String()), zap.Error(err))
		}
	}
	return nil
}

// loadForParticipantOp loads the meeting + workspace and enforces the ACL.
func (s *Services) loadForParticipantOp(ctx context.Context, workspaceID, meetingID, userID uuid.UUID) (postgres.Meeting, postgres.Workspace, error) {
	m, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return postgres.Meeting{}, postgres.Workspace{}, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return postgres.Meeting{}, postgres.Workspace{}, err
	}
	if !ownerOrOrganizer(w, m.OrganizerUserID, userID) {
		return postgres.Meeting{}, postgres.Workspace{}, ErrForbidden
	}
	return m, w, nil
}

// syncAttendees patches the Google event's guest list to the meeting's current
// participants (no-op when the meeting has no Google event).
func (s *Services) syncAttendees(ctx context.Context, workspaceID uuid.UUID, googleEventID string, meetingID uuid.UUID) error {
	if googleEventID == "" {
		return nil
	}
	parts, err := s.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return err
	}
	var emails []string
	for _, p := range parts {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	calSvc, err := s.Calendar.For(ctx, workspaceID)
	if err != nil {
		return err
	}
	if err := calSvc.UpdateAttendees(ctx, googleEventID, emails); err != nil {
		return fmt.Errorf("calendar: %w", err)
	}
	return nil
}
```

Add the `zap` import to this file's import block:

```go
	"go.uber.org/zap"
```

(Both `AddParticipant` and `RemoveParticipant` use `m, _, err := s.loadForParticipantOp(...)` — they need only the meeting's `GoogleEventID`; the workspace is consumed inside the helper for the ACL.)

- [ ] **Step 4: Run, verify pass.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/application/ -run 'TestNormalizeEmail|TestFilterEmployees' -v` → PASS.

- [ ] **Step 5: DRY — use `ownerOrOrganizer` in the existing commands.** In `backend/internal/application/meeting_service.go`, replace the inline ACL in `UpdateMeeting` (currently `isOwner := …; isOrganizer := …; if !isOwner && !isOrganizer { return postgres.Meeting{}, ErrForbidden }`) with:

```go
	if !ownerOrOrganizer(w, cur.OrganizerUserID, userID) {
		return postgres.Meeting{}, ErrForbidden
	}
```

and in `CancelMeeting` (currently `isOwner := …; isOrganizer := …; if !isOwner && !isOrganizer { return ErrForbidden }`) with:

```go
	if !ownerOrOrganizer(w, m.OrganizerUserID, userID) {
		return ErrForbidden
	}
```

- [ ] **Step 6: Build + test.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/application/ -v` → build OK; all application tests PASS.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/ && git commit -m "feat(meetings): Add/RemoveParticipant + SearchEmployees + ownerOrOrganizer

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: `meetingedit` — participants sub-flow

**Files:**
- Modify: `backend/internal/platform/meetingedit/state.go` (add `PartList`, `PartCands`)
- Modify: `backend/internal/platform/meetingedit/service.go` (Backend extension, callbacks, sub-flow, "Участники" button)
- Modify: `backend/internal/platform/meetingedit/service_test.go` (extend fake + add tests)

- [ ] **Step 1: Add session fields.** In `backend/internal/platform/meetingedit/state.go`, add two fields to `State`:

```go
type State struct {
	Step          string            `json:"step"` // menu | awaiting
	MeetingID     string            `json:"meeting_id"`
	WorkspaceID   string            `json:"workspace_id"`
	UserID        string            `json:"user_id"`
	AwaitingField string            `json:"awaiting_field,omitempty"`
	Cur           map[string]string `json:"cur"`       // current display values
	Overrides     map[string]string `json:"overrides"` // pending edits
	PartList      []string          `json:"part_list,omitempty"`  // emails shown in the remove list (index → email)
	PartCands     []string          `json:"part_cands,omitempty"` // emails shown as add candidates (index → email)
}
```

- [ ] **Step 2: Extend the Backend interface.** In `backend/internal/platform/meetingedit/service.go`, extend `Backend`:

```go
type Backend interface {
	ListEditableMeetings(ctx context.Context, telegramID int64) ([]postgres.MeetingWithTZ, error)
	UpdateMeeting(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
	SearchEmployees(ctx context.Context, workspaceID uuid.UUID, query string) ([]postgres.Employee, error)
	AddParticipant(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, email string) error
	RemoveParticipant(ctx context.Context, workspaceID, userID, meetingID uuid.UUID, email string) error
}
```

- [ ] **Step 3: Add the "Участники" button.** In `menuKeyboard`, add a row before the Apply/Cancel row:

```go
func menuKeyboard() [][]Button {
	return [][]Button{
		{{Text: "📅 Дата/время", Data: "medit:field:datetime"}},
		{{Text: "🏢 Отдел", Data: "medit:field:dept"}, {Text: "🏷 Тип", Data: "medit:field:type"}},
		{{Text: "🎤 Ведущий", Data: "medit:field:host"}, {Text: "📝 Описание", Data: "medit:field:description"}},
		{{Text: "🔁 Частота", Data: "medit:field:rec"}},
		{{Text: "👥 Участники", Data: "medit:parts"}},
		{{Text: "✅ Применить", Data: "medit:apply"}, {Text: "✖ Отмена", Data: "medit:cancel"}},
	}
}
```

- [ ] **Step 4: Route the new callbacks.** In `OnCallback`, add cases (order: exact matches and the `premc`/`prem` ordering matters — put `premc:` before `prem:`):

```go
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
	case data == "medit:menu":
		return s.backToMenu(ctx, telegramID), true
	case data == "medit:parts":
		return s.parts(ctx, telegramID), true
	case data == "medit:padd":
		return s.padd(ctx, telegramID), true
	case strings.HasPrefix(data, "medit:padd:"):
		return s.paddPick(ctx, telegramID, strings.TrimPrefix(data, "medit:padd:")), true
	case strings.HasPrefix(data, "medit:premc:"):
		return s.premConfirm(ctx, telegramID, strings.TrimPrefix(data, "medit:premc:")), true
	case strings.HasPrefix(data, "medit:prem:"):
		return s.prem(ctx, telegramID, strings.TrimPrefix(data, "medit:prem:")), true
	}
	return Reply{}, false
}
```

- [ ] **Step 5: Handle the participant search in `OnText`.** In `OnText`, add a branch at the top of the body (after loading `st` and trimming `text`), before the `datetime` branch:

```go
	if st.AwaitingField == "participant" {
		return s.searchParticipant(ctx, telegramID, st, text), true
	}
```

(Insert this immediately after `text = strings.TrimSpace(text)` and before `if st.AwaitingField == "datetime" {`.)

- [ ] **Step 6: Implement the sub-flow.** In `backend/internal/platform/meetingedit/service.go`, add these methods + helpers (place after `apply`):

```go
func (s *Service) backToMenu(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	return menuReply(*st, true)
}

// parts renders the participants sub-menu and records the shown emails by index.
func (s *Service) parts(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	mid, _ := uuid.Parse(st.MeetingID)
	ps, err := s.backend.ListParticipants(ctx, mid)
	if err != nil {
		return Reply{Text: "Не удалось получить участников."}
	}
	var emails []string
	for _, p := range ps {
		if p.Email != "" {
			emails = append(emails, p.Email)
		}
	}
	st.PartList = emails
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return partsReply(emails, true)
}

func partsReply(emails []string, edit bool) Reply {
	var rows [][]Button
	for i, e := range emails {
		rows = append(rows, []Button{{Text: "✖ " + e, Data: fmt.Sprintf("medit:prem:%d", i)}})
	}
	rows = append(rows, []Button{{Text: "➕ Добавить", Data: "medit:padd"}})
	rows = append(rows, []Button{{Text: "⬅ Назад", Data: "medit:menu"}})
	text := "Участники встречи:"
	if len(emails) == 0 {
		text = "Участников пока нет."
	}
	return Reply{Text: text, Keyboard: rows, Edit: edit}
}

func (s *Service) padd(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	st.Step = stepAwaiting
	st.AwaitingField = "participant"
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Введи email участника или часть имени для поиска:"}
}

// searchParticipant looks up directory matches for the typed query and offers
// them (plus the raw email if valid) as add buttons. Stays in the awaiting step
// so re-typing re-searches.
func (s *Service) searchParticipant(ctx context.Context, telegramID int64, st *State, query string) Reply {
	ws, _ := uuid.Parse(st.WorkspaceID)
	emps, err := s.backend.SearchEmployees(ctx, ws, query)
	if err != nil {
		return Reply{Text: "Не удалось выполнить поиск, попробуй ещё раз:"}
	}
	var cands []string
	var rows [][]Button
	seen := map[string]bool{}
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		seen[e.Email] = true
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("medit:padd:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if addr, perr := mail.ParseAddress(strings.TrimSpace(query)); perr == nil {
		email := strings.ToLower(addr.Address)
		if !seen[email] {
			rows = append(rows, []Button{{Text: "➕ Добавить " + email, Data: fmt.Sprintf("medit:padd:%d", len(cands))}})
			cands = append(cands, email)
		}
	}
	if len(cands) == 0 {
		return Reply{Text: "Ничего не найдено. Введи корректный email или часть имени:"}
	}
	st.PartCands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Выбери, кого добавить:", Keyboard: rows}
}

func (s *Service) paddPick(ctx context.Context, telegramID int64, idxStr string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	email, ok := indexInto(st.PartCands, idxStr)
	if !ok {
		return Reply{Text: "Кандидат не найден, начни добавление заново."}
	}
	ws, _ := uuid.Parse(st.WorkspaceID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if err := s.backend.AddParticipant(ctx, ws, uid, mid, email); err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			return Reply{Text: "Уже участник или неверный email."}
		case errors.Is(err, application.ErrForbidden):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: "Нет доступа к этой встрече."}
		default:
			return Reply{Text: "Не удалось добавить участника, попробуй позже."}
		}
	}
	return s.parts(ctx, telegramID)
}

func (s *Service) prem(ctx context.Context, telegramID int64, idxStr string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	email, ok := indexInto(st.PartList, idxStr)
	if !ok {
		return Reply{Text: "Участник не найден, открой список заново."}
	}
	return Reply{
		Text: "Удалить участника " + email + "?",
		Edit: true,
		Keyboard: [][]Button{
			{{Text: "✅ Да", Data: fmt.Sprintf("medit:premc:%s", idxStr)}},
			{{Text: "⬅ Отмена", Data: "medit:parts"}},
		},
	}
}

func (s *Service) premConfirm(ctx context.Context, telegramID int64, idxStr string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	email, ok := indexInto(st.PartList, idxStr)
	if !ok {
		return Reply{Text: "Участник не найден, открой список заново."}
	}
	ws, _ := uuid.Parse(st.WorkspaceID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if err := s.backend.RemoveParticipant(ctx, ws, uid, mid, email); err != nil {
		if errors.Is(err, application.ErrForbidden) {
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: "Нет доступа к этой встрече."}
		}
		return Reply{Text: "Не удалось удалить участника, попробуй позже."}
	}
	return s.parts(ctx, telegramID)
}

// indexInto resolves a string index into a slice, guarding bounds.
func indexInto(list []string, idxStr string) (string, bool) {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(list) {
		return "", false
	}
	return list[i], true
}
```

Add `"strconv"` and `"net/mail"` to `service.go`'s import block (keep all existing imports).

- [ ] **Step 7: Extend the fake backend + add tests.** In `backend/internal/platform/meetingedit/service_test.go`, extend `fakeBackend` with participant methods + recording, and add flow tests:

```go
// add these fields to fakeBackend:
//   participants []postgres.MeetingParticipant
//   employees    []postgres.Employee
//   addErr       error
//   addedEmail   string
//   removedEmail string

func (f *fakeBackend) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.participants, nil
}
func (f *fakeBackend) SearchEmployees(_ context.Context, _ uuid.UUID, query string) ([]postgres.Employee, error) {
	var out []postgres.Employee
	for _, e := range f.employees {
		if strings.Contains(strings.ToLower(e.FullName), strings.ToLower(query)) || strings.Contains(strings.ToLower(e.Email), strings.ToLower(query)) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeBackend) AddParticipant(_ context.Context, _, _, _ uuid.UUID, email string) error {
	if f.addErr != nil {
		return f.addErr
	}
	f.addedEmail = email
	f.participants = append(f.participants, postgres.MeetingParticipant{Email: email})
	return nil
}
func (f *fakeBackend) RemoveParticipant(_ context.Context, _, _, _ uuid.UUID, email string) error {
	f.removedEmail = email
	var kept []postgres.MeetingParticipant
	for _, p := range f.participants {
		if p.Email != email {
			kept = append(kept, p)
		}
	}
	f.participants = kept
	return nil
}

func TestParticipants_AddBySearch(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{
		meetings:  []postgres.MeetingWithTZ{m},
		applied:   m.Meeting,
		employees: []postgres.Employee{{FullName: "Иван Иванов", Email: "ivan@corp.kz"}},
	}
	svc := New(be, newMemSessions())
	const tg = int64(50)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:parts")
	svc.OnCallback(ctx, tg, "medit:padd")
	if r, ok := svc.OnText(ctx, tg, "иван"); !ok || len(r.Keyboard) == 0 {
		t.Fatalf("search reply: %+v ok=%v", r, ok)
	}
	svc.OnCallback(ctx, tg, "medit:padd:0")
	if be.addedEmail != "ivan@corp.kz" {
		t.Fatalf("added email = %q", be.addedEmail)
	}
}

func TestParticipants_AddByRawEmail(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(51)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:padd")
	svc.OnText(ctx, tg, "new@corp.kz")
	svc.OnCallback(ctx, tg, "medit:padd:0")
	if be.addedEmail != "new@corp.kz" {
		t.Fatalf("added email = %q", be.addedEmail)
	}
}

func TestParticipants_AddDuplicate(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting, addErr: application.ErrInvalidInput}
	svc := New(be, newMemSessions())
	const tg = int64(52)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:padd")
	svc.OnText(ctx, tg, "dup@corp.kz")
	if r, _ := svc.OnCallback(ctx, tg, "medit:padd:0"); !strings.Contains(r.Text, "Уже участник") {
		t.Fatalf("duplicate reply: %+v", r)
	}
}

func TestParticipants_Remove(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{
		meetings:     []postgres.MeetingWithTZ{m},
		applied:      m.Meeting,
		participants: []postgres.MeetingParticipant{{Email: "bye@corp.kz"}},
	}
	svc := New(be, newMemSessions())
	const tg = int64(53)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:parts")            // PartList[0] = bye@corp.kz
	if r, _ := svc.OnCallback(ctx, tg, "medit:prem:0"); !strings.Contains(r.Text, "Удалить участника bye@corp.kz") {
		t.Fatalf("confirm reply: %+v", r)
	}
	svc.OnCallback(ctx, tg, "medit:premc:0")
	if be.removedEmail != "bye@corp.kz" {
		t.Fatalf("removed email = %q", be.removedEmail)
	}
}
```

- [ ] **Step 8: Run tests + build.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -v && env -u GOROOT go build ./...` → all PASS (existing + 4 new), build OK.

- [ ] **Step 9: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meetingedit/ && git commit -m "feat(meetings): participants sub-flow in meetingedit FSM

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: wire the participant asynq handlers (main.go)

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Add the two handlers + register them.** In `backend/cmd/server/main.go`, after the existing `meetingUpdatedHandler`, add:

```go
	participantAddedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseParticipant(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.WorkspaceID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleParticipantAdded(c, wid, mid, p.Email)
	}
	participantRemovedHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseParticipant(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.WorkspaceID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleParticipantRemoved(c, wid, mid, p.Email)
	}
```

and add both to the `NewServer` map:

```go
	asynqSrv, err := asynqqueue.NewServer(cfg.RedisURL, logger, map[string]asynq.HandlerFunc{
		asynqqueue.TaskRunScenario:        asynqHandler,
		asynqqueue.TaskMeetingCreated:     meetingCreatedHandler,
		asynqqueue.TaskMeetingUpdated:     meetingUpdatedHandler,
		asynqqueue.TaskParticipantAdded:   participantAddedHandler,
		asynqqueue.TaskParticipantRemoved: participantRemovedHandler,
	})
```

- [ ] **Step 2: Build + vet.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./cmd/server/` → clean.

(No `MultiHandler` change is needed: it already forwards every `medit:*` callback to `editor.OnCallback` and every private free-text to `editor.OnText`; the extended `Backend` is satisfied by `*application.Services`, which already gains the new methods in Task 5 and is already passed to `NewMultiHandler`.)

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/cmd/server/main.go && git commit -m "feat(meetings): register participant added/removed asynq handlers

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 8: full verification + docs

**Files:**
- Modify: `docs/MEETINGS.md`

- [ ] **Step 1: Run the full suite.** From the repo root: `make test && make lint && make build`. (Fallback: `cd backend && env -u GOROOT go test ./... && env -u GOROOT go vet ./... && env -u GOROOT go build ./...`, plus `gofmt -l backend` — if it lists any file, run `gofmt -w` on it.) If anything fails, STOP and report.

- [ ] **Step 2: Document the feature.** In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list, after the "Meeting field editing (§4.4.1, done)" line, add:

```markdown
> **Participant management (§4.3, done):** the `/edit` FSM gains a "👥 Участники" sub-screen — add a guest by email or by searching the employee directory, remove one with confirmation. Each op syncs the Google attendee list (`UpdateAttendees`, `SendUpdates=all` → Google emails the invite/cancellation) and enqueues a `participant:added`/`participant:removed` asynq job that DMs the affected person (if registered). Participant ops apply immediately (separate from field accumulate-then-apply). Employee-directory CSV seeding (§4 directory), recurring-series participants (§4.4.2), and conflict checks (§4.7) remain planned.
```

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document §4.3 participant management

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** entry/flow/layering (Tasks 5,6) · FSM sub-flow incl. index-safe callbacks (Task 6) · `UpdateAttendees` Google sync (Task 1) · repo `RemoveParticipant` (Task 2) · `SearchEmployees`/email normalize/ACL helper (Task 5) · `AddParticipant`/`RemoveParticipant` commands with DB-first ordering (Task 5) · `participant:added/removed` queue + worker + messages (Tasks 3,4) · wiring (Task 7) · testing (Tasks 1,4,5,6,8) · docs (Task 8). Out-of-scope items recorded in spec + Task 8 note. All covered.
- **Type consistency:** `Backend` methods (Task 6) — `ListParticipants`, `SearchEmployees`, `AddParticipant(ctx, ws, userID, meetingID, email)`, `RemoveParticipant(...)` — match `*application.Services` (Task 5). `ParticipantPayload{WorkspaceID,MeetingID,Email}` + `ParseParticipant` + `EnqueueParticipantAdded/Removed` (Task 3) used in Tasks 5,7. `UpdateAttendees(ctx, eventID, emails)` (Task 1) called by `syncAttendees` (Task 5). `buildRemovedMessage`/`buildEventMessage` (Task 4) reused; `HandleParticipantAdded/Removed` (Task 4) called in Task 7. `State.PartList/PartCands` + `indexInto` (Task 6) keep callback_data within 64 bytes. `ownerOrOrganizer` (Task 5) reused in `UpdateMeeting`/`CancelMeeting`.
- **No placeholders:** every code/command step is concrete. The one conditional note (the `_ = w` vs `m, _, err :=` in `AddParticipant`) is explicit and build-checked.
```