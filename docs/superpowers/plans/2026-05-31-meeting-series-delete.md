# §4.5 deletion ("this / whole series") + cancellation notification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete a meeting (or a whole series, this-and-following) from the `/edit` flow, with a confirmation step and a `meeting:cancelled` DM to participants/organizer.

**Architecture:** A new `meeting:cancelled` asynq task + `meeting_notifier.HandleCancelled` deliver the cancel DM. `Services.CancelMeeting` is retrofitted (idempotent early-return + enqueue); `Services.CancelSeries` cancels future occurrences in one atomic UPDATE + best-effort Google deletes. The `meetingedit` FSM gains a "🗑 Удалить" button + confirm, scope-aware (one→CancelMeeting, series→CancelSeries).

**Tech Stack:** Go, go-telegram/bot, asynq, pgx, zap, google/uuid.

**Spec:** `docs/superpowers/specs/2026-05-31-meeting-series-delete-design.md`

**Conventions:** Run Go from `backend/` with `env -u GOROOT`. Module `github.com/luckyrogue/lead-cat`. Build check: `env -u GOROOT go build ./...`.

---

## Task 1: queue — `meeting:cancelled` task

**Files:** Modify `backend/internal/infrastructure/queue/asynq/queue.go`.

- [ ] **Step 1: Add the task.** After the existing `meeting:updated` block (after `ParseMeetingUpdated`), add:

```go
const TaskMeetingCancelled = "meeting:cancelled"

type MeetingCancelledPayload struct {
	WorkspaceID string `json:"workspace_id"`
	MeetingID   string `json:"meeting_id"`
}

func (c *Client) EnqueueMeetingCancelled(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
	p, _ := json.Marshal(MeetingCancelledPayload{
		WorkspaceID: workspaceID.String(),
		MeetingID:   meetingID.String(),
	})
	task := asynq.NewTask(TaskMeetingCancelled, p)
	_, err := c.client.EnqueueContext(ctx, task, asynq.MaxRetry(5))
	return err
}

func ParseMeetingCancelled(t *asynq.Task) (MeetingCancelledPayload, error) {
	var p MeetingCancelledPayload
	err := json.Unmarshal(t.Payload(), &p)
	return p, err
}
```

- [ ] **Step 2: Build.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./...` → OK.

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/queue/asynq/queue.go && git commit -m "feat(queue): meeting:cancelled task

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 2: notifier — `buildCancelledMessage` + `HandleCancelled`

**Files:** Modify `backend/internal/platform/meeting_notifier/message.go`, `message_test.go`, `notifier.go`.

Context: `message.go` has `buildRemovedMessage(name string, startsAt time.Time, loc *time.Location) string` and `tzLabel`. `notifier.go` has `HandleUpdated` (the structural template) and imports `cmp,fmt,time`, bot, uuid, zap, postgres, meetingrecipients.

- [ ] **Step 1: Failing test.** In `message_test.go`, add:

```go
func TestBuildCancelledMessage(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	start := time.Date(2026, 5, 31, 14, 0, 0, 0, loc)
	m := buildCancelledMessage("Разработка | Планёрка", start, loc)
	for _, want := range []string{"отменена", "Разработка | Планёрка", "31.05.2026", "UTC+5"} {
		if !strings.Contains(m, want) {
			t.Fatalf("message %q missing %q", m, want)
		}
	}
	if strings.Contains(m, "🔗") {
		t.Fatal("cancelled message has no meet link")
	}
}
```

- [ ] **Step 2: Run, verify fail.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -run TestBuildCancelledMessage -v` → FAIL.

- [ ] **Step 3: Implement message.** In `message.go`, add after `buildRemovedMessage`:

```go
func buildCancelledMessage(name string, startsAt time.Time, loc *time.Location) string {
	s := startsAt.In(loc)
	return fmt.Sprintf("❌ Встреча отменена\n«%s»\n🗓 %s (%s)", name, s.Format("02.01.2006"), tzLabel(s))
}
```

