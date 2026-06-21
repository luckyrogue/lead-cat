# Survey Phase 2a — bot delivery (meeting decline) design

**Date:** 2026-06-22
**Status:** Approved design, ready for implementation plan
**Scope:** Phase 2a of the survey feature — the **bot** vertical. Mini-app (2b) is a separate spec.

## Problem & goal

Phase 1 delivers a survey on a public **web booking** decline. Phase 2 extends surveys to
**meeting participants**: when a participant declines a meeting, mark them declined (RSVP),
notify the organizer, and deliver the org's survey conversationally **in the Telegram bot**.
This increment (2a) builds the whole bot vertical + the shared RSVP core; Phase 2b adds the
mini-app decline button and mini-app survey rendering on top of the same core.

## Phasing

- **Phase 1 (done):** web survey on booking decline.
- **Phase 2a (this spec) — bot:** RSVP decline via a bot inline button → participant marked
  declined + organizer notified + conversational survey delivery in the bot; org-level + per-meeting
  survey assignment; admin assignment UI; responses table shows the new source.
- **Phase 2b — mini-app:** decline button on the meeting card → same RSVP core + mini-app survey
  rendering (reuses Phase 1 web render).
- **Phase 3:** operator notifications on response.

Each phase is its own spec → plan → implementation cycle.

## Approved decisions (from brainstorming)

- Decline happens via a bot inline button; delivery is conversational in the bot (2a).
- Decline is **RSVP**: participant status becomes `declined`, the organizer is notified.
- Both RSVP buttons: `[✓ Буду]` (accept) and `[❌ Не смогу]` (decline). Statuses:
  `invited` (default) | `accepted` | `declined`.
- Buttons appear on the **meeting-created** notification AND on **reminders**.
- After decline: the survey is **offered with a skip** (`[Начать]` / `[Пропустить]`). The decline
  reason is captured by the operator's survey questions — no separate hardcoded reason list.
- Assignment: **org-level default + per-meeting override**; resolve `meeting ?? org`.
- Architecture: **reuse + extend Phase 1** (one `survey_responses` table with a `source`
  discriminator; reuse `model.Survey`, `ValidateAnswers`, the survey library, the admin responses
  view). New code only where genuinely new: the bot delivery FSM and the RSVP core.

## End-to-end flow

```
1. Meeting created / reminder fires → notifier sends each bot-linked participant a personal
   notification WITH inline keyboard: [✓ Буду] [❌ Не смогу]
   (callback rsvp:accept:<meetingId> / rsvp:decline:<meetingId>)
2. Participant taps [❌ Не смогу]:
       ├─ participant.rsvp_status = declined
       ├─ organizer notified in the bot ("<participant> can't make «<meeting>»")
       └─ resolve survey = COALESCE(meeting.survey_on_decline_id, org.survey_on_decline_id)
            active survey? yes → create survey_responses (source=meeting_decline, meeting_id,
                                 participant_telegram_id, decline_reason='meeting_decline',
                                 status=sent) → bot: "Marked. Take a short survey?" [Начать][Пропустить]
                            no → bot: "Marked, the organizer was notified." (end)
3. [Начать] (survey:start:<responseId>) → botsurvey FSM walks the questions:
       single/multi/rating → inline buttons; text → bot waits for the next message (OnText)
       progress in Redis {response_id, q_index, answers}
4. Last question → SubmitSurveyResponse(responseId, answers) (Phase-1 validation) → status=completed
       → bot: "Thanks!"  |  [Пропустить] at any step → state cleared, response stays sent.
```

`[✓ Буду]` → `rsvp_status = accepted`, bot: "Great, see you there!" (no survey).

Module split:
- **rsvp** — RSVP buttons (in the notifier) + the `rsvp:` callback handler → `RecordRSVP` command.
- **botsurvey** — a new bot platform service (FSM) that delivers a survey **by `response_id`**; it
  knows nothing about meetings, so it is reusable for any survey response.
- **Reused Phase-1 core** — `model.Survey`, `ValidateAnswers`, `survey_repo`, the submit logic.

## Data model

One migration, following the goose convention. Timestamp later than `20260621120000`.

```sql
-- 1) survey_responses: generalize for a second source
ALTER TABLE survey_responses ALTER COLUMN token DROP NOT NULL;   -- token is web-only now
ALTER TABLE survey_responses
  ADD COLUMN source TEXT NOT NULL DEFAULT 'web_booking'
      CHECK (source IN ('web_booking','meeting_decline')),
  ADD COLUMN meeting_id UUID REFERENCES meetings(id) ON DELETE SET NULL,
  ADD COLUMN participant_telegram_id BIGINT;   -- bot delivery target + callback auth

-- 2) Survey assignment for meeting decline
ALTER TABLE organizations
  ADD COLUMN survey_on_decline_id UUID REFERENCES surveys(id) ON DELETE SET NULL;
ALTER TABLE meetings
  ADD COLUMN survey_on_decline_id UUID REFERENCES surveys(id) ON DELETE SET NULL;  -- override

-- 3) Participant RSVP status (confirm the participants table name during planning)
ALTER TABLE meeting_participants
  ADD COLUMN rsvp_status TEXT NOT NULL DEFAULT 'invited'
      CHECK (rsvp_status IN ('invited','accepted','declined'));
```

