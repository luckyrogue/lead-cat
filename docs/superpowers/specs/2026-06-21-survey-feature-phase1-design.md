# Survey-on-decline — Phase 1 (web) design

**Date:** 2026-06-21
**Status:** Approved design, ready for implementation plan
**Scope:** Phase 1 of a multi-phase feature. This spec covers **web only**.

## Problem & goal

When a public booking attempt fails because no time works ("отказ по времени" —
slot already taken or requested time outside availability), the lead today hits a
dead end. We want to turn that moment into feedback/lead-capture: the org operator
builds a survey from arbitrary questions and assigns it to a booking service; when
a booking is declined, the lead is offered that survey. The operator controls
whether a survey is sent (by assigning one and keeping it active) and reads the
answers in the admin.

## Phasing (whole feature)

This feature is too large for one plan. It is split by delivery surface:

- **Phase 1 (this spec) — web.** Survey library + builder in admin, assignment to
  booking event-types, public survey page after a declined booking, responses
  table + CSV export.
- **Phase 2 — bot.** Conversational survey delivery in Telegram for the
  "participant declined a meeting" trigger (inline buttons for choice/rating,
  message capture for free text), org-level assignment for meetings, a "decline
  meeting" action. Reuses Phase 1's model and builder.
- **Phase 3 — operator notifications.** Per-response push to the org notify-chat,
  and any response aggregation.

Each phase is its own spec → plan → implementation cycle.

## Phase 1 scope

In: survey library (per org), survey builder (4 question types), assignment to a
booking event-type, public web survey page, responses table with filters, CSV
export, landing-page advertisement of the feature.

Out (later phases): Telegram bot delivery, meeting-decline trigger, operator
notifications, response analytics/aggregation, deduplication of repeat sends.

## End-to-end flow (web)

```
1. Lead on /book/acme-30min picks a time → POST /api/book/:slug
2. Backend: booking fails (409 slot_taken | 400 invalid_booking)
       └─ does the event-type have an assigned, active survey?
            yes → create survey_responses row (status=sent, random token)
                  respond 409/400 with { survey_token } in the error body
            no  → unchanged behavior (plain error)
3. Booking page: if the error body has survey_token →
       show CTA block "Не нашли время? Пройдите короткий опрос →"
       linking to /survey/:token
4. /survey/:token (public route in the admin SPA):
       GET  /api/survey/:token → questions + context (service title, org name)
       POST /api/survey/:token → answers saved, status=completed
5. Operator in admin → "Опросы" → responses table + filters + CSV export
```

Design decisions:
- Survey page is a **public route in the admin SPA** (`survey.$token.tsx`), like
  `book.$slug.tsx` — not a backend-rendered page, not the mini-app.
- On decline we **create a `survey_responses` row with `status=sent`** and a random
  token. This yields a sent-vs-completed conversion metric. Trade-off: repeat
  declines by the same lead create multiple `sent` rows (known limitation below).
- Assignment is stored as a nullable `survey_id` column on `booking_event_types`
  (assigned-or-not = the on/off toggle for the web path). A general assignment
  table is deferred to Phase 2 (meeting-level).

## Data model

Three new tables + one column. Goose migrations, following the
`booking_event_types` convention.

