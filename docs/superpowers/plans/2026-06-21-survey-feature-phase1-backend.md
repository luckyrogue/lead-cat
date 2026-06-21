# Survey-on-decline Phase 1 — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Go backend for survey-on-booking-decline: survey library CRUD, public survey delivery by token, and the decline trigger that issues a survey token when a public booking fails.

**Architecture:** Clean Architecture as in the existing booking feature — `model` (domain structs + validation) → `application` (CQRS commands/queries on `Services`, backed by the `Repository` interface) → `infrastructure/persistence/postgres` (`Store` SQL methods) and `delivery/http/handlers` (Fiber handlers). Survey delivery is a public, token-based endpoint mirroring `/api/book/:slug`; admin CRUD is org-scoped via the `X-Org-Id` header, mirroring `/api/booking/event-types`.

**Tech Stack:** Go, Fiber v2, pgx (raw SQL, no ORM), goose migrations, `github.com/google/uuid`, zap. Tests use the in-package fake-`Repository` pattern (`submitFakeStore` embeds `Repository` and overrides methods) and `httptest` for handlers.

## Global Constraints

- Module path: `github.com/luckyrogue/lead-cat`; backend root `apps/backend`.
- Migrations live in `apps/backend/migrations/`, named `YYYYMMDDHHMMSS_<name>.sql`, with `-- +goose Up` / `-- +goose Down` sections. Use a timestamp later than `20260619130000`.
- Domain stays free of Fiber/pgx/Telegram (interfaces at boundaries).
- CQRS: commands change state and return errors/IDs; queries are side-effect free. `CreatePendingResponse` is a command, never called from a query.
- Question types: exactly `single`, `multi`, `rating`, `text`. Decline reasons: exactly `slot_taken`, `invalid_booking`. Response status: exactly `sent`, `completed`.
- No secrets/PII in logs. Wrap errors with `%w`; log once at the handler boundary.
- Answer storage is a self-contained snapshot: `answers` JSONB is an array of `{question_id, prompt, type, value}`.
- `survey_responses.survey_id` FK uses default `NO ACTION` (not `RESTRICT`/`CASCADE`); delete-with-responses is blocked in the application layer (returns `ErrSurveyHasResponses`).
- Run `gofmt`/`go vet` clean; `go test ./...` green before each commit.

---

## File map

Create:
- `apps/backend/migrations/20260621120000_surveys.sql`
- `apps/backend/internal/application/model/survey.go`
- `apps/backend/internal/application/model/survey_test.go`
- `apps/backend/internal/infrastructure/persistence/postgres/survey_repo.go`
- `apps/backend/internal/infrastructure/persistence/postgres/survey_response_repo.go`
- `apps/backend/internal/application/survey.go`
- `apps/backend/internal/application/survey_test.go`
- `apps/backend/internal/application/survey_submit.go`
- `apps/backend/internal/application/survey_submit_test.go`
- `apps/backend/internal/delivery/http/handlers/surveys.go`
- `apps/backend/internal/delivery/http/handlers/surveys_public.go`
- `apps/backend/internal/delivery/http/handlers/surveys_test.go`

Modify:
- `apps/backend/internal/application/repository.go` — add survey methods to the `Repository` interface.
- `apps/backend/internal/application/booking_submit.go` — issue a pending survey response on decline.
- `apps/backend/internal/delivery/http/handlers/public_booking_submit.go` — include `survey_token` in the 409/400 body.
- `apps/backend/internal/delivery/http/handlers/booking.go` — accept `survey_id` in the event-type PATCH body.
- `apps/backend/internal/application/booking.go` — `EventTypeInput` gains `SurveyID *uuid.UUID`; update/get carry it.
- `apps/backend/internal/infrastructure/persistence/postgres/booking_repo.go` — read/write the new `survey_id` column.
- `apps/backend/internal/delivery/http/app.go` — register survey routes.
- `apps/backend/openapi/openapi.json` — regenerated.

---

## Task 1: Migration

**Files:**
- Create: `apps/backend/migrations/20260621120000_surveys.sql`

**Interfaces:**
- Produces: tables `surveys`, `survey_questions`, `survey_responses`; column `booking_event_types.survey_id`.

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
CREATE TABLE surveys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX surveys_org_idx ON surveys (organization_id);

CREATE TABLE survey_questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id   UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    order_index INT  NOT NULL,
    prompt      TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('single','multi','rating','text')),
    options     TEXT[] NOT NULL DEFAULT '{}',
    rating_max  INT  NOT NULL DEFAULT 5,
    required    BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX survey_questions_survey_idx ON survey_questions (survey_id);

CREATE TABLE survey_responses (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id             UUID NOT NULL REFERENCES surveys(id),
    organization_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    booking_event_type_id UUID REFERENCES booking_event_types(id) ON DELETE SET NULL,
    token                 TEXT NOT NULL UNIQUE,
    booker_email          TEXT NOT NULL DEFAULT '',
    booker_name           TEXT NOT NULL DEFAULT '',
    decline_reason        TEXT NOT NULL DEFAULT '',
    status                TEXT NOT NULL DEFAULT 'sent' CHECK (status IN ('sent','completed')),
    answers               JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at          TIMESTAMPTZ
);
CREATE INDEX survey_responses_survey_idx ON survey_responses (survey_id);
CREATE INDEX survey_responses_org_idx ON survey_responses (organization_id);