- [ ] **Step 4: Run, verify pass.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v` → PASS.

- [ ] **Step 5: Add handler.** In `notifier.go`, add after `HandleUpdated`:

```go
// HandleCancelled DMs the meeting's recipients that it was cancelled. The meeting
// is already status='cancelled' but GetMeeting has no status filter. Best-effort
// sends; returns an error only on read failures (asynq retries before any send).
func (n *Notifier) HandleCancelled(ctx context.Context, workspaceID, meetingID uuid.UUID) error {
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
	text := buildCancelledMessage(m.Name, m.StartsAt, loc)

	recs, err := meetingrecipients.Resolve(ctx, n.store, m)
	if err != nil {
		return fmt.Errorf("resolve recipients: %w", err)
	}
	for _, r := range recs {
		if _, err := n.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: r.TelegramID,
			Text:   text,
		}); err != nil {
			n.log.Warn("send meeting cancelled",
				zap.Int64("telegram_id", r.TelegramID),
				zap.String("meeting_id", m.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}
```

- [ ] **Step 6: Build + test.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go test ./internal/platform/meeting_notifier/ -v` → build OK, PASS.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meeting_notifier/ && git commit -m "feat(meetings): meeting-cancelled DM notification

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 3: repo — `CancelSeriesOccurrences`

**Files:** Modify `backend/internal/infrastructure/persistence/postgres/meeting_repo.go`.

- [ ] **Step 1: Add the method** (after `CancelMeeting`):

```go
// CancelSeriesOccurrences cancels (status='cancelled') the scheduled occurrences
// of a series at or after fromStart, in one atomic statement. Returns the count.
func (s *Store) CancelSeriesOccurrences(ctx context.Context, workspaceID, seriesID uuid.UUID, fromStart time.Time) (int, error) {
	ct, err := s.pool.Exec(ctx, `
		UPDATE meetings SET status = 'cancelled', updated_at = now()
		WHERE series_id = $1 AND workspace_id = $2 AND starts_at >= $3 AND status = 'scheduled'`,
		seriesID, workspaceID, fromStart)
	if err != nil {
		return 0, err
	}
	return int(ct.RowsAffected()), nil
}
```

- [ ] **Step 2: Build + vet.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/infrastructure/persistence/postgres/` → clean.

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/infrastructure/persistence/postgres/meeting_repo.go && git commit -m "feat(meetings): CancelSeriesOccurrences

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 4: application — `enqueueCancelled` + `CancelMeeting` retrofit + `CancelSeries`

**Files:** Modify `backend/internal/application/meeting_service.go` (enqueueCancelled + CancelMeeting), `backend/internal/application/series_edit.go` (CancelSeries).

Context: `Services{Store, Cipher, Queue, Calendar, Log}`. `ownerOrOrganizer`, `deleteEventsBestEffort(ctx, cal CalendarService, ids []string)` exist. `Store.ListSeriesOccurrences(ctx, ws, seriesID, fromStart) ([]postgres.Meeting, error)`, `Store.CancelSeriesOccurrences(ctx, ws, seriesID, fromStart) (int, error)`, `Store.CancelMeeting(ctx, ws, id) error`, `Store.GetMeeting`, `Store.GetWorkspace`. `Queue.EnqueueMeetingCancelled(ctx, ws, id) error`. `s.Calendar.For(ctx, ws)`.

- [ ] **Step 1: Add `enqueueCancelled` + retrofit `CancelMeeting`.** In `meeting_service.go`, add the helper (next to `enqueueCreated`):

```go
// enqueueCancelled best-effort enqueues the meeting-cancelled notification.
func (s *Services) enqueueCancelled(ctx context.Context, workspaceID, meetingID uuid.UUID) {
	if s.Queue == nil {
		return
	}
	if err := s.Queue.EnqueueMeetingCancelled(ctx, workspaceID, meetingID); err != nil && s.Log != nil {
		s.Log.Warn("enqueue meeting cancelled",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("meeting_id", meetingID.String()),
			zap.Error(err))
	}
}
```

Replace the body of `CancelMeeting` with (adds the not-scheduled early-return + the enqueue):

```go
func (s *Services) CancelMeeting(ctx context.Context, workspaceID, userID, id uuid.UUID) error {
	m, err := s.Store.GetMeeting(ctx, workspaceID, id)
	if err != nil {
		return err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if !ownerOrOrganizer(w, m.OrganizerUserID, userID) {
		return ErrForbidden
	}
	if m.Status != "scheduled" {
		return nil // already cancelled or past — nothing to do
	}
	if m.GoogleEventID != "" {
		if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr == nil {
			_ = calSvc.DeleteEvent(ctx, m.GoogleEventID) // best-effort
		}
	}
	if err := s.Store.CancelMeeting(ctx, workspaceID, id); err != nil {
		return err
	}
	s.enqueueCancelled(ctx, workspaceID, id)
	return nil
}
```

- [ ] **Step 2: Add `CancelSeries`.** In `series_edit.go`, append:

```go
// CancelSeries cancels the picked occurrence and all later scheduled ones of its
// series (organizer or owner only): cancels in one atomic UPDATE, deletes the
// Google events best-effort, and enqueues one cancellation notification. Returns
// the number of occurrences cancelled.
func (s *Services) CancelSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID) (int, error) {
	picked, err := s.Store.GetMeeting(ctx, workspaceID, meetingID)
	if err != nil {
		return 0, err
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	if !ownerOrOrganizer(w, picked.OrganizerUserID, userID) {
		return 0, ErrForbidden
	}
	if picked.SeriesID == nil {
		return 0, fmt.Errorf("%w: not a series", ErrInvalidInput)
	}
	occs, err := s.Store.ListSeriesOccurrences(ctx, workspaceID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}
	n, err := s.Store.CancelSeriesOccurrences(ctx, workspaceID, *picked.SeriesID, picked.StartsAt)
	if err != nil {
		return 0, err
	}
	// Google best-effort: events are irreversible deletes, so DB-first (above)
	// keeps Postgres the source of truth; a lingering event is logged, not fatal.
	if calSvc, ferr := s.Calendar.For(ctx, workspaceID); ferr != nil {
		if s.Log != nil {
			s.Log.Warn("cancel series calendar provider", zap.String("workspace_id", workspaceID.String()), zap.Error(ferr))
		}
	} else {
		var ids []string
		for _, oc := range occs {
			if oc.GoogleEventID != "" {
				ids = append(ids, oc.GoogleEventID)
			}
		}
		s.deleteEventsBestEffort(ctx, calSvc, ids)
	}
	s.enqueueCancelled(ctx, workspaceID, meetingID)
	return n, nil
}
```

(`series_edit.go` already imports `context`, `fmt`, `uuid`, `zap`, `postgres` from the §4.4.2 work. Confirm; add any missing.)

- [ ] **Step 3: Build + vet + test.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./internal/application/ && env -u GOROOT go test ./internal/application/` → build OK; existing tests PASS. (Orchestration build-verified per convention.)

- [ ] **Step 4: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/application/ && git commit -m "feat(meetings): CancelMeeting notify + CancelSeries

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 5: `meetingedit` — delete button + confirm + scope-aware delete

**Files:** Modify `backend/internal/platform/meetingedit/service.go`, `service_test.go`.

- [ ] **Step 1: Reword the scope screen + add the delete button.** In `service.go`:

(a) Reword `scopeReply` (neutral text; callbacks unchanged):

```go
func scopeReply() Reply {
	return Reply{
		Text: "Эта встреча или вся серия (эта и далее)?",
		Edit: true,
		Keyboard: [][]Button{
			{{Text: "📍 Эта встреча", Data: "medit:scope:one"}},
			{{Text: "🔁 Вся серия (эта и далее)", Data: "medit:scope:series"}},
		},
	}
}
```

(b) Add a `{🗑 Удалить}` row to BOTH branches of `menuKeyboard(scope)`, immediately before the `{Применить}{Отмена}` row. For the series branch:

```go
		return [][]Button{
			{{Text: "🕒 Время", Data: "medit:field:datetime"}},
			{{Text: "🏢 Отдел", Data: "medit:field:dept"}, {Text: "🏷 Тип", Data: "medit:field:type"}},
			{{Text: "🎤 Ведущий", Data: "medit:field:host"}, {Text: "📝 Описание", Data: "medit:field:description"}},
			{{Text: "🗑 Удалить", Data: "medit:delete"}},
			{{Text: "✅ Применить", Data: "medit:apply"}, {Text: "✖ Отмена", Data: "medit:cancel"}},
		}
```

For the "one" branch:

```go
	return [][]Button{
		{{Text: "📅 Дата/время", Data: "medit:field:datetime"}},
		{{Text: "🏢 Отдел", Data: "medit:field:dept"}, {Text: "🏷 Тип", Data: "medit:field:type"}},
		{{Text: "🎤 Ведущий", Data: "medit:field:host"}, {Text: "📝 Описание", Data: "medit:field:description"}},
		{{Text: "🔁 Частота", Data: "medit:field:rec"}},
		{{Text: "👥 Участники", Data: "medit:parts"}},
		{{Text: "🗑 Удалить", Data: "medit:delete"}},
		{{Text: "✅ Применить", Data: "medit:apply"}, {Text: "✖ Отмена", Data: "medit:cancel"}},
	}
```

- [ ] **Step 2: Route the delete callbacks.** In `OnCallback`, add (before the final `return Reply{}, false`):

```go
	case data == "medit:delete":
		return s.confirmDelete(ctx, telegramID), true
	case data == "medit:delconf":
		return s.doDelete(ctx, telegramID), true
```

- [ ] **Step 3: Implement `confirmDelete` + `doDelete`.** Add (near `apply`):

```go
func (s *Service) confirmDelete(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	text := "Удалить эту встречу?"
	if st.Scope == "series" {
		text = "Удалить всю серию (эту и далее)? Это отменит все будущие встречи серии."
	}
	return Reply{
		Text: text,
		Edit: true,
		Keyboard: [][]Button{
			{{Text: "✅ Да, удалить", Data: "medit:delconf"}},
			{{Text: "⬅ Отмена", Data: "medit:menu"}},
		},
	}
}

func (s *Service) doDelete(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /edit"}
	}
	ws, _ := uuid.Parse(st.WorkspaceID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)

	if st.Scope == "series" {
		n, err := s.backend.CancelSeries(ctx, ws, uid, mid)
		if err != nil {
			return s.deleteErrReply(ctx, telegramID, err)
		}
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: fmt.Sprintf("Удалено встреч серии: %d ❌", n)}
	}
	if err := s.backend.CancelMeeting(ctx, ws, uid, mid); err != nil {
		return s.deleteErrReply(ctx, telegramID, err)
	}
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: "Встреча удалена ❌"}
}

func (s *Service) deleteErrReply(ctx context.Context, telegramID int64, err error) Reply {
	switch {
	case errors.Is(err, application.ErrForbidden):
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: "Нет доступа к этой встрече."}
	case errors.Is(err, postgres.ErrMeetingNotEditable):
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: "Встреча больше недоступна."}
	default:
		return Reply{Text: "Не удалось удалить, попробуй позже."}
	}
}
```

- [ ] **Step 4: Extend the Backend interface.** Add to `Backend`:

```go
	CancelMeeting(ctx context.Context, workspaceID, userID, meetingID uuid.UUID) error
	CancelSeries(ctx context.Context, workspaceID, userID, meetingID uuid.UUID) (int, error)
```

- [ ] **Step 5: Tests.** In `service_test.go`, extend `fakeBackend` with `cancelledOne bool`, `cancelledSeries int`, and the methods:

```go
func (f *fakeBackend) CancelMeeting(_ context.Context, _, _, _ uuid.UUID) error {
	f.cancelledOne = true
	return nil
}
func (f *fakeBackend) CancelSeries(_ context.Context, _, _, _ uuid.UUID) (int, error) {
	f.cancelledSeries++
	return 4, nil
}
```

Add tests:

```go
func TestDeleteFlow_Single(t *testing.T) {
	ctx := context.Background()
	m := sampleMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(90)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String()) // non-series → Scope=one menu
	if r, _ := svc.OnCallback(ctx, tg, "medit:delete"); !strings.Contains(r.Text, "Удалить эту встречу") {
		t.Fatalf("confirm: %+v", r)
	}
	if r, _ := svc.OnCallback(ctx, tg, "medit:delconf"); !strings.Contains(r.Text, "удалена") {
		t.Fatalf("delconf reply: %+v", r)
	}
	if !be.cancelledOne || be.cancelledSeries != 0 {
		t.Fatalf("expected single cancel, got one=%v series=%d", be.cancelledOne, be.cancelledSeries)
	}
}

func TestDeleteFlow_Series(t *testing.T) {
	ctx := context.Background()
	m := seriesMeeting()
	be := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}, applied: m.Meeting}
	svc := New(be, newMemSessions())
	const tg = int64(91)
	svc.OnCallback(ctx, tg, "medit:pick:"+m.ID.String())
	svc.OnCallback(ctx, tg, "medit:scope:series")
	if r, _ := svc.OnCallback(ctx, tg, "medit:delete"); !strings.Contains(r.Text, "всю серию") {
		t.Fatalf("series confirm: %+v", r)
	}
	if r, _ := svc.OnCallback(ctx, tg, "medit:delconf"); !strings.Contains(r.Text, "серии") {
		t.Fatalf("series delconf reply: %+v", r)
	}
	if be.cancelledSeries != 1 || be.cancelledOne {
		t.Fatalf("expected series cancel, got one=%v series=%d", be.cancelledOne, be.cancelledSeries)
	}
}
```

(`sampleMeeting` / `seriesMeeting` already exist in this test file.)

- [ ] **Step 6: Run + build.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go test ./internal/platform/meetingedit/ -v && env -u GOROOT go build ./...` → all PASS, build OK.

- [ ] **Step 7: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/internal/platform/meetingedit/ && git commit -m "feat(meetings): delete (this/whole series) in /edit FSM

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 6: wire the `meeting:cancelled` asynq handler (main.go)

**Files:** Modify `backend/cmd/server/main.go`.

- [ ] **Step 1: Add the handler + register it.** After the existing `meetingUpdatedHandler`, add:

```go
	meetingCancelledHandler := func(c context.Context, t *asynq.Task) error {
		p, err := asynqqueue.ParseMeetingCancelled(t)
		if err != nil {
			return err
		}
		wid, _ := uuid.Parse(p.WorkspaceID)
		mid, _ := uuid.Parse(p.MeetingID)
		return notifier.HandleCancelled(c, wid, mid)
	}
```

and add it to the `NewServer` map:

```go
		asynqqueue.TaskMeetingCancelled: meetingCancelledHandler,
```

- [ ] **Step 2: Build + vet.** `cd /Users/temirlan/Workspace/in-house/lead-cat/backend && env -u GOROOT go build ./... && env -u GOROOT go vet ./cmd/server/` → clean.

- [ ] **Step 3: Commit.**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add backend/cmd/server/main.go && git commit -m "feat(meetings): register meeting:cancelled handler

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Task 7: full verification + docs

**Files:** Modify `docs/MEETINGS.md`.

- [ ] **Step 1: Full suite from the repo ROOT.** `cd /Users/temirlan/Workspace/in-house/lead-cat && make test && make lint && make build`. `make lint` runs golangci-lint (incl. gofmt + errcheck). If gofmt issues → `gofmt -w` the listed backend files and re-run lint. If a real failure occurs, STOP and report BLOCKED.

- [ ] **Step 2: Document.** In `docs/MEETINGS.md`, in the "Backend (planned)" blockquote list, after the "Recurring-series editing (§4.4.2, done)" line, add:

```markdown
> **Deletion + cancellation notice (§4.5, done):** `/edit` gains a "🗑 Удалить" action with a confirmation step; for a series the chosen scope ("эта встреча / вся серия") governs it — single → `CancelMeeting`, series → `CancelSeries` (cancels the picked occurrence and all later scheduled ones in one atomic UPDATE, deletes Google events best-effort). All cancellations (including the single path and REST delete) now enqueue a `meeting:cancelled` DM to participants + organizer (one per delete). `CancelMeeting` is idempotent (no-op on an already-cancelled meeting). Restoring cancelled meetings and deleting past occurrences are out of scope.
```

- [ ] **Step 3: Commit (do NOT add frontend/vite.config.ts; add any gofmt-fixed backend files).**

```bash
cd /Users/temirlan/Workspace/in-house/lead-cat && git add docs/MEETINGS.md && git commit -m "docs(meetings): document §4.5 deletion + cancellation notice

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Self-review notes

- **Spec coverage:** `meeting:cancelled` queue (Task 1) · notifier message + handler (Task 2) · `CancelSeriesOccurrences` (Task 3) · `CancelMeeting` retrofit + `CancelSeries` + `enqueueCancelled` (Task 4) · FSM delete button + confirm + scope-aware + scope reword + Backend methods (Task 5) · handler wiring (Task 6) · testing (Tasks 2,5,7) · docs (Task 7). Out-of-scope (past occurrences, restore) recorded in spec + Task 7 note. All covered.
- **Type consistency:** `TaskMeetingCancelled`/`MeetingCancelledPayload`/`EnqueueMeetingCancelled`/`ParseMeetingCancelled` (Task 1) used in Tasks 4,6. `HandleCancelled` + `buildCancelledMessage` (Task 2) called in Task 6. `CancelSeriesOccurrences` (Task 3) called by `CancelSeries` (Task 4). `enqueueCancelled` (Task 4) used by `CancelMeeting`+`CancelSeries`. `meetingedit.Backend.CancelMeeting/CancelSeries` (Task 5) satisfied by `*application.Services` (CancelMeeting retrofit + CancelSeries in Task 4). `deleteEventsBestEffort`/`ListSeriesOccurrences` reused. Scope `one`/`series` from §4.4.2 drives `doDelete`.
- **No placeholders:** every code/command step is concrete. `deleteErrReply` is a shared helper for both delete branches.

```

```