```sql
-- 1) Org survey library
CREATE TABLE surveys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX surveys_org_idx ON surveys (organization_id);

-- 2) Ordered questions
CREATE TABLE survey_questions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    survey_id   UUID NOT NULL REFERENCES surveys(id) ON DELETE CASCADE,
    order_index INT  NOT NULL,
    prompt      TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('single','multi','rating','text')),
    options     TEXT[] NOT NULL DEFAULT '{}',   -- single/multi
    rating_max  INT  NOT NULL DEFAULT 5,        -- rating
    required    BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX survey_questions_survey_idx ON survey_questions (survey_id);

-- 3) Responses (created on decline, filled on submit)
CREATE TABLE survey_responses (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- NO ACTION (default), NOT cascade/restrict: a direct survey delete with
    -- responses is blocked in the application (409); deleting the org still
    -- cascades cleanly via organization_id because NO ACTION is checked at the
    -- end of the statement, after the cascade has removed the responses.
    survey_id             UUID NOT NULL REFERENCES surveys(id),
    organization_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    booking_event_type_id UUID REFERENCES booking_event_types(id) ON DELETE SET NULL,
    token                 TEXT NOT NULL UNIQUE,
    booker_email          TEXT NOT NULL DEFAULT '',
    booker_name           TEXT NOT NULL DEFAULT '',
    decline_reason        TEXT NOT NULL DEFAULT '',  -- 'slot_taken' | 'invalid_booking'
    status                TEXT NOT NULL DEFAULT 'sent'
                          CHECK (status IN ('sent','completed')),
    answers               JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at          TIMESTAMPTZ
);
CREATE INDEX survey_responses_survey_idx ON survey_responses (survey_id);
CREATE INDEX survey_responses_org_idx ON survey_responses (organization_id);

-- 4) Assignment to a booking service
ALTER TABLE booking_event_types
    ADD COLUMN survey_id UUID REFERENCES surveys(id) ON DELETE SET NULL;
```

Decisions:
- **Answer snapshot.** `answers` is an array of self-contained objects
  `{ question_id, prompt, type, value }`, not `{question_id: value}`. Responses stay
  readable in the table/CSV even if a question is later edited or deleted; responses
  are historical records.
- **`value` by type:** `text` → string; `single` → selected option (string);
  `multi` → array of strings; `rating` → number.
- **`organization_id` denormalized onto `survey_responses`** (derivable via survey)
  for simple scoping and fast filtering.
- **Two off-switches:** a survey can be globally paused via `is_active` without
  removing the per-event-type assignment. Sent only if assigned **and** active.
- **A survey with responses cannot be hard-deleted** (history preservation). This
  is enforced in the application — `DeleteSurvey` checks for responses and returns
  409; the operator deactivates instead. The DB FK uses default `NO ACTION` (not
  `RESTRICT`, not `CASCADE`): `NO ACTION` still blocks a direct survey delete that
  would orphan responses, while letting an org delete cascade cleanly (the check
  runs at end-of-statement, after the org cascade has already removed the
  responses). `CASCADE` was rejected because a bug bypassing the app guard would
  silently delete response history; `RESTRICT` was rejected because its immediate
  check breaks org deletion.

## Backend

Layers per Clean Architecture: `model` → `application` (CQRS) → `infrastructure` /
`delivery`.

**Domain (`internal/application/model/`):** `survey.go` — `Survey`,
`SurveyQuestion`, `QuestionType` enum; `survey_response.go` — `SurveyResponse`,
`Answer{QuestionID, Prompt, Type, Value}`.

**Repos (`internal/infrastructure/persistence/postgres/`):**
- `survey_repo.go` — survey CRUD + questions (create/replace questions in a
  transaction), `GetWithQuestions`, `ListByOrg`.
- `survey_response_repo.go` — `Create`, `GetByToken`, `Complete`,
  `ListBySurvey(filters)`.

**Application (CQRS):**
- Commands: `CreateSurvey`, `UpdateSurvey` (name/active + full question replace),
  `DeleteSurvey`, `AssignSurveyToEventType`, `SubmitSurveyResponse`,
  `CreatePendingResponse` (called from the booking command on decline).
- Queries: `ListSurveys`, `GetSurvey`, `GetPublicSurveyByToken`, `ListResponses`,
  `ExportResponsesCSV`.
- Validation in `domain`/`application`: question type ↔ fields (`single`/`multi`
  need ≥2 options; `rating` `rating_max` 2..10; `text` no options); on submit:
  required answered, choice values ∈ current question options, rating in range,
  answer type matches question type.

**HTTP endpoints.**

