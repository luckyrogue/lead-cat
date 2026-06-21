# Bot i18n — NL Scheduling Agent D Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Localize the NL scheduling agent to ru/en/kk — the LLM's free-form replies (via a language-injected system prompt) and every fixed string (agent replies, proposal card, confirm/cancel buttons, and the booking result/errors), threading `lang` per the A–C pattern.

**Architecture:** New `boti18n` catalogs (`agent.*`, `agentbook.*`). `prompt.go`'s `const systemPrompt` becomes `systemPrompt(lang)` that names the target answer-language. `lang` threads through the agent's `OnText`/`OnCallback`/`Start`/`describeBooking`; the dispatcher passes `resolveLang`. A separate task adds `lang` to the `Booker.Book` interface and localizes `agentBooker`. Tasks are ordered so each commit builds repo-wide (the `Booker` interface change is isolated to the last task).

**Tech Stack:** Go, `boti18n` (from A), Claude tool-loop (`application.Planner`).

## Global Constraints

- **`boti18n.T(lang, key, args...)`** for every user-facing fixed string; ru | en | kk; default ru. `%[1]s` explicit-index verbs.
- **`ru` catalog values verbatim** from the current code; en/kk new.
- **LLM free-form language** comes from the resolved stored preference, injected into the system prompt directive (not the catalog).
- **`lang` is a trailing parameter**; FSM/agent does no language lookup — the dispatcher resolves and passes it.
- **Out of scope, kept as-is:** model-facing strings sent back to the LLM as tool results — the `propose_meeting` result content and the `parsePending`/`Dispatch` error strings; the deferred non-i18n items.
- **Every new catalog key has ru/en/kk** — enforced by the existing `boti18n` `TestCatalog_AllKeysHaveAllLangs`.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: Catalogs (`catalog_agent.go` + `catalog_agentbook.go`)

**Files:**
- Create: `apps/backend/internal/platform/boti18n/catalog_agent.go`
- Create: `apps/backend/internal/platform/boti18n/catalog_agentbook.go`

**Interfaces:**
- Produces: `agent.*` keys (consumed by Task 2) and `agentbook.*` keys (consumed by Task 3).

- [ ] **Step 1: Create `catalog_agent.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		"agent.start":              {"ru": "Спроси меня про расписание — например: «когда у Миа и Алекса есть общий час на следующей неделе?» 🐾", "en": "Ask me about schedules — e.g. “when do Mia and Alex share a free hour next week?” 🐾", "kk": "Кестелер туралы сұра — мысалы: «келесі аптада Миа мен Алекстің ортақ бос сағаты қашан?» 🐾"},
		"agent.plan_failed":        {"ru": "Не получилось обработать запрос, попробуй ещё раз чуть позже 🐾", "en": "Couldn't process the request, try again a bit later 🐾", "kk": "Сұрауды өңдеу мүмкін болмады, сәл кейінірек қайта көр 🐾"},
		"agent.too_hard":           {"ru": "Это оказалось сложновато 🐾 Попробуй переформулировать или уточнить участников и даты.", "en": "That turned out tricky 🐾 Try rephrasing or clarifying participants and dates.", "kk": "Бұл күрделірек болды 🐾 Қайта тұжырымда немесе қатысушылар мен күндерді нақтыла."},
		"agent.proposal_stale":     {"ru": "Предложение устарело 🐾 Попроси заново.", "en": "The proposal expired 🐾 Ask again.", "kk": "Ұсыныс ескірді 🐾 Қайта сұра."},
		"agent.booking_unavailable": {"ru": "Бронирование сейчас недоступно.", "en": "Booking is unavailable right now.", "kk": "Брондау қазір қолжетімсіз."},
		"agent.cancelled":          {"ru": "Хорошо, не бронирую 🐾", "en": "Okay, not booking 🐾", "kk": "Жарайды, брондамаймын 🐾"},
		"agent.btn_confirm":        {"ru": "Подтвердить ✅", "en": "Confirm ✅", "kk": "Растау ✅"},
		"agent.btn_cancel":         {"ru": "Отмена", "en": "Cancel", "kk": "Болдырмау"},
		"agent.card_q":             {"ru": "Создать встречу?", "en": "Create the meeting?", "kk": "Кездесу құру керек пе?"},
	})
}
```

