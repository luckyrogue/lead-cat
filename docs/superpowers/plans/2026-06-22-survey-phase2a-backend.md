# Survey Phase 2a — Backend Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Go application core for survey-on-meeting-decline: extend the survey-response model for a meeting source, refactor the submit path to be response-id based, and add the RSVP command (status + survey resolution + meeting-decline response) — all pure/unit-testable, with no Telegram in the application layer.

**Architecture:** Clean Architecture as in Phase 1. `RecordRSVP` is a pure application command (testable with a fake `Repository`); it returns a struct describing what to message, and the **bot layer** (a separate plan) performs the Telegram I/O. This keeps `application`/`domain` free of Telegram per AGENTS.md.

**Tech Stack:** Go, pgx (raw SQL), goose migrations, `github.com/google/uuid`, zap. Tests use the in-package fake-`Repository` pattern.

## Global Constraints

- Module path `github.com/luckyrogue/lead-cat`; backend root `apps/backend`.
- Migrations in `apps/backend/migrations/`, `YYYYMMDDHHMMSS_<name>.sql`, goose Up/Down. Use a timestamp later than `20260621120000`.
- Domain/application stay free of Fiber/pgx/Telegram. CQRS: commands change state and return data/IDs; queries are side-effect free. `RecordRSVP` and `CreateMeetingDeclineResponse` are commands.
- `survey_responses` is ONE table for both sources; `source IN ('web_booking','meeting_decline')`. `token` becomes nullable but keeps its `UNIQUE` constraint (Postgres treats NULLs as distinct — many `token=NULL` meeting rows are fine; do NOT add a partial index).
- `booker_email`/`booker_name` are reused as respondent fields (no rename).
- RSVP statuses exactly `invited|accepted|declined`. Decline reason for this source is exactly `meeting_decline`.
- Survey resolution: `COALESCE(meeting.survey_on_decline_id, org.survey_on_decline_id)`; used only if found AND `is_active`.
- gofmt/go vet clean; `go test ./...` green before each commit.

---

## File map

Create:
- `apps/backend/migrations/20260622120000_survey_meeting_decline.sql`
- `apps/backend/internal/application/rsvp.go`
- `apps/backend/internal/application/rsvp_test.go`

Modify:
- `apps/backend/internal/application/model/survey.go` — `SurveyResponse` gains `Source`, `MeetingID *uuid.UUID`, `ParticipantTelegramID *int64`.
- `apps/backend/internal/application/model/model.go` — `MeetingParticipant` gains `RSVPStatus string`.
- `apps/backend/internal/infrastructure/persistence/postgres/survey_response_repo.go` — persist/scan the new columns; add `GetSurveyResponse(id)`.
- `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go` — `UpdateParticipantRSVP`, `GetParticipant`, scan `rsvp_status` in `ListParticipants`.
- `apps/backend/internal/infrastructure/persistence/postgres/survey_repo.go` (or a new `survey_assignment_repo.go`) — `ResolveDeclineSurvey(meetingID)` query.
- `apps/backend/internal/application/repository.go` — add the new methods to the `Repository` interface.
- `apps/backend/internal/application/survey_submit.go` — extract `SubmitSurveyResponse(responseID, answers)`; `SubmitSurvey(token)` wraps it.

---

## Task 1: Migration

**Files:**
- Create: `apps/backend/migrations/20260622120000_survey_meeting_decline.sql`

**Interfaces:**
- Produces: new columns on `survey_responses`, `organizations`, `meetings`, `meeting_participants`.

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
ALTER TABLE survey_responses ALTER COLUMN token DROP NOT NULL;
ALTER TABLE survey_responses
  ADD COLUMN source TEXT NOT NULL DEFAULT 'web_booking'
      CHECK (source IN ('web_booking','meeting_decline')),
  ADD COLUMN meeting_id UUID REFERENCES meetings(id) ON DELETE SET NULL,
  ADD COLUMN participant_telegram_id BIGINT;

ALTER TABLE organizations
  ADD COLUMN survey_on_decline_id UUID REFERENCES surveys(id) ON DELETE SET NULL;
