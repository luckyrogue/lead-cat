# Bot i18n — NL Scheduling Agent D (design)

## Context: bot localization, sub-project D of A–D (final)

Sub-projects A (catalog + commands/botreg), B (notifications), C1 (checker/
scheduleview/botsettings), and C2 (meetingedit) are done. D localizes the **NL
scheduling agent** — the Claude tool-loop that handles free-form private messages in
the bot. This is the last bot-i18n sub-project; after it the bot is trilingual end to
end (excluding the separately-tracked deferred items: `Asia/Almaty` date parsing,
a few hardcoded guard lines in `multitenant.go`, the domain `Recurrence.Label()`).

## Problem

The agent is hardcoded Russian on two axes:

1. **The LLM's free-form replies** are driven by a system prompt (`prompt.go`) whose
   directive is `Отвечай по-русски`. So the model always answers in Russian regardless
   of the user's language.
2. **Fixed Go strings** — the agent's error replies, the proposal confirm card
   (`describeBooking` in `booker.go`), the confirm/cancel buttons, the `Start` hint
   (all in `scheduler_agent`), and the booking result/error messages in
   `agent_booker.go` (`telegram` package) — are Russian literals.

`OnText`/`OnCallback`/`Start` and `Booker.Book` take no language. This is a
registered-user flow (the agent only runs after a `GetBotUserByTelegramID` check), so
the language is the stored `bot_users.language`, resolvable by the dispatcher's
`resolveLang` helper.

## Goal

The agent answers — both its free-form LLM text and every fixed string, including the
final booking confirmation — in the user's stored language (ru/en/kk). Fixed strings
go through `boti18n`; the LLM's free-form language is set by injecting the resolved
language into the system prompt.

## Design

### 1. System prompt — inject the target language

`prompt.go`'s `const systemPrompt` becomes a function `systemPrompt(lang string) string`.
The only behavioral change is the answer-language directive: the current
`Отвечай по-русски, кратко и по-доброму 🐾` line is replaced with one that names the
resolved language (e.g. `Отвечай на языке пользователя: <ru|en|kk>, кратко и
по-доброму 🐾`). The rest of the prompt (the rules, tool guidance) stays as-is — it is
model-facing instruction text, not user-facing, and a multilingual model follows a
Russian-instruction prompt that targets any output language. `lang` is the resolved
stored preference (consistent with the rest of the app), `boti18n.Normalize`d.

### 2. Fixed strings in `scheduler_agent` → catalog

New `boti18n/catalog_agent.go` (`agent.*` keys), ru verbatim, en/kk new:
- `Start` hint ("Спроси меня про расписание …").
- Errors: plan-failed ("Не получилось обработать запрос …"), too-hard ("Это оказалось
  сложновато …").
- `OnCallback`: proposal-stale ("Предложение устарело …"), booking-unavailable
  ("Бронирование сейчас недоступно."), cancelled-ok ("Хорошо, не бронирую 🐾").
- Buttons: "Подтвердить ✅", "Отмена".
- `describeBooking` card (`booker.go`): "Создать встречу?" + the `📌/📅/👥/📝` line
  labels (the glyphs and the date/time/email values stay neutral).

### 3. `agent_booker.go` — localize `Book` (interface change)

`scheduler_agent.Booker.Book` gains a trailing `lang string`:
`Book(ctx, telegramID int64, b PendingBooking, lang string) (string, error)`. The
`agentBooker` implementation routes every user-facing message through a new
`boti18n/catalog_agentbook.go` (`agentbook.*`): the success line ("Встреча создана ✅"
+ optional Meet link), and the six failure messages (register-first, Google-not-
configured, telegram-linked-elsewhere, invalid-input, generic create-failed). ru
verbatim, en/kk new. The agent's `OnCallback` passes its `lang` into `Book`.

### 4. Thread `lang` + dispatcher

`OnText(ctx, id, text, lang)`, `OnCallback(ctx, id, data, lang)`, `Start(ctx, id, lang)`,
`describeBooking(b, lang)` all gain `lang`. The dispatcher (`multitenant.go`) passes
`h.resolveLang(ctx, from)` at the agent `OnText` site (~line 99) and
`h.resolveLang(ctx, &cq.From)` at the agent `OnCallback` site (~line 213). `Start`
(not currently dispatched) is threaded for consistency.

### Out of scope (D)

- **Model-facing strings** that go back to the LLM as tool results, not to the user —
  the `propose_meeting` tool-result content ("Предложение показано пользователю …")
  and the `parsePending` / `Dispatch` error strings (sent as `IsError` tool results).
  These influence the model, not the user; the model is multilingual and reads them
  fine. Localizing them adds noise with no user benefit.
- The deferred non-i18n items (`Asia/Almaty` parsing, `multitenant.go` guard lines,
  domain `Recurrence.Label()`).
- Plural/gender rules — plain catalog strings (consistent with A–C).

## Error handling

- Missing key → key returned (loud in dev); missing language → ru. The agent's
  control flow, tool loop, and booking transaction are unchanged — only string sources
  and the prompt directive move.

## Testing

- **`scheduler_agent` tests:** `Start` localized (ru≠en, English phrase); `describeBooking`
  localized (card label differs by language, neutral values intact); `OnCallback`
  cancel/stale paths localized. Reuse the existing `service_test.go`/`booker_test.go`
  fakes; the planner is already faked, so a fake turn with no tool calls exercises the
  free-form path, and the system-prompt function can be unit-tested directly
  (`systemPrompt("en")` names English / differs from `systemPrompt("ru")`).
- **`agent_booker.go`:** a test (or extension) asserting `Book(..., "en")` returns the
  English success line and an English error for a forced failure, using its existing
  test seam if present; otherwise assert via the catalog render.
- The `boti18n` coverage test confirms all new `agent.*` / `agentbook.*` keys have
  ru/en/kk.