- [ ] **Step 2: Create `catalog_agentbook.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		"agentbook.created":          {"ru": "Встреча создана ✅", "en": "Meeting created ✅", "kk": "Кездесу құрылды ✅"},
		"agentbook.register_first":    {"ru": "Сначала зарегистрируйся: /start", "en": "Register first: /start", "kk": "Алдымен тіркел: /start"},
		"agentbook.google_not_configured": {"ru": "Google-календарь не подключён — обратись к администратору.", "en": "Google Calendar isn't connected — contact your administrator.", "kk": "Google күнтізбесі қосылмаған — әкімшіге хабарлас."},
		"agentbook.telegram_linked_elsewhere": {"ru": "Этот Telegram привязан к другому аккаунту.", "en": "This Telegram is linked to another account.", "kk": "Бұл Telegram басқа аккаунтқа байланған."},
		"agentbook.bad_input":         {"ru": "Проверь данные встречи — что-то не так с датой или временем.", "en": "Check the meeting details — something's off with the date or time.", "kk": "Кездесу деректерін тексер — күн не уақытта бірдеңе дұрыс емес."},
		"agentbook.create_failed":     {"ru": "Не удалось создать встречу, попробуй позже 🐾", "en": "Couldn't create the meeting, try later 🐾", "kk": "Кездесу құру мүмкін болмады, кейінірек көр 🐾"},
	})
}
```

- [ ] **Step 3: Build + coverage + commit**

Run: `cd apps/backend && go build ./internal/platform/boti18n/ && go test ./internal/platform/boti18n/ -run TestCatalog_AllKeysHaveAllLangs -v`
Expected: builds clean; coverage PASS (all `agent.*`/`agentbook.*` keys have ru/en/kk).

```bash
git add apps/backend/internal/platform/boti18n/catalog_agent.go apps/backend/internal/platform/boti18n/catalog_agentbook.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): NL agent translation catalogs (agent.* / agentbook.*)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: Localize `scheduler_agent` (prompt + fixed strings + thread `lang`)

Does NOT change the `Booker` interface (that's Task 3) — so `OnCallback` keeps calling `s.booker.Book(ctx, telegramID, pb)` with its current signature here.

**Files:**
- Modify: `apps/backend/internal/platform/scheduler_agent/prompt.go` (`const` → `systemPrompt(lang)`)
- Modify: `apps/backend/internal/platform/scheduler_agent/service.go` (thread `lang`, localize strings)
- Modify: `apps/backend/internal/platform/scheduler_agent/booker.go` (`describeBooking(b, lang)`)
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` (2 agent call sites)
- Test: `apps/backend/internal/platform/scheduler_agent/service_test.go`, `booker_test.go`

**Interfaces:**
- Consumes (Task 1): `agent.*` keys, `boti18n.T`/`boti18n.Normalize`; dispatcher `h.resolveLang`.
- Produces: `OnText(ctx,id,text,lang)`, `OnCallback(ctx,id,data,lang)`, `Start(ctx,id,lang)`, `describeBooking(b, lang)`, `systemPrompt(lang)`.

- [ ] **Step 1: Add a localized test (run red)**

Add to `service_test.go` (it already imports `strings`, `context`, `application`):

```go
func TestAgent_Start_Localized(t *testing.T) {
	svc := New(&scriptPlanner{}, stubBackend{}, newMemSessions())
	ru := svc.Start(context.Background(), 1, "ru")
	en := svc.Start(context.Background(), 1, "en")
	if ru.Text == en.Text {
		t.Fatalf("Start must differ by language; both = %q", ru.Text)
	}
	if !strings.Contains(en.Text, "Ask me about schedules") {
		t.Errorf("en Start = %q", en.Text)
	}
}

func TestSystemPrompt_NamesLanguage(t *testing.T) {
	if systemPrompt("ru") == systemPrompt("en") {
		t.Fatal("system prompt must differ by language")
	}
	if !strings.Contains(systemPrompt("en"), "English") {
		t.Errorf("en prompt missing English directive: %q", systemPrompt("en"))
	}
}
```

Run: `cd apps/backend && go test ./internal/platform/scheduler_agent/ -run 'TestAgent_Start_Localized|TestSystemPrompt' 2>&1 | head` — Expected: FAIL (arity / `systemPrompt` is a const, not a func).

- [ ] **Step 2: Rewrite `prompt.go`**