ALTER TABLE meetings
  ADD COLUMN survey_on_decline_id UUID REFERENCES surveys(id) ON DELETE SET NULL;

ALTER TABLE meeting_participants
  ADD COLUMN rsvp_status TEXT NOT NULL DEFAULT 'invited'
      CHECK (rsvp_status IN ('invited','accepted','declined'));

-- +goose Down
ALTER TABLE meeting_participants DROP COLUMN rsvp_status;
ALTER TABLE meetings DROP COLUMN survey_on_decline_id;
ALTER TABLE organizations DROP COLUMN survey_on_decline_id;
ALTER TABLE survey_responses DROP COLUMN participant_telegram_id;
ALTER TABLE survey_responses DROP COLUMN meeting_id;
ALTER TABLE survey_responses DROP COLUMN source;
ALTER TABLE survey_responses ALTER COLUMN token SET NOT NULL;
```

> Note: confirm the org FK target. Phase-1 `survey_responses.organization_id` references `organizations`. The `meetings` table column for the org may be named `workspace_id` (referencing `organizations` after the workspaces→organizations rename). The new `meetings.survey_on_decline_id` references `surveys(id)` regardless — no dependence on the org column name. The Down's `ALTER COLUMN token SET NOT NULL` only succeeds if no meeting rows exist; this is a dev-only rollback, acceptable.

- [ ] **Step 2: Apply + roll back**

Run: `cd apps/backend && make migrate` then verify columns:
`psql "$DATABASE_URL" -c '\d survey_responses' -c '\d organizations' -c '\d meetings' -c '\d meeting_participants'`
Expected: the new columns present. Then `make migrate-down` and re-apply to confirm Down is valid (on an empty DB).

- [ ] **Step 3: Commit**

```bash
git add apps/backend/migrations/20260622120000_survey_meeting_decline.sql
git commit -m "feat(surveys): migration — meeting-decline source, assignment, RSVP status"
```

---

## Task 2: Model extensions

**Files:**
- Modify: `apps/backend/internal/application/model/survey.go`
- Modify: `apps/backend/internal/application/model/model.go`

**Interfaces:**
- Produces:
  - `SurveyResponse` gains `Source string \`json:"source"\``, `MeetingID *uuid.UUID \`json:"meeting_id"\``, `ParticipantTelegramID *int64 \`json:"-"\``.
  - `MeetingParticipant` gains `RSVPStatus string \`json:"rsvp_status"\``.
  - const block: `SourceWebBooking = "web_booking"`, `SourceMeetingDecline = "meeting_decline"`, `RSVPInvited = "invited"`, `RSVPAccepted = "accepted"`, `RSVPDeclined = "declined"`, `DeclineReasonMeeting = "meeting_decline"`.

- [ ] **Step 1: Add fields + consts to `model/survey.go`**

In the `SurveyResponse` struct add:
```go
	Source                string     `json:"source"`
	MeetingID             *uuid.UUID `json:"meeting_id"`
	ParticipantTelegramID *int64     `json:"-"`
```
And a const block:
```go
const (
	SourceWebBooking     = "web_booking"
	SourceMeetingDecline = "meeting_decline"
	DeclineReasonMeeting = "meeting_decline"
	RSVPInvited          = "invited"
	RSVPAccepted         = "accepted"
	RSVPDeclined         = "declined"
)
```

- [ ] **Step 2: Add `RSVPStatus` to `MeetingParticipant` in `model/model.go`**

```go
	RSVPStatus string `json:"rsvp_status"`
```

- [ ] **Step 3: Build**

Run: `cd apps/backend && go build ./...`
Expected: compiles (existing code that constructs `SurveyResponse`/`MeetingParticipant` still works — new fields are zero-valued).

- [ ] **Step 4: Commit**

```bash
git add apps/backend/internal/application/model/survey.go apps/backend/internal/application/model/model.go
git commit -m "feat(surveys): model — response source/meeting fields + participant rsvp_status"
```

---

## Task 3: Repository — persistence for the new columns