Decisions:
- **One `survey_responses` table for both sources.** `source` discriminates; `token` nullable
  (web), `meeting_id`/`participant_telegram_id` (meeting). Keeps Approach A's promise — the admin
  responses view shows both.
- **`token` keeps its `UNIQUE` constraint after `DROP NOT NULL`.** This is intentional and safe:
  Postgres treats NULLs as distinct, so many meeting-decline rows with `token = NULL` do not collide,
  while web tokens stay unique. Do NOT replace the unique index with a partial `WHERE token IS NOT NULL`
  index — the default behavior already gives the right semantics.
- **`participant_telegram_id` authorizes the bot callback:** on `survey:start:<responseId>` the bot
  verifies `update.from.id == response.participant_telegram_id` — a user cannot answer someone
  else's survey by guessing the UUID (the web `token` played this role for web).
- **`booker_email`/`booker_name` are reused as respondent fields** (filled with the participant's
  email/name for the meeting source) — no rename, to avoid churn in Phase-1 model/CSV/admin.
- Snapshot answers (`answers` JSONB) as in Phase 1.
- Survey resolution: `COALESCE(meeting.survey_on_decline_id, org.survey_on_decline_id)`; sent only
  if a survey is found **and** `is_active`.

## RSVP backend + organizer notification

Buttons: the notifier (`meeting_notifier`) attaches `[✓ Буду][❌ Не смогу]` to the meeting-created
notification and reminders, with callback data `rsvp:accept:<meetingId>` / `rsvp:decline:<meetingId>`.

A new `rsvp:` dispatch prefix in `multitenant.go` → application command:

```go
// CQRS command (changes state + sends a notification + may create a response)
func (s *Services) RecordRSVP(ctx context.Context, meetingID uuid.UUID, telegramID int64, status string)
    (offerSurvey *uuid.UUID, err error)
```

Steps:
1. `GetBotUserByTelegramID(telegramID)` → the participant's email/name/userID. If this Telegram user
   is **not** a participant of the meeting → `model.ErrForbidden`.
2. `UpdateParticipantRSVP(meetingID, email, status)` → sets `rsvp_status`.
3. **accept** → return `nil` (bot: "Great, see you there!").
4. **decline** →
   - notify the organizer (host) in the bot: `boti18n.T(hostLang, "rsvp.declined_host", name, title)`;
   - resolve the survey; if active → `CreateMeetingDeclineResponse(...)` (source=meeting_decline,
     meeting_id, participant_telegram_id, booker_email/name = participant, decline_reason=
     'meeting_decline', status=sent) → return `responseID`; else → `nil`.

Decisions:
- One command for both statuses; Telegram→participant mapping via the existing
  `meetingrecipients` pattern (`GetBotUserByEmail`/`GetBotUserByTelegramID`). Non-participant → 403/ignore.
- Organizer notification + response creation are **synchronous** in the RSVP handler (a single
  `SendMessage` + an INSERT) — no asynq; the participant is already in a bot dialog.
- A repeat RSVP overwrites the status (accept↔decline). A repeat decline with an existing
  unfinished `sent` response for the same meeting+participant **reuses** that response id (no
  duplicate rows).

## Bot conversational delivery (`botsurvey` FSM)

A new platform service `internal/platform/botsurvey/`, mirroring `botreg`/`meetingedit`: a Redis FSM
with `OnCallback`/`OnText`, registered in the `multitenant.go` dispatcher under the `survey:` prefix.

State (`botsurvey:<telegramID>`, Redis, ~30-min TTL):

```go
type State struct {
    ResponseID   uuid.UUID
    SurveyID     uuid.UUID
    QIndex       int            // current question
    Answers      []model.Answer // accumulated snapshot answers
    AwaitingText bool           // waiting for free-text
    MultiPicks   []string       // selection buffer for a multi question
}
```

Entry: `survey:start:<responseId>` → verify `from.ID == response.participant_telegram_id` (else
ignore) → load survey+questions → `QIndex=0` → render question 0. `survey:skip` → clear state,
response stays `sent`.

Rendering by question type:

| Type | UI | Callback |
|---|---|---|
| single | one inline button per option | `survey:opt:<i>` → store → next |
| rating | buttons `1..rating_max` | `survey:rate:<n>` |
| multi | toggle buttons (✓ on selected) + `[Готово]` | `survey:multi:<i>` (toggle), `survey:done` |
| text | bot sends the prompt, `AwaitingText=true` | next message captured by `OnText` |

Progress: after an answer `QIndex++` → more questions? render next : `SubmitSurveyResponse(responseID,
answers)` (Phase-1 validation) → `status=completed` → "Thanks!" → clear state.