Admin (cookie session, org-scoped via `X-Org-Id`):

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/surveys` | list org surveys |
| POST | `/api/surveys` | create (name + questions) |
| GET | `/api/surveys/:id` | survey with questions |
| PATCH | `/api/surveys/:id` | name/active + replace questions |
| DELETE | `/api/surveys/:id` | delete (409 if it has responses) |
| GET | `/api/surveys/:id/responses` | responses (filters: status, reason, date) |
| GET | `/api/surveys/:id/responses.csv` | CSV export |

Assignment extends the existing `PATCH /api/booking/event-types/:id` with a
`survey_id` field (no separate endpoint).

Public (unauthenticated):

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/survey/:token` | questions + context; 404 if token invalid or survey inactive |
| POST | `/api/survey/:token` | submit answers → `status=completed`; repeat → 409 |

**Decline trigger (`application/booking_submit.go`):** after submit returns
`ErrSlotTaken` / `ErrInvalidBooking`, if the event-type has a `survey_id` and the
survey is `is_active`, call `CreatePendingResponse` (random token, email/name/reason
from the request) and bubble the token up. `public_booking_submit.go` puts
`survey_token` into the 409/400 error body.

**Token:** cryptographically random (`crypto/rand`, base64url, ~32 bytes) — an
unguessable link.

Decisions:
- **CQRS split** — `CreatePendingResponse` is a command, invoked from the booking
  command, never from a query.
- **Question update = full replace** (delete+insert in a transaction) on survey
  PATCH; responses are untouched (they carry snapshots).
- **Assignment via the existing event-type PATCH**, no new endpoint.
- **Response row created synchronously** in the booking flow (a plain INSERT); asynq
  is reserved for bot delivery / notifications in later phases.

## Admin UI

Feature-Sliced Design, following `event-type-dialog` (RHF + Zod) and
`entities/booking-event-type`.

**New entity `entities/survey/`** — `api.ts`, `queries.ts`, `types.ts`:
`surveyKeys.list(orgId)`, hooks `useSurveys/useSurvey/useCreateSurvey/`
`useUpdateSurvey/useDeleteSurvey/useResponses`.

**New feature slice `features/surveys/`:**
1. **Survey list** (`pages/surveys-page.tsx`) — cards/table of org surveys: name,
   question count, active badge, "N responses", Edit/Delete/Responses. "Create
   survey" button.
2. **Builder** (`components/survey-dialog.tsx`) — dialog with survey fields (`name`,
   `is_active` switch) and a **question editor** (`components/question-editor.tsx`):
   list with ↑/↓ buttons (ordering without a drag library), "Add question",
   "Remove"; per question `prompt`, `type` select, `required` switch, and
   conditional fields — options editor (add/remove rows) for single/multi,
   `rating_max` select (2–10) for rating. Zod validation mirrors the backend.
3. **Responses** (`pages/responses-page.tsx`) — table: date, lead name/email,
   service, decline reason, status (sent/completed), expanded answers. Filters:
   status, reason, date range. "Export CSV" button.
4. **Assignment** — in `event-type-dialog.tsx`, a "Survey on decline" select
   (active org surveys + "None"), written to `survey_id` via the existing
   event-type PATCH.
5. **Navigation** — "Опросы" item in the sidebar (`_app.tsx`), routes
   `_app.surveys._index.tsx` and `_app.surveys.$id.responses.tsx`.

**api-client:** new endpoints land in OpenAPI → regenerate `@leadcat/api-client`
(`pnpm openapi:generate`); types come from there.

Decisions: question ordering via ↑/↓ buttons (no DnD lib); single dialog with an
inline question editor (not a wizard); CSV downloaded via a direct authenticated
GET (backend sends `text/csv` + `Content-Disposition`).

## Public survey page + booking wiring

**Public route `routes/survey.$token.tsx`** (admin SPA, like `book.$slug.tsx`),
wrapped in `AuthLocaleShell` — public and localized, with the language switcher.