**Files:**
- Modify: `apps/backend/internal/infrastructure/persistence/postgres/survey_response_repo.go`
- Modify: `apps/backend/internal/infrastructure/persistence/postgres/meeting_repo.go`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/survey_assignment_repo.go`
- Modify: `apps/backend/internal/application/repository.go`

**Interfaces:**
- Produces (added to `Repository` and implemented on `*Store`):
  - `GetSurveyResponse(ctx, id uuid.UUID) (model.SurveyResponse, error)` — by id (for the bot).
  - `UpdateParticipantRSVP(ctx, meetingID uuid.UUID, email, status string) error`.
  - `GetParticipant(ctx, meetingID uuid.UUID, email string) (model.MeetingParticipant, bool, error)`.
  - `ResolveDeclineSurvey(ctx, meetingID uuid.UUID) (uuid.UUID, bool, error)` — returns the resolved active survey id (`COALESCE(meeting, org)` + `is_active`), `ok=false` if none.
  - `GetMeetingForRSVP(ctx, meetingID uuid.UUID) (model.RSVPMeetingInfo, error)` — org id, organizer_user_id, meeting name (for the RSVP command).
  - `GetOpenMeetingResponse(ctx, meetingID uuid.UUID, telegramID int64) (model.SurveyResponse, bool, error)` — an existing unfinished (`status='sent'`) meeting-decline response for the same meeting+participant, for dedup (spec §3c).
- Consumes: existing `Store` (`s.pool`, `rowScanner`), `CreateSurveyResponse` (extend to write the new columns).

> Add `model.RSVPMeetingInfo{OrgID uuid.UUID; OrganizerUserID *uuid.UUID; Title string}` to `model/survey.go` as part of Step 1 (it is a small read DTO).

- [ ] **Step 1: Extend `CreateSurveyResponse` + add `GetSurveyResponse` in `survey_response_repo.go`**

Update `CreateSurveyResponse`'s INSERT to include `source, meeting_id, participant_telegram_id` (default `source` to `'web_booking'` when empty so Phase-1 callers are unaffected):

```go
func (s *Store) CreateSurveyResponse(ctx context.Context, r model.SurveyResponse) (model.SurveyResponse, error) {
	if r.Source == "" {
		r.Source = model.SourceWebBooking
	}
	answers, err := json.Marshal([]model.Answer{})
	if err != nil {
		return model.SurveyResponse{}, err
	}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO survey_responses
			(survey_id, organization_id, booking_event_type_id, token, booker_email, booker_name,
			 decline_reason, status, answers, source, meeting_id, participant_telegram_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'sent',$8,$9,$10,$11)
		 RETURNING id, created_at`,
		r.SurveyID, r.OrganizationID, r.BookingEventTypeID, nullableToken(r.Token), r.BookerEmail, r.BookerName,
		r.DeclineReason, answers, r.Source, r.MeetingID, r.ParticipantTelegramID).
		Scan(&r.ID, &r.CreatedAt)
	r.Status = "sent"
	return r, err
}

// nullableToken stores "" as SQL NULL (web tokens are non-empty; meeting responses have none).
func nullableToken(t string) any {
	if t == "" {
		return nil
	}
	return t
}
```

Update `responseCols` and `scanResponse` to include the three new columns, and add `GetSurveyResponse`:

```go
const responseCols = `id, survey_id, organization_id, booking_event_type_id, token,
	booker_email, booker_name, decline_reason, status, answers, created_at, completed_at,
	source, meeting_id, participant_telegram_id`

func scanResponse(row rowScanner) (model.SurveyResponse, error) {
	var r model.SurveyResponse
	var raw []byte
	var token *string
	if err := row.Scan(&r.ID, &r.SurveyID, &r.OrganizationID, &r.BookingEventTypeID, &token,
		&r.BookerEmail, &r.BookerName, &r.DeclineReason, &r.Status, &raw, &r.CreatedAt, &r.CompletedAt,
		&r.Source, &r.MeetingID, &r.ParticipantTelegramID); err != nil {
		return model.SurveyResponse{}, err
	}
	if token != nil {
		r.Token = *token
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &r.Answers)
	}
	return r, nil
}

func (s *Store) GetSurveyResponse(ctx context.Context, id uuid.UUID) (model.SurveyResponse, error) {
	return scanResponse(s.pool.QueryRow(ctx, `SELECT `+responseCols+` FROM survey_responses WHERE id=$1`, id))
}
```

> The `token` column is now scanned into a `*string` because it is nullable. `GetSurveyResponseByToken` already filters `WHERE token=$1` (never matches NULL) — fine; update its scan to use `scanResponse` too so column order stays consistent.

- [ ] **Step 2: Add participant RSVP methods in `meeting_repo.go`**

```go
func (s *Store) UpdateParticipantRSVP(ctx context.Context, meetingID uuid.UUID, email, status string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE meeting_participants SET rsvp_status=$3 WHERE meeting_id=$1 AND email=$2`,
		meetingID, email, status)
	return err
}