`OnText` fires only when a `botsurvey` state exists AND `AwaitingText` — otherwise it passes through
(does not swallow other services' text, consistent with the existing OnText chain).

Decisions:
- **Reuse refactor:** extract `SubmitSurveyResponse(ctx, responseID, answers)` (validate + complete)
  as the core; Phase-1 `SubmitSurvey(token)` becomes a thin wrapper `token → id →
  SubmitSurveyResponse`. The bot calls the core by `responseID`. Web is unchanged.
- **Forward-only**, no "back" button in v1 (KISS). The whole survey can be abandoned via
  `survey:skip` at any step.
- **~30-min TTL:** if state expires mid-survey the response stays `sent` (like an abandoned web
  survey); the participant can restart from the same `[Начать]` (the responseId is still alive).
- `botsurvey` is decoupled from meetings — it only knows `survey_responses.id`. Reusable for
  Phase 3 / other triggers.

## Assignment + admin UI

Backend:
- Org level: `organizations.survey_on_decline_id`, set via the existing org PATCH (no new endpoint).
- Per-meeting override: `meetings.survey_on_decline_id`, threaded through the existing meeting
  create/update.

Admin (admin app):
- Org setting: a "Survey on meeting decline" select (active org surveys + "None") on the settings
  page, written to `organizations.survey_on_decline_id`.
- Per-meeting: a "Survey on decline (override)" select in the meeting edit dialog; "Org default" = null.
- Responses table: extend the Phase-1 responses view with a **source badge** (web booking / meeting
  decline) and a source filter; for the meeting source show the meeting title instead of the service.

Decisions:
- Org setting goes in the existing org PATCH; per-meeting in the existing meeting update — no new
  routes (KISS).
- The responses table is **extended** (badge + filter), not duplicated — Approach A promised a
  single view.
- Resolution (`meeting ?? org`) lives in the RSVP application command, not a DB view; the UI only
  sets the values.

## boti18n

A new `boti18n/catalog_survey.go` (ru/en/kk), all bot strings:
- RSVP: `rsvp.btn_accept`, `rsvp.btn_decline`, `rsvp.accepted`, `rsvp.declined_ok`,
  `rsvp.declined_host` ("{0} can't make «{1}»").
- Survey: `survey.offer`, `survey.btn_start`, `survey.btn_skip`, `survey.btn_done`, `survey.thanks`,
  `survey.closed`, `survey.already`.
- Operator-authored question/option text is **not** localized (their content, as in Phase 1).

## Error handling & edge cases

- **a)** Telegram user is not a participant on `rsvp:decline` → silently ignore (no error in chat).
- **b)** Survey deactivated between decline and `[Начать]` → "survey unavailable"; the FSM does not start.
- **c)** Response already `completed` (repeat `[Начать]`) → "you already completed this survey".
- **d)** FSM state expired mid-survey → response stays `sent`; `[Начать]` (responseId alive) restarts at question 0.
- **e)** Participant without a bot account (email not linked to Telegram) → they never received the
  bot notification (buttons exist only for bot users). Email-based RSVP is out of 2a (mini-app in 2b).
- **f)** Organizer without Telegram → organizer notification is best-effort, log
  `rsvp_host_notify_skipped`; the decline still goes through.
- **g)** Meeting deleted after decline → `meeting_id ON DELETE SET NULL`; the response survives.
- **h)** `multi` with no selection + `[Готово]` on a required question → re-render the question
  (`ValidateAnswers` rejects).
- **i)** Repeat decline (3c dedup): an unfinished `sent` response for the same meeting+participant is
  reused; no duplicate rows.

## Testing

Backend (unit, fake store as in Phase 1):
- `RecordRSVP`: accept → status updated, `offer=nil`; decline + assigned active survey → status,
  organizer notified, response created, `responseID` returned; decline with no survey → `nil`;
  non-participant → `ErrForbidden`; repeat decline → reuses the `sent` response (dedup).
- Assignment resolution: meeting override > org; org when meeting is null; neither → no survey.
- `SubmitSurveyResponse(responseID, answers)` core: validates + completes; web `SubmitSurvey(token)`
  still works (wrapper).
- `CreateMeetingDeclineResponse`: correct `source`/`meeting_id`/`participant_telegram_id`/respondent
  fields.

Bot FSM (`botsurvey`) — a **pure step function**, no Telegram:
- `(state, input) → (nextState, reply)`: start → render q0; single pick → next; rating → next; multi
  toggle + done → next; text awaiting → capture → next; last question → submit → thanks + clear;
  skip → clear; start with the wrong `telegram_id` → ignore; completed → "already".
  The store is mocked.

boti18n: a parity test that all `survey.*`/`rsvp.*` keys exist in ru/en/kk.

Frontend (admin): the assignment selects (org settings + meeting dialog) — typecheck; the
responses-page source badge/filter — a light change, no heavy logic.

## Out of scope (later)
- Mini-app decline button + mini-app survey rendering (Phase 2b).
- Operator notifications on response (Phase 3).
- A "back" button in the bot survey; editing submitted answers.
- Email-based RSVP for non-bot participants.