ALTER TABLE booking_event_types
    ADD COLUMN survey_id UUID REFERENCES surveys(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE booking_event_types DROP COLUMN survey_id;
DROP TABLE survey_responses;
DROP TABLE survey_questions;
DROP TABLE surveys;
```

- [ ] **Step 2: Apply and roll back to verify**

Run: `cd apps/backend && make migrate` (or the project's goose up command), then confirm tables exist:
`psql "$DATABASE_URL" -c '\d surveys' -c '\d survey_questions' -c '\d survey_responses' -c '\d booking_event_types'`
Expected: all four show the new schema (booking_event_types has `survey_id`).

- [ ] **Step 3: Commit**

```bash
git add apps/backend/migrations/20260621120000_surveys.sql
git commit -m "feat(surveys): migration — surveys, questions, responses, event-type assignment"
```

---

## Task 2: Domain model + validation

**Files:**
- Create: `apps/backend/internal/application/model/survey.go`
- Create: `apps/backend/internal/application/model/survey_test.go`

**Interfaces:**
- Produces:
  - `QuestionType` (string) with consts `QuestionSingle`, `QuestionMulti`, `QuestionRating`, `QuestionText`.
  - `Survey{ID, OrganizationID uuid.UUID; Name string; IsActive bool; Questions []SurveyQuestion; CreatedAt, UpdatedAt time.Time}`.
  - `SurveyQuestion{ID, SurveyID uuid.UUID; OrderIndex int; Prompt string; Type QuestionType; Options []string; RatingMax int; Required bool}`.
  - `Answer{QuestionID uuid.UUID; Prompt string; Type QuestionType; Value any}`.
  - `SurveyResponse{ID, SurveyID, OrganizationID uuid.UUID; BookingEventTypeID *uuid.UUID; Token, BookerEmail, BookerName, DeclineReason, Status string; Answers []Answer; CreatedAt time.Time; CompletedAt *time.Time}`.
  - `var ErrInvalidSurvey = errors.New("invalid survey")`, `ErrSurveyHasResponses = errors.New("survey has responses")`, `ErrSurveyClosed = errors.New("survey closed")`, `ErrResponseCompleted = errors.New("response already completed")`.
  - `func (s Survey) Validate() error` — structural validation.
  - `func ValidateAnswers(questions []SurveyQuestion, answers []Answer) ([]Answer, error)` — validates submitted answers against questions and returns normalized snapshot answers (prompt/type filled from the question).

- [ ] **Step 1: Write the failing test**

```go
package model

import (
	"testing"

	"github.com/google/uuid"
)

func textQ() SurveyQuestion {
	return SurveyQuestion{ID: uuid.New(), OrderIndex: 0, Prompt: "Why?", Type: QuestionText, Required: true}
}
func singleQ() SurveyQuestion {
	return SurveyQuestion{ID: uuid.New(), OrderIndex: 1, Prompt: "Pick", Type: QuestionSingle, Options: []string{"a", "b"}, Required: true}
}
func ratingQ() SurveyQuestion {
	return SurveyQuestion{ID: uuid.New(), OrderIndex: 2, Prompt: "Rate", Type: QuestionRating, RatingMax: 5, Required: false}
}

func TestSurveyValidate(t *testing.T) {
	ok := Survey{Name: "S", Questions: []SurveyQuestion{textQ(), singleQ(), ratingQ()}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := (Survey{Name: "", Questions: []SurveyQuestion{textQ()}}).Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := (Survey{Name: "S"}).Validate(); err == nil {
		t.Fatal("expected error for zero questions")
	}
	bad := Survey{Name: "S", Questions: []SurveyQuestion{{Prompt: "p", Type: QuestionSingle, Options: []string{"only"}}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error: single needs >=2 options")
	}
	badRating := Survey{Name: "S", Questions: []SurveyQuestion{{Prompt: "p", Type: QuestionRating, RatingMax: 1}}}
	if err := badRating.Validate(); err == nil {
		t.Fatal("expected error: rating_max must be 2..10")
	}
}

func TestValidateAnswers(t *testing.T) {
	tq, sq, rq := textQ(), singleQ(), ratingQ()
	qs := []SurveyQuestion{tq, sq, rq}

	got, err := ValidateAnswers(qs, []Answer{
		{QuestionID: tq.ID, Value: "because"},
		{QuestionID: sq.ID, Value: "a"},
	})
	if err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if len(got) != 2 || got[0].Prompt != "Why?" || got[0].Type != QuestionText {
		t.Fatalf("expected snapshotted answers, got %+v", got)
	}

	// required text missing
	if _, err := ValidateAnswers(qs, []Answer{{QuestionID: sq.ID, Value: "a"}}); err == nil {
		t.Fatal("expected error: required text unanswered")
	}
	// single value not in options
	if _, err := ValidateAnswers(qs, []Answer{{QuestionID: tq.ID, Value: "x"}, {QuestionID: sq.ID, Value: "z"}}); err == nil {
		t.Fatal("expected error: option not allowed")
	}
	// rating out of range
	if _, err := ValidateAnswers(qs, []Answer{{QuestionID: tq.ID, Value: "x"}, {QuestionID: sq.ID, Value: "a"}, {QuestionID: rq.ID, Value: 9}}); err == nil {
		t.Fatal("expected error: rating out of range")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/backend && go test ./internal/application/model/ -run 'Survey|ValidateAnswers' -v`
Expected: FAIL (undefined: Survey/QuestionText/...).

- [ ] **Step 3: Write the model + validation**

```go
package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type QuestionType string

const (
	QuestionSingle QuestionType = "single"
	QuestionMulti  QuestionType = "multi"
	QuestionRating QuestionType = "rating"
	QuestionText   QuestionType = "text"
)

var (
	ErrInvalidSurvey      = errors.New("invalid survey")
	ErrSurveyHasResponses = errors.New("survey has responses")
	ErrSurveyClosed       = errors.New("survey closed")
	ErrResponseCompleted  = errors.New("response already completed")
)

type Survey struct {
	ID             uuid.UUID        `json:"id"`
	OrganizationID uuid.UUID        `json:"organization_id"`
	Name           string           `json:"name"`
	IsActive       bool             `json:"is_active"`
	Questions      []SurveyQuestion `json:"questions"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type SurveyQuestion struct {
	ID         uuid.UUID    `json:"id"`
	SurveyID   uuid.UUID    `json:"survey_id"`
	OrderIndex int          `json:"order_index"`
	Prompt     string       `json:"prompt"`
	Type       QuestionType `json:"type"`
	Options    []string     `json:"options"`
	RatingMax  int          `json:"rating_max"`
	Required   bool         `json:"required"`
}

type Answer struct {
	QuestionID uuid.UUID    `json:"question_id"`
	Prompt     string       `json:"prompt"`
	Type       QuestionType `json:"type"`
	Value      any          `json:"value"`
}

type SurveyResponse struct {
	ID                 uuid.UUID  `json:"id"`
	SurveyID           uuid.UUID  `json:"survey_id"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	BookingEventTypeID *uuid.UUID `json:"booking_event_type_id"`
	Token              string     `json:"-"`
	BookerEmail        string     `json:"booker_email"`
	BookerName         string     `json:"booker_name"`
	DeclineReason      string     `json:"decline_reason"`
	Status             string     `json:"status"`
	Answers            []Answer   `json:"answers"`
	CreatedAt          time.Time  `json:"created_at"`
	CompletedAt        *time.Time `json:"completed_at"`
}

func (q SurveyQuestion) validate() error {
	if q.Prompt == "" {
		return ErrInvalidSurvey
	}
	switch q.Type {
	case QuestionSingle, QuestionMulti:
		if len(q.Options) < 2 {
			return ErrInvalidSurvey
		}
	case QuestionRating:
		if q.RatingMax < 2 || q.RatingMax > 10 {
			return ErrInvalidSurvey
		}
	case QuestionText:
		// no extra fields required
	default:
		return ErrInvalidSurvey
	}
	return nil
}

func (s Survey) Validate() error {
	if s.Name == "" || len(s.Questions) == 0 {
		return ErrInvalidSurvey
	}
	for _, q := range s.Questions {
		if err := q.validate(); err != nil {
			return err
		}
	}
	return nil
}

// ValidateAnswers checks submitted answers against the survey's questions and
// returns normalized snapshot answers (prompt+type filled from the question).
func ValidateAnswers(questions []SurveyQuestion, answers []Answer) ([]Answer, error) {
	byID := map[uuid.UUID]SurveyQuestion{}
	for _, q := range questions {
		byID[q.ID] = q
	}
	given := map[uuid.UUID]Answer{}
	for _, a := range answers {
		given[a.QuestionID] = a
	}
	out := make([]Answer, 0, len(answers))
	for _, q := range questions {
		a, ok := given[q.ID]
		if !ok || isEmptyAnswer(a.Value) {
			if q.Required {
				return nil, ErrInvalidSurvey
			}
			continue
		}
		if err := validateAnswerValue(q, a.Value); err != nil {
			return nil, err
		}
		out = append(out, Answer{QuestionID: q.ID, Prompt: q.Prompt, Type: q.Type, Value: a.Value})
	}
	return out, nil
}

func isEmptyAnswer(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []string:
		return len(t) == 0
	case []any:
		return len(t) == 0
	}
	return false
}

func validateAnswerValue(q SurveyQuestion, v any) error {
	switch q.Type {
	case QuestionText:
		if _, ok := v.(string); !ok {
			return ErrInvalidSurvey
		}
	case QuestionSingle:
		s, ok := v.(string)
		if !ok || !contains(q.Options, s) {
			return ErrInvalidSurvey
		}
	case QuestionMulti:
		vals, err := toStringSlice(v)
		if err != nil {
			return err
		}
		for _, s := range vals {
			if !contains(q.Options, s) {
				return ErrInvalidSurvey
			}
		}
	case QuestionRating:
		n, ok := toInt(v)
		if !ok || n < 1 || n > q.RatingMax {
			return ErrInvalidSurvey
		}
	default:
		return ErrInvalidSurvey
	}
	return nil
}

func contains(opts []string, s string) bool {
	for _, o := range opts {
		if o == s {
			return true
		}
	}
	return false
}

func toStringSlice(v any) ([]string, error) {
	switch t := v.(type) {
	case []string:
		return t, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, ErrInvalidSurvey
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, ErrInvalidSurvey
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64: // JSON numbers decode to float64
		return int(t), true
	}
	return 0, false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/backend && go test ./internal/application/model/ -run 'Survey|ValidateAnswers' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/application/model/survey.go apps/backend/internal/application/model/survey_test.go
git commit -m "feat(surveys): domain model + question/answer validation"
```

---

## Task 3: Repository interface + Store SQL methods

**Files:**
- Modify: `apps/backend/internal/application/repository.go`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/survey_repo.go`
- Create: `apps/backend/internal/infrastructure/persistence/postgres/survey_response_repo.go`

**Interfaces:**
- Produces (added to the `Repository` interface and implemented on `*Store`):
  - `CreateSurvey(ctx, s model.Survey) (model.Survey, error)` — inserts survey + questions in a tx, returns IDs.
  - `UpdateSurvey(ctx, s model.Survey) error` — updates name/is_active and replaces all questions in a tx.
  - `GetSurvey(ctx, id uuid.UUID) (model.Survey, error)` — survey + ordered questions.
  - `ListSurveys(ctx, orgID uuid.UUID) ([]model.Survey, error)` — surveys (with questions) for an org, newest first.
  - `DeleteSurvey(ctx, id uuid.UUID) error` — deletes survey (caller guarantees no responses).
  - `CountResponses(ctx, surveyID uuid.UUID) (int, error)`.
  - `CreateSurveyResponse(ctx, r model.SurveyResponse) (model.SurveyResponse, error)`.
  - `GetSurveyResponseByToken(ctx, token string) (model.SurveyResponse, error)`.
  - `CompleteSurveyResponse(ctx, id uuid.UUID, answers []model.Answer) error`.
  - `ListSurveyResponses(ctx, surveyID uuid.UUID, f model.ResponseFilter) ([]model.SurveyResponse, error)`.
- Consumes: existing `Store` (`s.pool` pgx pool, `rowScanner` helper).

> `model.ResponseFilter{Status, Reason string; From, To *time.Time}` — add this struct to `model/survey.go` (it is a domain query filter). Add it as part of this task's first step.

- [ ] **Step 1: Add the `ResponseFilter` type**

Append to `apps/backend/internal/application/model/survey.go`:

```go
type ResponseFilter struct {
	Status string
	Reason string
	From   *time.Time
	To     *time.Time
}
```

- [ ] **Step 2: Add methods to the Repository interface**

In `apps/backend/internal/application/repository.go`, inside the `Repository` interface (next to the booking methods), add:

```go
	CreateSurvey(ctx context.Context, s model.Survey) (model.Survey, error)
	UpdateSurvey(ctx context.Context, s model.Survey) error
	GetSurvey(ctx context.Context, id uuid.UUID) (model.Survey, error)
	ListSurveys(ctx context.Context, orgID uuid.UUID) ([]model.Survey, error)
	DeleteSurvey(ctx context.Context, id uuid.UUID) error
	CountResponses(ctx context.Context, surveyID uuid.UUID) (int, error)
	CreateSurveyResponse(ctx context.Context, r model.SurveyResponse) (model.SurveyResponse, error)
	GetSurveyResponseByToken(ctx context.Context, token string) (model.SurveyResponse, error)
	CompleteSurveyResponse(ctx context.Context, id uuid.UUID, answers []model.Answer) error
	ListSurveyResponses(ctx context.Context, surveyID uuid.UUID, f model.ResponseFilter) ([]model.SurveyResponse, error)
```

- [ ] **Step 3: Implement `survey_repo.go`**

```go
package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateSurvey(ctx context.Context, sv model.Survey) (model.Survey, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Survey{}, err
	}
	defer tx.Rollback(ctx)

	if err := tx.QueryRow(ctx,
		`INSERT INTO surveys (organization_id, name, is_active) VALUES ($1,$2,$3)
		 RETURNING id, created_at, updated_at`,
		sv.OrganizationID, sv.Name, sv.IsActive).
		Scan(&sv.ID, &sv.CreatedAt, &sv.UpdatedAt); err != nil {
		return model.Survey{}, err
	}
	for i := range sv.Questions {
		q := &sv.Questions[i]
		q.OrderIndex = i
		if err := tx.QueryRow(ctx,
			`INSERT INTO survey_questions (survey_id, order_index, prompt, type, options, rating_max, required)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
			sv.ID, q.OrderIndex, q.Prompt, string(q.Type), q.Options, q.RatingMax, q.Required).
			Scan(&q.ID); err != nil {
			return model.Survey{}, err
		}
		q.SurveyID = sv.ID
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Survey{}, err
	}
	return sv, nil
}

func (s *Store) UpdateSurvey(ctx context.Context, sv model.Survey) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE surveys SET name=$2, is_active=$3, updated_at=now() WHERE id=$1`,
		sv.ID, sv.Name, sv.IsActive); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM survey_questions WHERE survey_id=$1`, sv.ID); err != nil {
		return err
	}
	for i, q := range sv.Questions {
		if _, err := tx.Exec(ctx,
			`INSERT INTO survey_questions (survey_id, order_index, prompt, type, options, rating_max, required)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			sv.ID, i, q.Prompt, string(q.Type), q.Options, q.RatingMax, q.Required); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) GetSurvey(ctx context.Context, id uuid.UUID) (model.Survey, error) {
	var sv model.Survey
	if err := s.pool.QueryRow(ctx,
		`SELECT id, organization_id, name, is_active, created_at, updated_at FROM surveys WHERE id=$1`, id).
		Scan(&sv.ID, &sv.OrganizationID, &sv.Name, &sv.IsActive, &sv.CreatedAt, &sv.UpdatedAt); err != nil {
		return model.Survey{}, err
	}
	qs, err := s.questionsFor(ctx, id)
	if err != nil {
		return model.Survey{}, err
	}
	sv.Questions = qs
	return sv, nil
}

func (s *Store) questionsFor(ctx context.Context, surveyID uuid.UUID) ([]model.SurveyQuestion, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, survey_id, order_index, prompt, type, options, rating_max, required
		 FROM survey_questions WHERE survey_id=$1 ORDER BY order_index`, surveyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SurveyQuestion{}
	for rows.Next() {
		var q model.SurveyQuestion
		var typ string
		if err := rows.Scan(&q.ID, &q.SurveyID, &q.OrderIndex, &q.Prompt, &typ, &q.Options, &q.RatingMax, &q.Required); err != nil {
			return nil, err
		}
		q.Type = model.QuestionType(typ)
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) ListSurveys(ctx context.Context, orgID uuid.UUID) ([]model.Survey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, organization_id, name, is_active, created_at, updated_at
		 FROM surveys WHERE organization_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Survey{}
	for rows.Next() {
		var sv model.Survey
		if err := rows.Scan(&sv.ID, &sv.OrganizationID, &sv.Name, &sv.IsActive, &sv.CreatedAt, &sv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, sv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		qs, err := s.questionsFor(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Questions = qs
	}
	return out, nil
}

func (s *Store) DeleteSurvey(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM surveys WHERE id=$1`, id)
	return err
}

func (s *Store) CountResponses(ctx context.Context, surveyID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM survey_responses WHERE survey_id=$1`, surveyID).Scan(&n)
	return n, err
}
```

- [ ] **Step 4: Implement `survey_response_repo.go`**

```go
package postgres

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateSurveyResponse(ctx context.Context, r model.SurveyResponse) (model.SurveyResponse, error) {
	answers, err := json.Marshal([]model.Answer{})
	if err != nil {
		return model.SurveyResponse{}, err
	}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO survey_responses
			(survey_id, organization_id, booking_event_type_id, token, booker_email, booker_name, decline_reason, status, answers)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'sent',$8)
		 RETURNING id, created_at`,
		r.SurveyID, r.OrganizationID, r.BookingEventTypeID, r.Token, r.BookerEmail, r.BookerName, r.DeclineReason, answers).
		Scan(&r.ID, &r.CreatedAt)
	r.Status = "sent"
	return r, err
}

func scanResponse(row rowScanner) (model.SurveyResponse, error) {
	var r model.SurveyResponse
	var raw []byte
	if err := row.Scan(&r.ID, &r.SurveyID, &r.OrganizationID, &r.BookingEventTypeID, &r.Token,
		&r.BookerEmail, &r.BookerName, &r.DeclineReason, &r.Status, &raw, &r.CreatedAt, &r.CompletedAt); err != nil {
		return model.SurveyResponse{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &r.Answers)
	}
	return r, nil
}

const responseCols = `id, survey_id, organization_id, booking_event_type_id, token,
	booker_email, booker_name, decline_reason, status, answers, created_at, completed_at`

func (s *Store) GetSurveyResponseByToken(ctx context.Context, token string) (model.SurveyResponse, error) {
	return scanResponse(s.pool.QueryRow(ctx, `SELECT `+responseCols+` FROM survey_responses WHERE token=$1`, token))
}

func (s *Store) CompleteSurveyResponse(ctx context.Context, id uuid.UUID, answers []model.Answer) error {
	raw, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE survey_responses SET status='completed', answers=$2, completed_at=now() WHERE id=$1`,
		id, raw)
	return err
}

func (s *Store) ListSurveyResponses(ctx context.Context, surveyID uuid.UUID, f model.ResponseFilter) ([]model.SurveyResponse, error) {
	q := `SELECT ` + responseCols + ` FROM survey_responses WHERE survey_id=$1`
	args := []any{surveyID}
	add := func(cond string, v any) {
		args = append(args, v)
		q += cond + "$" + strconv.Itoa(len(args))
	}
	if f.Status != "" {
		add(" AND status=", f.Status)
	}
	if f.Reason != "" {
		add(" AND decline_reason=", f.Reason)
	}
	if f.From != nil {
		add(" AND created_at>=", *f.From)
	}
	if f.To != nil {
		add(" AND created_at<", *f.To)
	}
	q += " ORDER BY created_at DESC"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SurveyResponse{}
	for rows.Next() {
		r, err := scanResponse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
```

- [ ] **Step 5: Build to verify Store satisfies Repository**

Run: `cd apps/backend && go build ./...`
Expected: compiles (the `Store` now implements every new `Repository` method; a missing method would fail the interface assertion used in wiring).

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/application/repository.go \
        apps/backend/internal/application/model/survey.go \
        apps/backend/internal/infrastructure/persistence/postgres/survey_repo.go \
        apps/backend/internal/infrastructure/persistence/postgres/survey_response_repo.go
git commit -m "feat(surveys): repository interface + postgres store methods"
```

---

## Task 4: Application — survey CRUD (commands + queries)

**Files:**
- Create: `apps/backend/internal/application/survey.go`
- Create: `apps/backend/internal/application/survey_test.go`

**Interfaces:**
- Consumes: `s.Store` (`Repository`), `model.Survey`, `model.ErrInvalidSurvey`, `model.ErrSurveyHasResponses`, `model.ErrForbidden`.
- Produces (methods on `*Services`):
  - `CreateSurvey(ctx, orgID uuid.UUID, in model.Survey) (model.Survey, error)`
  - `UpdateSurvey(ctx, orgID, id uuid.UUID, in model.Survey) (model.Survey, error)`
  - `DeleteSurvey(ctx, orgID, id uuid.UUID) error`
  - `GetSurvey(ctx, orgID, id uuid.UUID) (model.Survey, error)` (query)
  - `ListSurveys(ctx, orgID uuid.UUID) ([]model.Survey, error)` (query)
  - helper `requireSurveyOrg(ctx, orgID, id) (model.Survey, error)` returning `model.ErrForbidden` on org mismatch.

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

type surveyFakeStore struct {
	Repository
	survey     model.Survey
	created    model.Survey
	respCount  int
	deleted    bool
}

func (f *surveyFakeStore) CreateSurvey(_ context.Context, s model.Survey) (model.Survey, error) {
	s.ID = uuid.New()
	f.created = s
	return s, nil
}
func (f *surveyFakeStore) GetSurvey(_ context.Context, _ uuid.UUID) (model.Survey, error) {
	return f.survey, nil
}
func (f *surveyFakeStore) UpdateSurvey(_ context.Context, _ model.Survey) error { return nil }
func (f *surveyFakeStore) CountResponses(_ context.Context, _ uuid.UUID) (int, error) {
	return f.respCount, nil
}
func (f *surveyFakeStore) DeleteSurvey(_ context.Context, _ uuid.UUID) error { f.deleted = true; return nil }

func newSurveySvc(store Repository) *Services {
	return &Services{Store: store, Log: zap.NewNop()}
}

func TestCreateSurveyValidates(t *testing.T) {
	svc := newSurveySvc(&surveyFakeStore{})
	_, err := svc.CreateSurvey(context.Background(), uuid.New(), model.Survey{Name: ""})
	if !errors.Is(err, model.ErrInvalidSurvey) {
		t.Fatalf("expected ErrInvalidSurvey, got %v", err)
	}
}

func TestDeleteSurveyBlockedWhenResponsesExist(t *testing.T) {
	org := uuid.New()
	id := uuid.New()
	store := &surveyFakeStore{survey: model.Survey{ID: id, OrganizationID: org}, respCount: 3}
	svc := newSurveySvc(store)
	err := svc.DeleteSurvey(context.Background(), org, id)
	if !errors.Is(err, model.ErrSurveyHasResponses) {
		t.Fatalf("expected ErrSurveyHasResponses, got %v", err)
	}
	if store.deleted {
		t.Fatal("survey must not be deleted when responses exist")
	}
}

func TestSurveyOrgScoping(t *testing.T) {
	store := &surveyFakeStore{survey: model.Survey{ID: uuid.New(), OrganizationID: uuid.New()}}
	svc := newSurveySvc(store)
	_, err := svc.GetSurvey(context.Background(), uuid.New() /* different org */, store.survey.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/backend && go test ./internal/application/ -run 'Survey' -v`
Expected: FAIL (undefined: CreateSurvey/DeleteSurvey/GetSurvey).

- [ ] **Step 3: Implement `survey.go`**

```go
package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Services) CreateSurvey(ctx context.Context, orgID uuid.UUID, in model.Survey) (model.Survey, error) {
	in.OrganizationID = orgID
	if err := in.Validate(); err != nil {
		return model.Survey{}, err
	}
	return s.Store.CreateSurvey(ctx, in)
}

func (s *Services) requireSurveyOrg(ctx context.Context, orgID, id uuid.UUID) (model.Survey, error) {
	sv, err := s.Store.GetSurvey(ctx, id)
	if err != nil {
		return model.Survey{}, err
	}
	if sv.OrganizationID != orgID {
		return model.Survey{}, model.ErrForbidden
	}
	return sv, nil
}

func (s *Services) GetSurvey(ctx context.Context, orgID, id uuid.UUID) (model.Survey, error) {
	return s.requireSurveyOrg(ctx, orgID, id)
}

func (s *Services) ListSurveys(ctx context.Context, orgID uuid.UUID) ([]model.Survey, error) {
	return s.Store.ListSurveys(ctx, orgID)
}

func (s *Services) UpdateSurvey(ctx context.Context, orgID, id uuid.UUID, in model.Survey) (model.Survey, error) {
	if _, err := s.requireSurveyOrg(ctx, orgID, id); err != nil {
		return model.Survey{}, err
	}
	in.ID = id
	in.OrganizationID = orgID
	if err := in.Validate(); err != nil {
		return model.Survey{}, err
	}
	if err := s.Store.UpdateSurvey(ctx, in); err != nil {
		return model.Survey{}, fmt.Errorf("update survey: %w", err)
	}
	return s.Store.GetSurvey(ctx, id)
}

func (s *Services) DeleteSurvey(ctx context.Context, orgID, id uuid.UUID) error {
	if _, err := s.requireSurveyOrg(ctx, orgID, id); err != nil {
		return err
	}
	n, err := s.Store.CountResponses(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return model.ErrSurveyHasResponses
	}
	return s.Store.DeleteSurvey(ctx, id)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/backend && go test ./internal/application/ -run 'Survey' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/application/survey.go apps/backend/internal/application/survey_test.go
git commit -m "feat(surveys): application CRUD with org scoping + delete guard"
```

---

## Task 5: Application — public get/submit + decline trigger

**Files:**
- Create: `apps/backend/internal/application/survey_submit.go`
- Create: `apps/backend/internal/application/survey_submit_test.go`
- Modify: `apps/backend/internal/application/booking_submit.go`

**Interfaces:**
- Consumes: `model.ValidateAnswers`, `model.SurveyResponse`, `model.ErrSurveyClosed`, `model.ErrResponseCompleted`, `s.Store`.
- Produces (methods on `*Services`):
  - `GetPublicSurvey(ctx, token string) (model.SurveyResponse, model.Survey, error)` — returns the response + its survey; `ErrSurveyClosed` if survey inactive, `ErrResponseCompleted` if already completed, `sql.ErrNoRows` if token unknown.
  - `SubmitSurvey(ctx, token string, answers []model.Answer) error` — validates against the survey's questions and completes the response. `ErrResponseCompleted` if already done.
  - `CreatePendingResponse(ctx, et model.BookingEventType, reason string, req BookingRequest) (string, error)` — if `et.SurveyID` set and the survey is active, creates a `sent` response and returns its token; otherwise returns `""`.
  - `randomToken() (string, error)` — 32 random bytes, base64url.

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

type submitSurveyFakeStore struct {
	Repository
	resp        model.SurveyResponse
	respErr     error
	survey      model.Survey
	completedID uuid.UUID
	created     model.SurveyResponse
}

func (f *submitSurveyFakeStore) GetSurveyResponseByToken(_ context.Context, _ string) (model.SurveyResponse, error) {
	return f.resp, f.respErr
}
func (f *submitSurveyFakeStore) GetSurvey(_ context.Context, _ uuid.UUID) (model.Survey, error) {
	return f.survey, nil
}
func (f *submitSurveyFakeStore) CompleteSurveyResponse(_ context.Context, id uuid.UUID, _ []model.Answer) error {
	f.completedID = id
	return nil
}
func (f *submitSurveyFakeStore) CreateSurveyResponse(_ context.Context, r model.SurveyResponse) (model.SurveyResponse, error) {
	r.ID = uuid.New()
	f.created = r
	return r, nil
}

func TestSubmitSurveyRejectsCompleted(t *testing.T) {
	store := &submitSurveyFakeStore{resp: model.SurveyResponse{ID: uuid.New(), Status: "completed"}}
	svc := newSurveySvc(store)
	err := svc.SubmitSurvey(context.Background(), "tok", nil)
	if !errors.Is(err, model.ErrResponseCompleted) {
		t.Fatalf("expected ErrResponseCompleted, got %v", err)
	}
}

func TestSubmitSurveyValidatesAndCompletes(t *testing.T) {
	q := model.SurveyQuestion{ID: uuid.New(), Prompt: "Why?", Type: model.QuestionText, Required: true}
	respID := uuid.New()
	store := &submitSurveyFakeStore{
		resp:   model.SurveyResponse{ID: respID, Status: "sent", SurveyID: uuid.New()},
		survey: model.Survey{IsActive: true, Questions: []model.SurveyQuestion{q}},
	}
	svc := newSurveySvc(store)
	if err := svc.SubmitSurvey(context.Background(), "tok", []model.Answer{{QuestionID: q.ID, Value: "x"}}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if store.completedID != respID {
		t.Fatalf("expected completion of %v, got %v", respID, store.completedID)
	}
}

func TestCreatePendingResponseNoSurvey(t *testing.T) {
	svc := newSurveySvc(&submitSurveyFakeStore{})
	tok, err := svc.CreatePendingResponse(context.Background(), model.BookingEventType{}, "slot_taken", BookingRequest{})
	if err != nil || tok != "" {
		t.Fatalf("expected empty token + no error, got %q %v", tok, err)
	}
}

func TestCreatePendingResponseActiveSurvey(t *testing.T) {
	sid := uuid.New()
	store := &submitSurveyFakeStore{survey: model.Survey{ID: sid, IsActive: true}}
	svc := newSurveySvc(store)
	et := model.BookingEventType{OrganizationID: uuid.New(), SurveyID: &sid}
	tok, err := svc.CreatePendingResponse(context.Background(), et, "slot_taken", BookingRequest{Email: "a@b.c", Name: "Bo"})
	if err != nil || tok == "" {
		t.Fatalf("expected a token, got %q %v", tok, err)
	}
	if store.created.BookerEmail != "a@b.c" || store.created.DeclineReason != "slot_taken" {
		t.Fatalf("unexpected created response: %+v", store.created)
	}
}
```

> This task also requires `BookingEventType` to carry `SurveyID *uuid.UUID`. Add that field in Task 8 Step 1; if implementing strictly in order, add the field to `model/booking.go` now as part of Step 3 (it is needed to compile this test).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/backend && go test ./internal/application/ -run 'SubmitSurvey|CreatePendingResponse' -v`
Expected: FAIL (undefined: SubmitSurvey/CreatePendingResponse; `SurveyID` field missing).

- [ ] **Step 3: Add `SurveyID` to the model and implement `survey_submit.go`**

Add to `apps/backend/internal/application/model/booking.go` `BookingEventType` struct:

```go
	SurveyID *uuid.UUID `json:"survey_id"`
```

Create `survey_submit.go`:

```go
package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Services) GetPublicSurvey(ctx context.Context, token string) (model.SurveyResponse, model.Survey, error) {
	resp, err := s.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return model.SurveyResponse{}, model.Survey{}, err
	}
	if resp.Status == "completed" {
		return resp, model.Survey{}, model.ErrResponseCompleted
	}
	sv, err := s.Store.GetSurvey(ctx, resp.SurveyID)
	if err != nil {
		return model.SurveyResponse{}, model.Survey{}, err
	}
	if !sv.IsActive {
		return resp, sv, model.ErrSurveyClosed
	}
	return resp, sv, nil
}

func (s *Services) SubmitSurvey(ctx context.Context, token string, answers []model.Answer) error {
	resp, err := s.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return err
	}
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

// CreatePendingResponse issues a survey token for a declined booking when the
// event-type has an assigned, active survey. Returns "" when there is nothing
// to send. This is a command; it is called from the booking command path.
func (s *Services) CreatePendingResponse(ctx context.Context, et model.BookingEventType, reason string, req BookingRequest) (string, error) {
	if et.SurveyID == nil {
		return "", nil
	}
	sv, err := s.Store.GetSurvey(ctx, *et.SurveyID)
	if err != nil {
		return "", err
	}
	if !sv.IsActive {
		return "", nil
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	etID := et.ID
	_, err = s.Store.CreateSurveyResponse(ctx, model.SurveyResponse{
		SurveyID:           sv.ID,
		OrganizationID:     et.OrganizationID,
		BookingEventTypeID: &etID,
		Token:              token,
		BookerEmail:        req.Email,
		BookerName:         req.Name,
		DeclineReason:      reason,
	})
	if err != nil {
		return "", err
	}
	return token, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/backend && go test ./internal/application/ -run 'SubmitSurvey|CreatePendingResponse' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/backend/internal/application/survey_submit.go \
        apps/backend/internal/application/survey_submit_test.go \
        apps/backend/internal/application/model/booking.go
git commit -m "feat(surveys): public get/submit + decline-trigger command"
```

---

## Task 6: HTTP — admin survey CRUD handlers

**Files:**
- Create: `apps/backend/internal/delivery/http/handlers/surveys.go`

**Interfaces:**
- Consumes: `a.App` (`*application.Services`), `c.Locals("web_user")`, `X-Org-Id` header, `model.Survey`, the survey errors.
- Produces: handler methods `SurveyList`, `SurveyCreate`, `SurveyGet`, `SurveyUpdate`, `SurveyDelete` on `*API`; helper `orgIDFromHeader(c) (uuid.UUID, error)`; helper `surveyErr(log, err) error`.

- [ ] **Step 1: Implement the handlers**

```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func orgIDFromHeader(c *fiber.Ctx) (uuid.UUID, error) {
	return uuid.Parse(c.Get("X-Org-Id"))
}

func surveyErr(log *zap.Logger, err error) error {
	switch {
	case errors.Is(err, model.ErrInvalidSurvey):
		return fiber.NewError(fiber.StatusBadRequest, "invalid_survey")
	case errors.Is(err, model.ErrSurveyHasResponses):
		return fiber.NewError(fiber.StatusConflict, "survey_has_responses")
	case errors.Is(err, model.ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	case model.IsNotFound(err):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	default:
		return internalAPIError(log, "survey_failed", err)
	}
}

type surveyQuestionBody struct {
	Prompt    string   `json:"prompt"`
	Type      string   `json:"type"`
	Options   []string `json:"options"`
	RatingMax int      `json:"rating_max"`
	Required  bool     `json:"required"`
}

type surveyBody struct {
	Name      string               `json:"name"`
	IsActive  bool                 `json:"is_active"`
	Questions []surveyQuestionBody `json:"questions"`
}

func (b surveyBody) toModel() model.Survey {
	qs := make([]model.SurveyQuestion, len(b.Questions))
	for i, q := range b.Questions {
		ratingMax := q.RatingMax
		if ratingMax == 0 {
			ratingMax = 5
		}
		qs[i] = model.SurveyQuestion{
			OrderIndex: i,
			Prompt:     q.Prompt,
			Type:       model.QuestionType(q.Type),
			Options:    q.Options,
			RatingMax:  ratingMax,
			Required:   q.Required,
		}
	}
	return model.Survey{Name: b.Name, IsActive: b.IsActive, Questions: qs}
}

func (a *API) SurveyList(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	list, err := a.App.ListSurveys(c.UserContext(), orgID)
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(fiber.Map{"surveys": list})
}

func (a *API) SurveyCreate(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	var body surveyBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	sv, err := a.App.CreateSurvey(c.UserContext(), orgID, body.toModel())
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.Status(fiber.StatusCreated).JSON(sv)
}

func (a *API) SurveyGet(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	sv, err := a.App.GetSurvey(c.UserContext(), orgID, id)
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(sv)
}

func (a *API) SurveyUpdate(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	var body surveyBody
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	sv, err := a.App.UpdateSurvey(c.UserContext(), orgID, id, body.toModel())
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(sv)
}

func (a *API) SurveyDelete(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	if err := a.App.DeleteSurvey(c.UserContext(), orgID, id); err != nil {
		return surveyErr(a.Log, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

- [ ] **Step 2: Build**

Run: `cd apps/backend && go build ./...`
Expected: compiles. (`internalAPIError` already exists in the handlers package — confirm by grep; it is used by `bookingErr`.)

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/delivery/http/handlers/surveys.go
git commit -m "feat(surveys): admin CRUD HTTP handlers"
```

---

## Task 7: HTTP — admin responses list + CSV export

**Files:**
- Modify: `apps/backend/internal/delivery/http/handlers/surveys.go`
- Create: `apps/backend/internal/application/survey_csv.go` (CSV serialization — pure, testable)
- Create: `apps/backend/internal/application/survey_csv_test.go`

**Interfaces:**
- Produces:
  - `func ResponsesCSV(sv model.Survey, responses []model.SurveyResponse) []byte` (in `application`) — header row + one row per response.
  - `*Services.ListResponses(ctx, orgID, surveyID uuid.UUID, f model.ResponseFilter) (model.Survey, []model.SurveyResponse, error)` (query; enforces org scoping).
  - handlers `SurveyResponses`, `SurveyResponsesCSV` on `*API`.

- [ ] **Step 1: Write the failing CSV test**

```go
package application

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestResponsesCSV(t *testing.T) {
	q1 := model.SurveyQuestion{ID: uuid.New(), Prompt: "Why?", Type: model.QuestionText}
	q2 := model.SurveyQuestion{ID: uuid.New(), Prompt: "Pick", Type: model.QuestionMulti, Options: []string{"a", "b"}}
	sv := model.Survey{Questions: []model.SurveyQuestion{q1, q2}}
	resp := model.SurveyResponse{
		BookerName: "Bo", BookerEmail: "a@b.c", DeclineReason: "slot_taken", Status: "completed",
		Answers: []model.Answer{
			{QuestionID: q1.ID, Value: "no time"},
			{QuestionID: q2.ID, Value: []any{"a", "b"}},
		},
	}
	out := string(ResponsesCSV(sv, []model.SurveyResponse{resp}))
	if !strings.Contains(out, "Why?") || !strings.Contains(out, "Pick") {
		t.Fatalf("expected question headers, got:\n%s", out)
	}
	if !strings.Contains(out, "no time") || !strings.Contains(out, "a; b") {
		t.Fatalf("expected answer values incl multi join, got:\n%s", out)
	}
	if !strings.Contains(out, "a@b.c") || !strings.Contains(out, "slot_taken") {
		t.Fatalf("expected meta columns, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/backend && go test ./internal/application/ -run 'ResponsesCSV' -v`
Expected: FAIL (undefined: ResponsesCSV).

- [ ] **Step 3: Implement `survey_csv.go`**

```go
package application

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func ResponsesCSV(sv model.Survey, responses []model.SurveyResponse) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	header := []string{"created_at", "name", "email", "reason", "status"}
	for _, q := range sv.Questions {
		header = append(header, q.Prompt)
	}
	_ = w.Write(header)

	for _, r := range responses {
		byQ := map[string]string{}
		for _, a := range r.Answers {
			byQ[a.QuestionID.String()] = answerToString(a.Value)
		}
		row := []string{
			r.CreatedAt.Format("2006-01-02 15:04"),
			r.BookerName, r.BookerEmail, r.DeclineReason, r.Status,
		}
		for _, q := range sv.Questions {
			row = append(row, byQ[q.ID.String()])
		}
		_ = w.Write(row)
	}
	w.Flush()
	return buf.Bytes()
}

func answerToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, "; ")
	case []any:
		parts := make([]string, len(t))
		for i, e := range t {
			parts[i] = fmt.Sprintf("%v", e)
		}
		return strings.Join(parts, "; ")
	case float64:
		return fmt.Sprintf("%g", t)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/backend && go test ./internal/application/ -run 'ResponsesCSV' -v`
Expected: PASS.

- [ ] **Step 5: Add the `ListResponses` query to `survey.go`**

Append to `apps/backend/internal/application/survey.go`:

```go
func (s *Services) ListResponses(ctx context.Context, orgID, surveyID uuid.UUID, f model.ResponseFilter) (model.Survey, []model.SurveyResponse, error) {
	sv, err := s.requireSurveyOrg(ctx, orgID, surveyID)
	if err != nil {
		return model.Survey{}, nil, err
	}
	rs, err := s.Store.ListSurveyResponses(ctx, surveyID, f)
	if err != nil {
		return model.Survey{}, nil, err
	}
	return sv, rs, nil
}
```

- [ ] **Step 6: Add response handlers to `surveys.go`**

```go
func parseResponseFilter(c *fiber.Ctx) model.ResponseFilter {
	f := model.ResponseFilter{Status: c.Query("status"), Reason: c.Query("reason")}
	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			f.From = &t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(24 * time.Hour)
			f.To = &end
		}
	}
	return f
}

func (a *API) SurveyResponses(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	sv, rs, err := a.App.ListResponses(c.UserContext(), orgID, id, parseResponseFilter(c))
	if err != nil {
		return surveyErr(a.Log, err)
	}
	return c.JSON(fiber.Map{"survey": sv, "responses": rs})
}

func (a *API) SurveyResponsesCSV(c *fiber.Ctx) error {
	orgID, err := orgIDFromHeader(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing_or_invalid_org_id")
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_id")
	}
	sv, rs, err := a.App.ListResponses(c.UserContext(), orgID, id, parseResponseFilter(c))
	if err != nil {
		return surveyErr(a.Log, err)
	}
	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="survey-responses.csv"`)
	return c.Send(application.ResponsesCSV(sv, rs))
}
```

Add imports to `surveys.go`: `"time"` and the application package alias already imported in the handlers package (it is — `bookingErr` uses `application`). If not aliased, add `"github.com/luckyrogue/lead-cat/internal/application"`.

- [ ] **Step 7: Build + run application tests**

Run: `cd apps/backend && go build ./... && go test ./internal/application/ -v`
Expected: compiles, all application tests PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/backend/internal/application/survey_csv.go \
        apps/backend/internal/application/survey_csv_test.go \
        apps/backend/internal/application/survey.go \
        apps/backend/internal/delivery/http/handlers/surveys.go
git commit -m "feat(surveys): responses listing + CSV export"
```

---

## Task 8: Event-type assignment (`survey_id`) end-to-end

**Files:**
- Modify: `apps/backend/internal/application/booking.go` (`EventTypeInput`, update path)
- Modify: `apps/backend/internal/infrastructure/persistence/postgres/booking_repo.go` (read/write `survey_id`)
- Modify: `apps/backend/internal/delivery/http/handlers/booking.go` (`eventTypeBody.SurveyID`)

**Interfaces:**
- Consumes: `model.BookingEventType.SurveyID` (added in Task 5).
- Produces: `EventTypeInput.SurveyID *uuid.UUID`; the PATCH handler reads `survey_id`; the repo persists and scans it.

- [ ] **Step 1: Extend `EventTypeInput` and the update path**

In `apps/backend/internal/application/booking.go`, add `SurveyID *uuid.UUID` to `EventTypeInput`, and in the update method map it onto the `model.BookingEventType` before `UpdateBookingEventType`. (Find the existing `UpdateMyEventType`/equivalent; set `et.SurveyID = in.SurveyID`.)

- [ ] **Step 2: Persist `survey_id` in the repo**

In `booking_repo.go`:
- Add `survey_id` to `bookingCols` and to `scanBookingRow` (scan into `&et.SurveyID`, a `*uuid.UUID`).
- Add `survey_id=$N` to the `UpdateBookingEventType` SET list with `et.SurveyID`.
- Add `survey_id` to the `CreateBookingEventType` column list only if create should accept it (optional; assignment is via PATCH, so create may leave it NULL — leave create unchanged, just ensure SELECT/scan and UPDATE include it).

- [ ] **Step 3: Accept `survey_id` in the PATCH body**

In `handlers/booking.go`, add to `eventTypeBody`:

```go
	SurveyID *string `json:"survey_id"`
```

and in `toInput()` parse it:

```go
	var surveyID *uuid.UUID
	if b.SurveyID != nil && *b.SurveyID != "" {
		if id, err := uuid.Parse(*b.SurveyID); err == nil {
			surveyID = &id
		}
	}
	// set on the returned EventTypeInput: SurveyID: surveyID
```

> An empty string clears the assignment (NULL); a valid UUID sets it. The update path must distinguish "field present" — since assignment is the only writer, treat `nil`/empty as "no survey".

- [ ] **Step 4: Write/extend a handler test for assignment**

Add to `apps/backend/internal/delivery/http/handlers/surveys_test.go` (created in Task 9) or `booking_test.go`: PATCH an event-type with `survey_id` and assert the stored event-type carries it. (Concrete code lives with Task 9's handler-test harness; if implementing now, mirror the existing `booking_test.go` request helper.)

- [ ] **Step 5: Build + test**

Run: `cd apps/backend && go build ./... && go test ./internal/... `
Expected: compiles, green.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/application/booking.go \
        apps/backend/internal/infrastructure/persistence/postgres/booking_repo.go \
        apps/backend/internal/delivery/http/handlers/booking.go
git commit -m "feat(surveys): assign survey to booking event-type via PATCH"
```

---

## Task 9: HTTP — public survey + decline-response wiring + routing

**Files:**
- Create: `apps/backend/internal/delivery/http/handlers/surveys_public.go`
- Create: `apps/backend/internal/delivery/http/handlers/surveys_test.go`
- Modify: `apps/backend/internal/delivery/http/handlers/public_booking_submit.go`
- Modify: `apps/backend/internal/delivery/http/app.go`

**Interfaces:**
- Consumes: `a.App.GetPublicSurvey`, `a.App.SubmitSurvey`, `a.App.CreatePendingResponse`, `a.App.SubmitBooking`.
- Produces: handlers `PublicSurveyGet`, `PublicSurveySubmit`; the booking-submit handler now returns a JSON body with `survey_token` on decline; survey routes registered.

- [ ] **Step 1: Implement `surveys_public.go`**

```go
package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type publicQuestion struct {
	ID        string   `json:"id"`
	Prompt    string   `json:"prompt"`
	Type      string   `json:"type"`
	Options   []string `json:"options"`
	RatingMax int      `json:"rating_max"`
	Required  bool     `json:"required"`
}

func (a *API) PublicSurveyGet(c *fiber.Ctx) error {
	token := c.Params("token")
	resp, sv, err := a.App.GetPublicSurvey(c.UserContext(), token)
	switch {
	case errors.Is(err, model.ErrResponseCompleted):
		return fiber.NewError(fiber.StatusConflict, "already_completed")
	case errors.Is(err, model.ErrSurveyClosed):
		return fiber.NewError(fiber.StatusNotFound, "survey_closed")
	case model.IsNotFound(err):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	case err != nil:
		return internalAPIError(a.Log, "survey_get_failed", err)
	}
	qs := make([]publicQuestion, len(sv.Questions))
	for i, q := range sv.Questions {
		qs[i] = publicQuestion{
			ID: q.ID.String(), Prompt: q.Prompt, Type: string(q.Type),
			Options: q.Options, RatingMax: q.RatingMax, Required: q.Required,
		}
	}
	return c.JSON(fiber.Map{
		"survey_name": sv.Name,
		"questions":   qs,
		"booker_name": resp.BookerName,
	})
}

func (a *API) PublicSurveySubmit(c *fiber.Ctx) error {
	token := c.Params("token")
	var body struct {
		Answers []struct {
			QuestionID string `json:"question_id"`
			Value      any    `json:"value"`
		} `json:"answers"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid_body")
	}
	answers := make([]model.Answer, 0, len(body.Answers))
	for _, a := range body.Answers {
		id, err := uuidParse(a.QuestionID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid_question_id")
		}
		answers = append(answers, model.Answer{QuestionID: id, Value: a.Value})
	}
	err := a.App.SubmitSurvey(c.UserContext(), token, answers)
	switch {
	case errors.Is(err, model.ErrResponseCompleted):
		return fiber.NewError(fiber.StatusConflict, "already_completed")
	case errors.Is(err, model.ErrSurveyClosed):
		return fiber.NewError(fiber.StatusNotFound, "survey_closed")
	case errors.Is(err, model.ErrInvalidSurvey):
		return fiber.NewError(fiber.StatusBadRequest, "invalid_answers")
	case model.IsNotFound(err):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	case err != nil:
		return internalAPIError(a.Log, "survey_submit_failed", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
```

> `uuidParse` is `uuid.Parse`; import `"github.com/google/uuid"` and call `uuid.Parse`. (Replace `uuidParse` with `uuid.Parse` and add the import.)

- [ ] **Step 2: Make the booking decline response carry `survey_token`**

Replace the `ErrSlotTaken` / `ErrInvalidBooking` cases in `public_booking_submit.go` so they create a pending response and return a JSON body instead of `fiber.NewError`:

```go
	conf, err := a.App.SubmitBooking(c.UserContext(), slug, application.BookingRequest{Name: body.Name, Email: body.Email, Start: start, Language: body.Language})
	if err != nil {
		switch {
		case model.IsNotFound(err):
			return fiber.NewError(fiber.StatusNotFound, "not_found")
		case errors.Is(err, model.ErrSlotTaken):
			return a.declineWithSurvey(c, slug, "slot_taken", fiber.StatusConflict,
				application.BookingRequest{Name: body.Name, Email: body.Email, Start: start, Language: body.Language})
		case errors.Is(err, model.ErrInvalidBooking):
			return a.declineWithSurvey(c, slug, "invalid_booking", fiber.StatusBadRequest,
				application.BookingRequest{Name: body.Name, Email: body.Email, Start: start, Language: body.Language})
		default:
			return fiber.NewError(fiber.StatusInternalServerError, "booking_failed")
		}
	}
	return c.JSON(conf)
}

func (a *API) declineWithSurvey(c *fiber.Ctx, slug, reason string, status int, req application.BookingRequest) error {
	body := fiber.Map{"error": "error", "message": reason}
	et, err := a.App.GetBookingEventTypeBySlugPublic(c.UserContext(), slug)
	if err == nil {
		if token, terr := a.App.CreatePendingResponse(c.UserContext(), et, reason, req); terr == nil && token != "" {
			body["survey_token"] = token
		}
	}
	return c.Status(status).JSON(body)
}
```

> Add a thin query `GetBookingEventTypeBySlugPublic(ctx, slug) (model.BookingEventType, error)` to `Services` (wraps `s.Store.GetBookingEventTypeBySlug`) so the handler does not touch the Store directly. Add it to `application/booking.go`.

- [ ] **Step 3: Write the handler test (`surveys_test.go`)**

```go
package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// Mirror the existing booking_test.go harness: build an *API with a fake App,
// register routes, and fire httptest requests. Assert:
//  - GET /api/survey/<bad> → 404
//  - POST /api/survey/<completed-token> → 409 already_completed
//  - declined booking with assigned active survey → body has survey_token
func TestPublicSurveyNotFound(t *testing.T) {
	app := fiber.New()
	api := newTestAPI(t) // helper from booking_test.go style; constructs API with fakes
	app.Get("/api/survey/:token", api.PublicSurveyGet)

	req := httptest.NewRequest("GET", "/api/survey/nope", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	_ = strings.TrimSpace
}
```

> Use the same fake-`Services`/`Store` wiring the existing `booking_test.go` uses (replicate its `newTestAPI` helper or extend it). The key assertions: 404 for unknown token, 409 for completed, and that a decline with an active assigned survey yields `survey_token` in the body. Fill these in following `booking_test.go`'s exact construction.

- [ ] **Step 4: Register routes in `app.go`**

After the `booking` group (around line 162), add:

```go
	surveys := app.Group("/api/surveys", webAuth.Middleware)
	surveys.Get("", api.SurveyList)
	surveys.Post("", api.SurveyCreate)
	surveys.Get("/:id", api.SurveyGet)
	surveys.Patch("/:id", api.SurveyUpdate)
	surveys.Delete("/:id", api.SurveyDelete)
	surveys.Get("/:id/responses", api.SurveyResponses)
	surveys.Get("/:id/responses.csv", api.SurveyResponsesCSV)

	// Public survey delivery (unauthenticated, token-based, rate-limited).
	app.Get("/api/survey/:token", api.PublicSurveyGet)
	app.Post("/api/survey/:token", rateLimit(20, time.Minute, "survey_submit"), api.PublicSurveySubmit)
```

> Confirm the public booking submit route already uses `rateLimit`; mirror its limiter signature.

- [ ] **Step 5: Build + full backend test**

Run: `cd apps/backend && go build ./... && go test ./...`
Expected: compiles, all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/delivery/http/handlers/surveys_public.go \
        apps/backend/internal/delivery/http/handlers/surveys_test.go \
        apps/backend/internal/delivery/http/handlers/public_booking_submit.go \
        apps/backend/internal/application/booking.go \
        apps/backend/internal/delivery/http/app.go
git commit -m "feat(surveys): public delivery endpoints + decline-token wiring + routes"
```

---

## Task 10: OpenAPI regen + final verification

**Files:**
- Modify: `apps/backend/openapi/openapi.json` (regenerated)

**Interfaces:** none new — this surfaces the routes to the typed client used by the admin/public frontend plans.

- [ ] **Step 1: Annotate routes for OpenAPI**

Follow the existing pattern used by booking endpoints (the project generates `openapi.json` from route annotations or a generator — check how `/api/booking/event-types` appears in `openapi.json` and replicate for `/api/surveys*` and `/api/survey/:token`). If the spec is hand-maintained, add the survey paths and schemas (`Survey`, `SurveyQuestion`, `SurveyResponse`, request bodies) mirroring booking entries.

- [ ] **Step 2: Regenerate the typed client**

Run: `cd /Users/temirlan/Workspace/in-house/lead-cat && pnpm openapi:generate`
Expected: `packages/api-client/src/generated/schema.ts` updates with survey paths.

- [ ] **Step 3: Full verification**

Run:
```bash
cd apps/backend && gofmt -l . && go vet ./... && go test ./...
cd /Users/temirlan/Workspace/in-house/lead-cat && pnpm typecheck
```
Expected: gofmt prints nothing, vet clean, all Go tests PASS, monorepo typecheck green.

- [ ] **Step 4: Commit**

```bash
git add apps/backend/openapi/openapi.json packages/api-client/src/generated/schema.ts
git commit -m "feat(surveys): regenerate OpenAPI + typed client for survey endpoints"
```

---

## Self-review notes (addressed)

- **Spec coverage:** library CRUD (Tasks 3–4, 6), question types + validation (Task 2), assignment to event-type (Task 8), public delivery by token (Tasks 5, 9), decline trigger + `survey_token` in body (Tasks 5, 9), responses + CSV (Task 7), org scoping/authz (Tasks 4, 6, 7), delete-with-responses block (Tasks 2, 4), rate limit on submit (Task 9). Bot delivery / operator notifications / landing / admin UI are out of this backend plan (separate plans).
- **Type consistency:** `SurveyID *uuid.UUID` is introduced in Task 5 Step 3 and consumed in Tasks 8–9; `model.ResponseFilter` defined in Task 3 Step 1 and used in Tasks 3/7; `ResponsesCSV(model.Survey, []model.SurveyResponse) []byte` defined in Task 7 and called in the CSV handler.
- **Known follow-ups (confirm against the codebase during implementation):** confirm the exact `internalAPIError` and `rateLimit` signatures by grep before use; replicate `booking_test.go`'s API-construction harness for `surveys_test.go` (the plan references `newTestAPI` as a stand-in for whatever that file already uses); verify the project's goose migration command (`make migrate` vs a direct goose invocation).