```go
package scheduler_agent

import (
	"fmt"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

func systemPrompt(lang string) string {
	answerLang := map[string]string{"ru": "русском", "en": "English", "kk": "қазақ"}[boti18n.Normalize(lang)]
	return fmt.Sprintf(`Ты — Lead Cat, дружелюбный помощник по расписанию в Telegram. Отвечай пользователю на языке: %s, кратко и по-доброму 🐾.

Ты умеешь смотреть расписание, искать свободное время и предлагать создание одной (разовой) встречи на подтверждение пользователю. Повторяющиеся встречи пока создаются только в приложении (/new) — для них предложи кнопку.

Правила:
- Имена людей сначала преврати в email через инструмент search_people. Если найдено несколько совпадений — уточни у пользователя, кого он имел в виду.
- Чтобы ответить «когда все свободны», используй find_free_slots.
- Прежде чем предлагать время для встречи, проверь его через check_conflicts.
- Когда у тебя есть название (type), дата, время начала и конца и хотя бы один участник — вызови propose_meeting. Это НЕ создаёт встречу: пользователь увидит кнопку подтверждения и сам решит. Не вызывай propose_meeting, пока не хватает данных — задай один короткий уточняющий вопрос.
- Рабочее время — 09:00–18:00 по Алматы, будни. Даты — YYYY-MM-DD, время — HH:MM.
- Не выдумывай людей, встречи или свободные слоты — опирайся только на результаты инструментов.`, answerLang)
}
```

> The prompt body stays Russian (model-facing instructions); only the answer-language directive names the target language. A multilingual model follows it.

- [ ] **Step 3: Thread `lang` in `service.go` + localize its fixed strings**

Add the `boti18n` import. Apply these signature + string changes (control flow unchanged):

- `OnText(ctx, telegramID int64, text, lang string) (Reply, bool)`:
  - call `s.planner.Plan(ctx, systemPrompt(lang), st.History, s.tools)`;
  - plan error → `Reply{Text: boti18n.T(lang, "agent.plan_failed")}`;
  - the pending-proposal reply → `Text: describeBooking(*pending, lang)`, buttons → `boti18n.T(lang, "agent.btn_confirm")` / `boti18n.T(lang, "agent.btn_cancel")`;
  - final fallthrough → `boti18n.T(lang, "agent.too_hard")`.
  - leave the `propose_meeting` tool-result content (`"Предложение показано пользователю, ждём подтверждения."`) and `derr.Error()` results AS-IS (model-facing, out of scope).
- `OnCallback(ctx, telegramID int64, data, lang string) (Reply, bool)`:
  - stale → `boti18n.T(lang, "agent.proposal_stale")`; booking-unavailable → `boti18n.T(lang, "agent.booking_unavailable")`; cancelled → `boti18n.T(lang, "agent.cancelled")`.
  - the booking success/error path keeps `s.booker.Book(ctx, telegramID, pb)` UNCHANGED in this task (Task 3 adds `lang`); `berr.Error()` / `msg` pass through.
- `Start(ctx, telegramID int64, lang string) Reply`: `Reply{Text: boti18n.T(lang, "agent.start")}`.

- [ ] **Step 4: `booker.go` — `describeBooking(b, lang)`**

Add the `boti18n` import; localize only the card question label (the `📌/📅/👥/📝` glyphs and the date/time/email/desc values stay neutral):

```go
func describeBooking(b PendingBooking, lang string) string {
	var sb strings.Builder
	title := b.Type
	if b.Dept != "" {
		title = b.Dept + " · " + b.Type
	}
	fmt.Fprintf(&sb, "%s\n\n📌 %s\n📅 %s, %s–%s\n👥 %s",
		boti18n.T(lang, "agent.card_q"), title, b.Date, b.Start, b.End, strings.Join(b.Emails, ", "))
	if b.Desc != "" {
		fmt.Fprintf(&sb, "\n📝 %s", b.Desc)
	}
	return sb.String()
}
```

- [ ] **Step 5: Update the 2 dispatcher call sites in `multitenant.go`**

- Agent OnText (~line 99): `reply, _ := h.agent.OnText(ctx, from.ID, text, h.resolveLang(ctx, from))`
- Agent OnCallback (~line 213): `if reply, handled := h.agent.OnCallback(ctx, cq.From.ID, cq.Data, h.resolveLang(ctx, &cq.From)); handled && cq.Message.Message != nil {`