- `GET /api/survey/:token` → render questions by type: `single` → radio group;
  `multi` → checkboxes; `rating` → 1…N buttons (★); `text` → textarea. Client-side
  `required` check; submit → `POST /api/survey/:token`.
- **Page states:** loading · survey form · "unavailable" (bad token or inactive
  survey) · "already completed" (repeat submit, 409) · "thank you" (success).

**Booking wiring (`book.$slug.form.tsx`):** today the form shows static
status-based messages. Add: if the 409/400 body has `survey_token`, render a CTA
block below the decline message — "Не нашли удобное время? Пройдите короткий опрос
→" linking to `/survey/:token`. No token → behavior unchanged.

**i18n:** our chrome (buttons, labels, statuses, errors, CTA) goes into the admin
`shared/i18n` dictionaries `en/ru/kk`. The **operator's survey content** (question
prompts and options) is their own text in any language — rendered as-is, never run
through our i18n. Rendering uses plain React text nodes (auto-escaped), no
`dangerouslySetInnerHTML`.

Decisions: survey lives on its own `/survey/:token` page (not inline in the booking
form, so the link can be reused/emailed later); operator content not localized;
`rating` rendered as number/star buttons 1…rating_max.

## Landing

The landing (`apps/landing`) is localized (en/ru/kk; sections `features`,
`howItWorks`, `showcase`…). Add a **single feature card** to the existing
`features` section advertising the feature (e.g. "Surveys on decline: don't lose a
lead when the time doesn't fit — collect feedback automatically"), in **all three
locales**, matching the existing dictionary structure and cinematic style. The card
is fitted into the existing grid so the layout doesn't break. Marketing copy **is**
localized to all three languages (unlike operator survey content).

## Error handling & edge cases

- **Delete survey with responses → blocked.** `DeleteSurvey` checks for responses
  and returns 409; UI suggests "deactivate instead". (DB FK is `NO ACTION` — see the
  data-model note on why not `RESTRICT`/`CASCADE`.)
- **Bad/missing token** → `GET /api/survey/:token` 404 → "survey unavailable" page.
- **Survey deactivated after token issued** (`is_active=false`) → treated as
  unavailable; `is_active` checked at GET time.
- **Repeat submit** (`status=completed`) → 409 → "already completed" page.
- **Submit validation** (backend): required answered; `single`/`multi` values ∈
  current question `options`; `rating` ∈ 1..rating_max; answer type matches question
  type. Violation → 400 naming the question.
- **Authorization:** every admin endpoint checks
  `survey.organization_id == X-Org-Id` (no reading/editing other orgs' surveys or
  responses). Public endpoints are token-only, no org context.
- **CSV format:** one row per response; columns = meta (date, name, email, service,
  reason, status) + one column per question (survey order); `multi` joined with
  "; ". Headers taken from the survey's current questions.
- **Rate limit** on public submit — reuse the existing middleware (as booking does).

### Known limitations (Phase 1)
- Repeat booking declines by the same lead create multiple `sent` rows. Dedup by
  `(survey_id, email, event_type)` within a window is deferred.

## Testing

**Backend** (per `booking_test.go` + domain unit tests):
- Domain/application: question validation (type ↔ fields); submit validation
  (required, option membership, rating range, type match); CSV serialization.
- Handler tests: admin CRUD + org scoping (other org → 403/404); public token flow
  (valid · bad token → 404 · inactive survey → closed · repeat submit → 409);
  decline trigger (declined booking with assigned active survey → `sent` row +
  `survey_token` in body; no survey → no token; inactive survey → no token).
- Repo tests if a test-DB harness exists (match the current repository pattern).

**Frontend** (vitest, per convention): pure helpers — builder Zod schema (mirrors
backend validation), public-form `required` check, answer shaping by type.

**Landing:** en/ru/kk parity is enforced by the `LandingDict` type (typecheck fails
if a card key is missing in a locale); no separate test.