func (s *Store) GetParticipant(ctx context.Context, meetingID uuid.UUID, email string) (model.MeetingParticipant, bool, error) {
	var p model.MeetingParticipant
	err := s.pool.QueryRow(ctx,
		`SELECT employee_id, email, rsvp_status FROM meeting_participants WHERE meeting_id=$1 AND email=$2`,
		meetingID, email).Scan(&p.EmployeeID, &p.Email, &p.RSVPStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MeetingParticipant{}, false, nil
	}
	return p, err == nil, err
}
```

Also update `ListParticipants`' SELECT + scan to include `rsvp_status` (so existing reads carry the status). Add `"database/sql"` and `"errors"` imports if missing.

- [ ] **Step 3: Add `ResolveDeclineSurvey` + `GetMeetingForRSVP` in `survey_assignment_repo.go`**

```go
package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

// ResolveDeclineSurvey returns the active survey assigned for a meeting decline:
// the meeting override if set, else the org default — and only if is_active.
func (s *Store) ResolveDeclineSurvey(ctx context.Context, meetingID uuid.UUID) (uuid.UUID, bool, error) {
	var sid uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT sv.id
		FROM meetings m
		JOIN organizations o ON o.id = m.workspace_id
		JOIN surveys sv ON sv.id = COALESCE(m.survey_on_decline_id, o.survey_on_decline_id)
		WHERE m.id = $1 AND sv.is_active = true`, meetingID).Scan(&sid)
	if err != nil {
		return uuid.Nil, false, ignoreNoRows(err)
	}
	return sid, true, nil
}

func (s *Store) GetMeetingForRSVP(ctx context.Context, meetingID uuid.UUID) (model.RSVPMeetingInfo, error) {
	var info model.RSVPMeetingInfo
	err := s.pool.QueryRow(ctx,
		`SELECT workspace_id, organizer_user_id, name FROM meetings WHERE id=$1`, meetingID).
		Scan(&info.OrgID, &info.OrganizerUserID, &info.Title)
	return info, err
}

// GetOpenMeetingResponse finds an unfinished meeting-decline response for the
// same meeting+participant (dedup of a repeat decline, spec §3c).
func (s *Store) GetOpenMeetingResponse(ctx context.Context, meetingID uuid.UUID, telegramID int64) (model.SurveyResponse, bool, error) {
	r, err := scanResponse(s.pool.QueryRow(ctx,
		`SELECT `+responseCols+` FROM survey_responses
		 WHERE meeting_id=$1 AND participant_telegram_id=$2 AND status='sent'
		 ORDER BY created_at DESC LIMIT 1`, meetingID, telegramID))
	if errors.Is(err, sql.ErrNoRows) {
		return model.SurveyResponse{}, false, nil
	}
	return r, err == nil, err
}
```

> Add `"database/sql"` and `"errors"` imports to `survey_assignment_repo.go` (or place `GetOpenMeetingResponse` in `survey_response_repo.go` where `responseCols`/`scanResponse` live — preferred, so it shares the column list).

> Confirm the meeting→org column name (`workspace_id` here per the migration; if a later migration renamed it to `organization_id`, use that). `ignoreNoRows` returns nil for `sql.ErrNoRows`; if no such helper exists, inline `if errors.Is(err, sql.ErrNoRows) { return uuid.Nil, false, nil }`.

- [ ] **Step 4: Add the 5 methods to the `Repository` interface**

In `repository.go`:
```go
	GetSurveyResponse(ctx context.Context, id uuid.UUID) (model.SurveyResponse, error)
	UpdateParticipantRSVP(ctx context.Context, meetingID uuid.UUID, email, status string) error
	GetParticipant(ctx context.Context, meetingID uuid.UUID, email string) (model.MeetingParticipant, bool, error)
	ResolveDeclineSurvey(ctx context.Context, meetingID uuid.UUID) (uuid.UUID, bool, error)
	GetMeetingForRSVP(ctx context.Context, meetingID uuid.UUID) (model.RSVPMeetingInfo, error)
	GetOpenMeetingResponse(ctx context.Context, meetingID uuid.UUID, telegramID int64) (model.SurveyResponse, bool, error)
```

- [ ] **Step 5: Build**

Run: `cd apps/backend && go build ./... && go vet ./...`
Expected: compiles (Store satisfies the larger interface). Fix any fakes that manually implement `Repository` (grep `Repository =` / test stubs) by adding the new methods as no-ops if they fail to compile.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/infrastructure/persistence/postgres/ apps/backend/internal/application/repository.go apps/backend/internal/application/model/survey.go
git commit -m "feat(surveys): repo — response source columns, participant RSVP, decline-survey resolution"
```

---

## Task 4: Refactor — `SubmitSurveyResponse(responseID, answers)` core

**Files:**
- Modify: `apps/backend/internal/application/survey_submit.go`
- Modify: `apps/backend/internal/application/survey_submit_test.go`

**Interfaces:**
- Produces: `SubmitSurveyResponse(ctx, responseID uuid.UUID, answers []model.Answer) error` — validates against the response's survey and completes it; `ErrResponseCompleted` if already done, `ErrSurveyClosed` if inactive.
- Consumes: `GetSurveyResponse` (Task 3), existing `GetSurvey`, `CompleteSurveyResponse`, `model.ValidateAnswers`.
- `SubmitSurvey(token)` is rewritten to resolve token→id then call `SubmitSurveyResponse`.

- [ ] **Step 1: Write the failing test**

Add to `survey_submit_test.go` (the fake store already exists from Phase 1; ensure it implements `GetSurveyResponse`):

```go
func TestSubmitSurveyResponseValidatesAndCompletes(t *testing.T) {
	q := model.SurveyQuestion{ID: uuid.New(), Prompt: "Why?", Type: model.QuestionText, Required: true}
	respID := uuid.New()
	store := &submitSurveyFakeStore{
		respByID: model.SurveyResponse{ID: respID, Status: "sent", SurveyID: uuid.New()},
		survey:   model.Survey{IsActive: true, Questions: []model.SurveyQuestion{q}},
	}
	svc := newSurveySvc(store)
	if err := svc.SubmitSurveyResponse(context.Background(), respID, []model.Answer{{QuestionID: q.ID, Value: "x"}}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if store.completedID != respID {
		t.Fatalf("expected completion of %v, got %v", respID, store.completedID)
	}
}

func TestSubmitSurveyResponseRejectsCompleted(t *testing.T) {
	respID := uuid.New()
	store := &submitSurveyFakeStore{respByID: model.SurveyResponse{ID: respID, Status: "completed"}}
	svc := newSurveySvc(store)
	if err := svc.SubmitSurveyResponse(context.Background(), respID, nil); !errors.Is(err, model.ErrResponseCompleted) {
		t.Fatalf("expected ErrResponseCompleted, got %v", err)
	}
}
```

Extend `submitSurveyFakeStore` with a `respByID model.SurveyResponse` field and `GetSurveyResponse` returning it.

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/backend && go test ./internal/application/ -run 'SubmitSurveyResponse' -v`
Expected: FAIL (undefined: SubmitSurveyResponse).

- [ ] **Step 3: Implement the refactor**

```go
func (s *Services) SubmitSurveyResponse(ctx context.Context, responseID uuid.UUID, answers []model.Answer) error {
	resp, err := s.Store.GetSurveyResponse(ctx, responseID)
	if err != nil {
		return err
	}
	return s.completeResponse(ctx, resp, answers)
}

func (s *Services) SubmitSurvey(ctx context.Context, token string, answers []model.Answer) error {
	resp, err := s.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.completeResponse(ctx, resp, answers)
}

func (s *Services) completeResponse(ctx context.Context, resp model.SurveyResponse, answers []model.Answer) error {
	if resp.Status == "completed" {
		return model.ErrResponseCompleted
	}
	sv, err := s.Store.GetSurvey(ctx, resp.SurveyID)
	if err != nil {
		return err
	}
	if !sv.IsActive {
		return model.ErrSurveyClosed
	}
	normalized, err := model.ValidateAnswers(sv.Questions, answers)
	if err != nil {
		return err
	}
	return s.Store.CompleteSurveyResponse(ctx, resp.ID, normalized)
}
```

- [ ] **Step 4: Run to verify pass + the existing web submit tests**

Run: `cd apps/backend && go test ./internal/application/ -run 'SubmitSurvey' -v`
Expected: PASS (both the new response-id tests and the existing token tests).

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/application/survey_submit.go apps/backend/internal/application/survey_submit_test.go
git commit -m "refactor(surveys): SubmitSurveyResponse core, SubmitSurvey wraps it"
```

---

## Task 5: `RecordRSVP` command + meeting-decline response

**Files:**
- Create: `apps/backend/internal/application/rsvp.go`
- Create: `apps/backend/internal/application/rsvp_test.go`

**Interfaces:**
- Consumes: `GetBotUserByTelegramID`, `GetParticipant`, `UpdateParticipantRSVP`, `ResolveDeclineSurvey`, `GetMeetingForRSVP`, `CreateSurveyResponse`, `GetUserByID` (for the organizer's telegram/lang — confirm the method that maps a platform user to a bot user / telegram id; if organizer telegram lookup needs `GetBotUserByEmail`, fetch the organizer's email first).
- Produces:
  - `type RSVPResult struct { Status string; OfferSurveyResponseID *uuid.UUID; OrganizerTelegramID *int64; OrganizerLang string; ParticipantName string; MeetingTitle string }`
  - `func (s *Services) RecordRSVP(ctx context.Context, meetingID uuid.UUID, telegramID int64, status string) (RSVPResult, error)` — pure application; NO Telegram I/O. The bot layer uses the result to send messages.
  - `func (s *Services) CreateMeetingDeclineResponse(ctx, meetingID, surveyID uuid.UUID, p model.MeetingParticipant, telegramID int64, orgID uuid.UUID) (uuid.UUID, error)`.

- [ ] **Step 1: Write the failing test**

```go
package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type rsvpFakeStore struct {
	Repository
	botUser     model.BotUser
	participant model.MeetingParticipant
	isParticipant bool
	meeting     model.RSVPMeetingInfo
	surveyID    uuid.UUID
	hasSurvey   bool
	updatedStatus string
	created     model.SurveyResponse
}

func (f *rsvpFakeStore) GetBotUserByTelegramID(_ context.Context, _ int64) (model.BotUser, error) {
	return f.botUser, nil
}
func (f *rsvpFakeStore) GetParticipant(_ context.Context, _ uuid.UUID, _ string) (model.MeetingParticipant, bool, error) {
	return f.participant, f.isParticipant, nil
}
func (f *rsvpFakeStore) UpdateParticipantRSVP(_ context.Context, _ uuid.UUID, _, status string) error {
	f.updatedStatus = status
	return nil
}
func (f *rsvpFakeStore) GetMeetingForRSVP(_ context.Context, _ uuid.UUID) (model.RSVPMeetingInfo, error) {
	return f.meeting, nil
}
func (f *rsvpFakeStore) ResolveDeclineSurvey(_ context.Context, _ uuid.UUID) (uuid.UUID, bool, error) {
	return f.surveyID, f.hasSurvey, nil
}
func (f *rsvpFakeStore) CreateSurveyResponse(_ context.Context, r model.SurveyResponse) (model.SurveyResponse, error) {
	r.ID = uuid.New()
	f.created = r
	return r, nil
}

func newRSVP(store Repository) *Services { return &Services{Store: store, Log: zap.NewNop()} }

func TestRecordRSVPNonParticipant(t *testing.T) {
	store := &rsvpFakeStore{botUser: model.BotUser{Email: "a@b.c"}, isParticipant: false}
	_, err := newRSVP(store).RecordRSVP(context.Background(), uuid.New(), 1, model.RSVPDeclined)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRecordRSVPAcceptNoSurvey(t *testing.T) {
	store := &rsvpFakeStore{botUser: model.BotUser{Email: "a@b.c"}, isParticipant: true,
		participant: model.MeetingParticipant{Email: "a@b.c"}, meeting: model.RSVPMeetingInfo{Title: "Sync"}}
	res, err := newRSVP(store).RecordRSVP(context.Background(), uuid.New(), 1, model.RSVPAccepted)
	if err != nil {
		t.Fatal(err)
	}
	if store.updatedStatus != model.RSVPAccepted || res.OfferSurveyResponseID != nil {
		t.Fatalf("accept must update status and offer no survey: %+v", res)
	}
}

func TestRecordRSVPDeclineWithSurvey(t *testing.T) {
	sid := uuid.New()
	store := &rsvpFakeStore{botUser: model.BotUser{Email: "a@b.c", Language: "ru"}, isParticipant: true,
		participant: model.MeetingParticipant{Email: "a@b.c"},
		meeting:     model.RSVPMeetingInfo{OrgID: uuid.New(), Title: "Sync"},
		surveyID:    sid, hasSurvey: true}
	res, err := newRSVP(store).RecordRSVP(context.Background(), uuid.New(), 99, model.RSVPDeclined)
	if err != nil {
		t.Fatal(err)
	}
	if store.updatedStatus != model.RSVPDeclined || res.OfferSurveyResponseID == nil {
		t.Fatalf("decline+survey must set declined and offer a survey: %+v", res)
	}
	if store.created.Source != model.SourceMeetingDecline || store.created.ParticipantTelegramID == nil {
		t.Fatalf("created response wrong: %+v", store.created)
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `cd apps/backend && go test ./internal/application/ -run 'RecordRSVP' -v`
Expected: FAIL (undefined: RecordRSVP).

- [ ] **Step 3: Implement `rsvp.go`**

```go
package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type RSVPResult struct {
	Status                string
	OfferSurveyResponseID *uuid.UUID
	OrganizerTelegramID   *int64
	OrganizerLang         string
	ParticipantName       string
	MeetingTitle          string
}

func (s *Services) RecordRSVP(ctx context.Context, meetingID uuid.UUID, telegramID int64, status string) (RSVPResult, error) {
	bu, err := s.Store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return RSVPResult{}, err
	}
	p, ok, err := s.Store.GetParticipant(ctx, meetingID, bu.Email)
	if err != nil {
		return RSVPResult{}, err
	}
	if !ok {
		return RSVPResult{}, model.ErrForbidden
	}
	if err := s.Store.UpdateParticipantRSVP(ctx, meetingID, bu.Email, status); err != nil {
		return RSVPResult{}, err
	}
	mi, err := s.Store.GetMeetingForRSVP(ctx, meetingID)
	if err != nil {
		return RSVPResult{}, err
	}
	res := RSVPResult{Status: status, ParticipantName: displayName(bu, p), MeetingTitle: mi.Title}
	if status != model.RSVPDeclined {
		return res, nil
	}
	// organizer notification data (the bot layer sends it)
	if mi.OrganizerUserID != nil {
		if org, oerr := s.Store.GetBotUserByEmail(ctx, organizerEmail(ctx, s, *mi.OrganizerUserID)); oerr == nil && org.TelegramID != 0 {
			tid := org.TelegramID
			res.OrganizerTelegramID = &tid
			res.OrganizerLang = org.Language
		}
	}
	// resolve + create the survey response
	if sid, has, rerr := s.Store.ResolveDeclineSurvey(ctx, meetingID); rerr == nil && has {
		respID, cerr := s.CreateMeetingDeclineResponse(ctx, meetingID, sid, p, telegramID, mi.OrgID)
		if cerr == nil {
			res.OfferSurveyResponseID = &respID
		}
	}
	return res, nil
}

func (s *Services) CreateMeetingDeclineResponse(ctx context.Context, meetingID, surveyID uuid.UUID, p model.MeetingParticipant, telegramID int64, orgID uuid.UUID) (uuid.UUID, error) {
	mid := meetingID
	tid := telegramID
	r, err := s.Store.CreateSurveyResponse(ctx, model.SurveyResponse{
		SurveyID:              surveyID,
		OrganizationID:        orgID,
		Source:                model.SourceMeetingDecline,
		MeetingID:             &mid,
		ParticipantTelegramID: &tid,
		BookerEmail:           p.Email,
		BookerName:            p.Email, // display fallback; bot layer can pass a better name
		DeclineReason:         model.DeclineReasonMeeting,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return r.ID, nil
}

func displayName(bu model.BotUser, p model.MeetingParticipant) string {
	if bu.FullName != "" {
		return bu.FullName
	}
	return p.Email
}
```

> `organizerEmail(ctx, s, userID)` is a tiny helper that fetches the organizer's email from their platform user (use the existing `GetUserByID`/`GetPlatformUserByID` — confirm which exists and returns an email; if the organizer's bot user is found directly by user-id elsewhere, use that). If no clean path exists, fetch the platform user's email then `GetBotUserByEmail`. Confirm `model.BotUser` field names (`FullName`, `TelegramID`, `Language`, `Email`) against `model/model.go` and adjust. Add `GetBotUserByEmail` to the `Repository` interface if it is not already there (it exists on the recipients port; ensure the application `Repository` exposes it).

- [ ] **Step 4: Run to verify pass**

Run: `cd apps/backend && go test ./internal/application/ -run 'RecordRSVP' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/application/rsvp.go apps/backend/internal/application/rsvp_test.go
git commit -m "feat(surveys): RecordRSVP command + meeting-decline response (pure, no telegram)"
```

---

## Task 6: Full backend verification

- [ ] **Step 1: Build + vet + test + fmt**

Run:
```bash
cd apps/backend && gofmt -l internal/ && go vet ./... && go test ./...
```
Expected: gofmt prints nothing, vet clean, all tests PASS (Phase-1 survey/booking tests still green after the model/refactor changes).

- [ ] **Step 2: Commit any fmt fixes**

```bash
git add apps/backend && git commit -m "chore(surveys): phase2a backend fmt/verify" || echo "nothing to commit"
```

---

## Self-review notes (addressed)

- **Spec coverage:** migration incl. token-nullable-but-unique (Task 1); model source/meeting/telegram + rsvp_status (Task 2); repo persistence + GetSurveyResponse + participant RSVP + decline-survey resolution + meeting info (Task 3); SubmitSurveyResponse refactor reused by web (Task 4); RecordRSVP pure command + dedup-free creation + meeting-decline response + organizer-notification data (Task 5). Bot delivery (FSM, callbacks, notifier buttons, boti18n) and admin UI (assignment selects, responses badge, HTTP endpoints, OpenAPI) are SEPARATE plans.
- **Pure application:** `RecordRSVP` returns `RSVPResult` (telegram ids + text) and does no Telegram I/O — the bot plan sends the messages. Keeps AGENTS.md's "domain free of Telegram".
- **Type consistency:** `SubmitSurveyResponse(uuid, []Answer)` (Task 4) consumed by the bot plan; `RecordRSVP(...) (RSVPResult, error)` and `RSVPResult` (Task 5) consumed by the bot plan; `ResolveDeclineSurvey`/`GetSurveyResponse`/`GetParticipant`/`UpdateParticipantRSVP`/`GetMeetingForRSVP` (Task 3) consumed by Task 5.
- **Known confirmations for the implementer:** the meeting→org column name (`workspace_id` vs `organization_id`); `model.BotUser` field names (`FullName`/`TelegramID`/`Language`/`Email`); whether `GetBotUserByEmail` is already on the `Repository` interface; the dedup of a repeat decline (spec §3c) — if a repeat decline should reuse an existing unfinished `sent` response, add a `GetOpenMeetingResponse(meetingID, telegramID)` lookup in `CreateMeetingDeclineResponse` before inserting; this plan creates a fresh response each decline — **add the dedup lookup as Task 5 Step 3a if the repeat-decline path is exercised before the bot plan lands.**