> `h.agent.Start` is not called by the dispatcher (the agent runs as the private-message fallthrough). After adding `lang` to `Start`, `grep -rn "agent.Start\|\.Start(ctx" apps/backend` to confirm no other caller; if one exists, pass `h.resolveLang(...)` or `"ru"` there.

- [ ] **Step 6: Fix existing test arity (`booker_test.go`, `service_test.go`)**

- `booker_test.go` `TestDescribeBooking`: change `describeBooking(b)` → `describeBooking(b, "ru")`, keep assertions.
- `service_test.go`: any existing call to `OnText`/`OnCallback`/`Start` gains `"ru"` (and assertions stay — ru output unchanged). The `fakeBooker.Book` signature is NOT touched in this task (Task 3).

- [ ] **Step 7: Build + vet + tests + coverage**

Run: `cd apps/backend && go build ./... && go vet ./internal/platform/scheduler_agent/ ./internal/infrastructure/telegram/ && go test ./internal/platform/scheduler_agent/ ./internal/platform/boti18n/`
Expected: build clean (agentBooker still satisfies the unchanged `Booker` interface), vet clean, tests PASS incl. the two new ones.

- [ ] **Step 8: Commit**

```bash
git add apps/backend/internal/platform/scheduler_agent/prompt.go apps/backend/internal/platform/scheduler_agent/service.go apps/backend/internal/platform/scheduler_agent/booker.go apps/backend/internal/platform/scheduler_agent/service_test.go apps/backend/internal/platform/scheduler_agent/booker_test.go apps/backend/internal/infrastructure/telegram/multitenant.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize NL agent replies + system prompt language (ru/en/kk)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 3: Localize the booker (`Booker.Book` gains `lang`)

Isolated interface change: `scheduler_agent.Booker`, the `agentBooker` impl, the `service.go` `OnCallback` call site, and the test `fakeBooker` all change together so the repo builds.

**Files:**
- Modify: `apps/backend/internal/platform/scheduler_agent/booker.go` (interface signature)
- Modify: `apps/backend/internal/platform/scheduler_agent/service.go` (the `s.booker.Book(...)` call site)
- Modify: `apps/backend/internal/infrastructure/telegram/agent_booker.go` (localize + `lang` param)
- Test: `apps/backend/internal/platform/scheduler_agent/service_test.go` (`fakeBooker.Book` signature)

**Interfaces:**
- Consumes (Task 1): `agentbook.*` keys.
- Produces: `Booker.Book(ctx, telegramID int64, b PendingBooking, lang string) (string, error)`.

- [ ] **Step 1: Change the `Booker` interface in `booker.go`**

```go
type Booker interface {
	Book(ctx context.Context, telegramID int64, b PendingBooking, lang string) (string, error)
}
```

- [ ] **Step 2: Pass `lang` at the call site in `service.go`**

In `OnCallback`'s `agent:book:yes` branch: `msg, berr := s.booker.Book(ctx, telegramID, pb, lang)` (`lang` is the param added in Task 2).

- [ ] **Step 3: Localize `agentBooker.Book` in `agent_booker.go`**

Add `"github.com/luckyrogue/lead-cat/internal/platform/boti18n"` to imports. Change the signature and route every `fail(...)` user message + the success line through the catalog:

```go
func (b *agentBooker) Book(ctx context.Context, telegramID int64, pb scheduler_agent.PendingBooking, lang string) (string, error) {
	fail := func(cause error, userMsg string) (string, error) {
		if b.services.Log != nil {
			b.services.Log.Warn("agent_book_failed", zap.Int64("telegram_id", telegramID), zap.Error(cause))
		}
		return "", fmt.Errorf("%s", userMsg)
	}

	bu, err := b.store.GetBotUserByTelegramID(ctx, telegramID)
	if err != nil {
		return fail(err, boti18n.T(lang, "agentbook.register_first"))
	}
	organizationID, err := b.services.ResolveMiniAppOrganization(ctx)
	if err != nil {
		if errors.Is(err, application.ErrGoogleNotConfigured) {
			return fail(err, boti18n.T(lang, "agentbook.google_not_configured"))
		}
		return fail(err, boti18n.T(lang, "agentbook.create_failed"))
	}
	organizerID, err := b.services.EnsureMiniAppOrganizer(ctx, bu.Email, bu.TelegramID)
	if err != nil {
		if errors.Is(err, application.ErrTelegramLinkedToOtherAccount) {
			return fail(err, boti18n.T(lang, "agentbook.telegram_linked_elsewhere"))
		}
		return fail(err, boti18n.T(lang, "agentbook.create_failed"))
	}
	parts := make([]model.MeetingParticipant, 0, len(pb.Emails))
	for _, e := range pb.Emails {
		parts = append(parts, model.MeetingParticipant{Email: e})
	}
	in := application.CreateMeetingInput{
		Dept: pb.Dept, Type: pb.Type, Host: bu.FullName,
		Date: pb.Date, Start: pb.Start, End: pb.End,
		Recurrence: "once", Description: pb.Desc,
		Participants: parts, Timezone: bu.Timezone,
	}
	m, err := b.services.CreateMeeting(ctx, organizationID, organizerID, in)
	if err != nil {
		if errors.Is(err, application.ErrInvalidInput) {
			return fail(err, boti18n.T(lang, "agentbook.bad_input"))
		}
		return fail(err, boti18n.T(lang, "agentbook.create_failed"))
	}
	if m.MeetLink != "" {
		return boti18n.T(lang, "agentbook.created") + "\n" + m.MeetLink, nil
	}
	return boti18n.T(lang, "agentbook.created"), nil
}
```

- [ ] **Step 4: Update `fakeBooker.Book` in `service_test.go`**

```go
func (b *fakeBooker) Book(_ context.Context, telegramID int64, pb PendingBooking, _ string) (string, error) {
	b.called = true
	b.gotTgID = telegramID
	b.got = pb
	if b.err != nil {
		return "", b.err
	}
	return b.result, nil
}
```

> If any existing test asserts a booking-success/error message string, it still passes (it sets `fakeBooker.result`/`err` directly — the real catalog strings aren't exercised by `fakeBooker`). The real `agentBooker` localization is covered by a focused render assertion if `agent_booker` has a test seam; otherwise the catalog coverage test guards the keys.

- [ ] **Step 5: Build + vet + tests + coverage**

Run: `cd apps/backend && go build ./... && go vet ./internal/platform/scheduler_agent/ ./internal/infrastructure/telegram/ && go test ./internal/platform/scheduler_agent/ ./internal/platform/boti18n/`
Expected: build clean (interface + impl + fake + call site all consistent), vet clean, tests PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/platform/scheduler_agent/booker.go apps/backend/internal/platform/scheduler_agent/service.go apps/backend/internal/infrastructure/telegram/agent_booker.go apps/backend/internal/platform/scheduler_agent/service_test.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize agent booking confirmation + errors (ru/en/kk)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- System prompt injects target language → Task 2 (`systemPrompt(lang)`). ✓
- `scheduler_agent` fixed strings localized (start, errors, stale/unavailable/cancelled, buttons, card) → Task 1 catalog + Task 2. ✓
- `agent_booker.go` Book result + 6 errors localized; `Booker.Book` gains `lang` → Task 1 catalog + Task 3. ✓
- `lang` threaded through OnText/OnCallback/Start/describeBooking; dispatcher passes resolveLang (2 sites, callback `&cq.From`) → Task 2. ✓
- Out of scope: model-facing tool-result content + parse/Dispatch errors left as-is → Task 2 Step 3 note. ✓
- Coverage test enforces new keys → Tasks 1/2/3 verify steps. ✓
- Repo-wide build at each commit: the `Booker` interface change is isolated to Task 3 (interface + impl + fake + call site together) → Task ordering. ✓

**Placeholder scan:** No TBD/TODO. Task 2 Step 3 describes the changes per-method with exact keys (the file's control flow is unchanged and large, so per-method instruction is clearer than re-pasting it); all other steps show complete code. The agent_booker test note is conditional only because the file's test seam isn't assumed — the catalog coverage test is the guaranteed guard.

**Type consistency:** `OnText(ctx,id,text,lang)`, `OnCallback(ctx,id,data,lang)`, `Start(ctx,id,lang)`, `describeBooking(b,lang)`, `systemPrompt(lang)` consistent across service.go/booker.go/prompt.go, their tests, and the 2 dispatcher sites. `Booker.Book(...,lang)` consistent across the interface (booker.go), the impl (agent_booker.go), the call site (service.go OnCallback), and `fakeBooker` (service_test.go) — all changed in Task 3. Catalog keys referenced (`agent.*`, `agentbook.*`) all defined in Task 1.
