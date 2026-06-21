# Bot i18n — meetingedit C2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Localize the `meetingedit` FSM (reply text + inline-keyboard button labels + the `menuText` field summary + conflict warning) to ru/en/kk by threading `lang` through every function, per the A/B/C1 pattern.

**Architecture:** A new `boti18n/catalog_meetingedit.go` holds all `medit.*` ru/en/kk strings. Every `meetingedit` function that emits user-facing text gains a trailing `lang string`; the dispatcher resolves it once with `h.resolveLang(...)` and passes it into the three `editor` call sites. Because the call graph is fully connected, the threading lands as one atomic file rewrite (Task 2) after the catalog (Task 1).

**Tech Stack:** Go, `boti18n` (from sub-project A), `go-telegram/bot`.

## Global Constraints

- **`boti18n.T(lang, key, args...)`** for every user-facing string (reply text AND button labels); ru | en | kk; default ru. `%[1]s`/`%[1]d` explicit-index verbs.
- **`ru` catalog values verbatim** from the current code; en/kk new.
- **`lang` is a trailing parameter** threaded through every text-emitting method/helper; the FSM does no language lookup — the dispatcher resolves and passes it.
- **Parse errors localized at the FSM level** via clean keys (`medit.bad_datetime`/`medit.bad_time_range`); `parse.go` is NOT touched.
- **Out of scope, kept as-is:** the domain `meeting.Recurrence(v).Label()` call in `recLabel` (non-`once` values keep the domain Russian label); `loadLoc(tz)`; `summary` (`«name»` + meet link, neutral); the `•`/`★`/`«»`/`🔗`/emoji glyphs and date/time number formatting.
- **Every new catalog key has ru/en/kk** — enforced by the existing `boti18n` `TestCatalog_AllKeysHaveAllLangs`.
- **Commit message footer** (every commit):
  ```
  Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
  ```

---

### Task 1: Catalog (`catalog_meetingedit.go`)

**Files:**
- Create: `apps/backend/internal/platform/boti18n/catalog_meetingedit.go`

**Interfaces:**
- Produces: the `medit.*` keys consumed by Task 2.

- [ ] **Step 1: Create `catalog_meetingedit.go`**

```go
package boti18n

func init() {
	register(map[string]map[string]string{
		// ── shared ──
		"medit.session_expired": {"ru": "Сессия истекла. Начни заново: /edit", "en": "Session expired. Start over: /edit", "kk": "Сессия аяқталды. Қайта баста: /edit"},
		"medit.forbidden":       {"ru": "Нет доступа к этой встрече.", "en": "You don't have access to this meeting.", "kk": "Бұл кездесуге қол жеткізу жоқ."},
		// ── Start ──
		"medit.list_failed":   {"ru": "Не удалось получить встречи, попробуй позже.", "en": "Couldn't load meetings, try later.", "kk": "Кездесулерді алу мүмкін болмады, кейінірек көр."},
		"medit.none_editable": {"ru": "Нет предстоящих встреч для редактирования.\n(Убедись, что Telegram привязан в приложении.)", "en": "No upcoming meetings to edit.\n(Make sure Telegram is linked in the app.)", "kk": "Өңдеуге арналған алдағы кездесулер жоқ.\n(Telegram қосымшада байланғанына көз жеткіз.)"},
		"medit.pick_meeting":  {"ru": "Выбери встречу для редактирования:", "en": "Pick a meeting to edit:", "kk": "Өңдейтін кездесуді таңда:"},
		// ── OnCallback ──
		"medit.cancelled": {"ru": "Редактирование отменено.", "en": "Editing cancelled.", "kk": "Өңдеу болдырылмады."},
		// ── OnText ──
		"medit.bad_time_range": {"ru": "Неверный формат времени. Введи ЧЧ:ММ–ЧЧ:ММ и попробуй ещё раз:", "en": "Invalid time format. Enter HH:MM–HH:MM and try again:", "kk": "Уақыт форматы қате. СС:ММ–СС:ММ енгізіп қайта көр:"},
		"medit.bad_datetime":   {"ru": "Неверный формат даты и времени. Введи ГГГГ-ММ-ДД ЧЧ:ММ–ЧЧ:ММ и попробуй ещё раз:", "en": "Invalid date/time format. Enter YYYY-MM-DD HH:MM–HH:MM and try again:", "kk": "Күн/уақыт форматы қате. ЖЖЖЖ-АА-КК СС:ММ–СС:ММ енгізіп қайта көр:"},
		"medit.empty_value":    {"ru": "Пусто. Введи значение:", "en": "Empty. Enter a value:", "kk": "Бос. Мән енгіз:"},
		// ── pick ──
		"medit.unknown_meeting":     {"ru": "Неизвестная встреча.", "en": "Unknown meeting.", "kk": "Белгісіз кездесу."},
		"medit.get_meeting_failed":  {"ru": "Не удалось получить встречу.", "en": "Couldn't load the meeting.", "kk": "Кездесуді алу мүмкін болмады."},
		"medit.not_editable":        {"ru": "Эта встреча недоступна для редактирования.", "en": "This meeting can't be edited.", "kk": "Бұл кездесуді өңдеуге болмайды."},
		// ── field / prompts ──
		"medit.prompt_dept":        {"ru": "Введи новый отдел:", "en": "Enter the new department:", "kk": "Жаңа бөлімді енгіз:"},
		"medit.prompt_type":        {"ru": "Введи новый тип встречи:", "en": "Enter the new meeting type:", "kk": "Жаңа кездесу түрін енгіз:"},
		"medit.prompt_host":        {"ru": "Введи нового ведущего:", "en": "Enter the new host:", "kk": "Жаңа жүргізушіні енгіз:"},
		"medit.prompt_description":  {"ru": "Введи новое описание:", "en": "Enter the new description:", "kk": "Жаңа сипаттаманы енгіз:"},
		"medit.prompt_datetime":     {"ru": "Введи дату и время: ГГГГ-ММ-ДД ЧЧ:ММ–ЧЧ:ММ\n(например: 2026-06-01 14:00–15:00)", "en": "Enter date and time: YYYY-MM-DD HH:MM–HH:MM\n(e.g. 2026-06-01 14:00–15:00)", "kk": "Күн мен уақытты енгіз: ЖЖЖЖ-АА-КК СС:ММ–СС:ММ\n(мысалы: 2026-06-01 14:00–15:00)"},
		"medit.prompt_time_series":  {"ru": "Введи новое время ЧЧ:ММ–ЧЧ:ММ (например: 15:00–16:00):", "en": "Enter the new time HH:MM–HH:MM (e.g. 15:00–16:00):", "kk": "Жаңа уақытты енгіз СС:ММ–СС:ММ (мысалы: 15:00–16:00):"},
		// ── apply ──
		"medit.choose_scope_first": {"ru": "Сначала выбери: эту встречу или всю серию.", "en": "First choose: this meeting or the whole series.", "kk": "Алдымен таңда: осы кездесу немесе бүкіл серия."},
		"medit.no_changes":         {"ru": "Нет изменений. Выбери поле или нажми «Отмена».", "en": "No changes. Pick a field or press “Cancel”.", "kk": "Өзгерістер жоқ. Өрісті таңда немесе «Болдырмау» бас."},
		// ── doApply ──
		"medit.invalid_data":        {"ru": "Неверные данные. Поправь поле и попробуй снова.", "en": "Invalid data. Fix the field and try again.", "kk": "Деректер қате. Өрісті түзетіп қайта көр."},
		"medit.series_not_editable": {"ru": "Часть серии больше недоступна для редактирования.", "en": "Part of the series is no longer editable.", "kk": "Серияның бөлігі енді өңделмейді."},
		"medit.update_series_failed": {"ru": "Не удалось обновить серию, попробуй позже.", "en": "Couldn't update the series, try later.", "kk": "Серияны жаңарту мүмкін болмады, кейінірек көр."},
		"medit.series_updated":      {"ru": "Готово ✏️ — обновлено встреч серии: %[1]d", "en": "Done ✏️ — series meetings updated: %[1]d", "kk": "Дайын ✏️ — серия кездесулері жаңартылды: %[1]d"},
		"medit.meeting_not_editable": {"ru": "Встреча больше недоступна для редактирования.", "en": "The meeting is no longer editable.", "kk": "Кездесу енді өңделмейді."},
		"medit.update_failed":       {"ru": "Не удалось обновить встречу, попробуй позже.", "en": "Couldn't update the meeting, try later.", "kk": "Кездесуді жаңарту мүмкін болмады, кейінірек көр."},
		"medit.updated_done":        {"ru": "Готово ✏️", "en": "Done ✏️", "kk": "Дайын ✏️"},
		// ── conflict ──
		"medit.conflict_header":  {"ru": "⚠ Внимание! У следующих участников уже есть встречи в это время:\n", "en": "⚠ Warning! These participants already have meetings at this time:\n", "kk": "⚠ Назар аударыңыз! Мына қатысушыларда осы уақытта кездесулер бар:\n"},
		"medit.conflict_apply_q": {"ru": "\nПрименить изменения?", "en": "\nApply the changes?", "kk": "\nӨзгерістерді қолдану керек пе?"},
		"medit.btn_apply_yes":    {"ru": "Да, применить", "en": "Yes, apply", "kk": "Иә, қолдану"},
		"medit.btn_change_time":  {"ru": "Изменить время", "en": "Change the time", "kk": "Уақытты өзгерту"},
		// ── parts ──
		"medit.get_parts_failed": {"ru": "Не удалось получить участников.", "en": "Couldn't load participants.", "kk": "Қатысушыларды алу мүмкін болмады."},
		"medit.parts_title":      {"ru": "Участники встречи:", "en": "Meeting participants:", "kk": "Кездесу қатысушылары:"},
		"medit.parts_empty":      {"ru": "Участников пока нет.", "en": "No participants yet.", "kk": "Әзірге қатысушылар жоқ."},
		"medit.btn_add":          {"ru": "➕ Добавить", "en": "➕ Add", "kk": "➕ Қосу"},
		"medit.btn_back":         {"ru": "⬅ Назад", "en": "⬅ Back", "kk": "⬅ Артқа"},
		// ── padd / search ──
		"medit.padd_prompt":   {"ru": "Введи email участника или часть имени для поиска:", "en": "Enter a participant email or part of a name to search:", "kk": "Іздеу үшін қатысушының email-ін немесе атының бөлігін енгіз:"},
		"medit.search_failed": {"ru": "Не удалось выполнить поиск, попробуй ещё раз:", "en": "Search failed, try again:", "kk": "Іздеу сәтсіз, қайта көр:"},
		"medit.btn_add_email": {"ru": "➕ Добавить %[1]s", "en": "➕ Add %[1]s", "kk": "➕ %[1]s қосу"},
		"medit.search_none":   {"ru": "Ничего не найдено. Введи корректный email или часть имени:", "en": "Nothing found. Enter a valid email or part of a name:", "kk": "Ештеңе табылмады. Дұрыс email немесе атының бөлігін енгіз:"},
		"medit.padd_pick":     {"ru": "Выбери, кого добавить:", "en": "Pick who to add:", "kk": "Кімді қосатыныңды таңда:"},
		// ── paddPick ──
		"medit.cand_not_found":      {"ru": "Кандидат не найден, начни добавление заново.", "en": "Candidate not found, start adding again.", "kk": "Үміткер табылмады, қосуды қайта баста."},
		"medit.already_or_invalid":  {"ru": "Уже участник или неверный email.", "en": "Already a participant or invalid email.", "kk": "Қатысушы болып қойған немесе email қате."},
		"medit.add_failed":          {"ru": "Не удалось добавить участника, попробуй позже.", "en": "Couldn't add the participant, try later.", "kk": "Қатысушыны қосу мүмкін болмады, кейінірек көр."},
		// ── prem / premConfirm ──
		"medit.part_not_found":     {"ru": "Участник не найден, открой список заново.", "en": "Participant not found, reopen the list.", "kk": "Қатысушы табылмады, тізімді қайта аш."},
		"medit.remove_confirm":     {"ru": "Удалить участника %[1]s?", "en": "Remove participant %[1]s?", "kk": "%[1]s қатысушыны өшіру керек пе?"},
		"medit.btn_yes":            {"ru": "✅ Да", "en": "✅ Yes", "kk": "✅ Иә"},
		"medit.btn_cancel_back":    {"ru": "⬅ Отмена", "en": "⬅ Cancel", "kk": "⬅ Болдырмау"},
		"medit.nothing_to_remove":  {"ru": "Нечего удалять, открой список заново.", "en": "Nothing to remove, reopen the list.", "kk": "Өшіретін ештеңе жоқ, тізімді қайта аш."},
		"medit.remove_failed":      {"ru": "Не удалось удалить участника, попробуй позже.", "en": "Couldn't remove the participant, try later.", "kk": "Қатысушыны өшіру мүмкін болмады, кейінірек көр."},
		// ── menuKeyboard ──
		"medit.btn_time":         {"ru": "🕒 Время", "en": "🕒 Time", "kk": "🕒 Уақыт"},
		"medit.btn_datetime":     {"ru": "📅 Дата/время", "en": "📅 Date/time", "kk": "📅 Күн/уақыт"},
		"medit.btn_dept":         {"ru": "🏢 Отдел", "en": "🏢 Department", "kk": "🏢 Бөлім"},
		"medit.btn_type":         {"ru": "🏷 Тип", "en": "🏷 Type", "kk": "🏷 Түрі"},
		"medit.btn_host":         {"ru": "🎤 Ведущий", "en": "🎤 Host", "kk": "🎤 Жүргізуші"},
		"medit.btn_description":  {"ru": "📝 Описание", "en": "📝 Description", "kk": "📝 Сипаттама"},
		"medit.btn_recurrence":   {"ru": "🔁 Частота", "en": "🔁 Recurrence", "kk": "🔁 Жиілік"},
		"medit.btn_participants": {"ru": "👥 Участники", "en": "👥 Participants", "kk": "👥 Қатысушылар"},
		"medit.btn_delete":       {"ru": "🗑 Удалить", "en": "🗑 Delete", "kk": "🗑 Өшіру"},
		"medit.btn_apply":        {"ru": "✅ Применить", "en": "✅ Apply", "kk": "✅ Қолдану"},
		"medit.btn_cancel":       {"ru": "✖ Отмена", "en": "✖ Cancel", "kk": "✖ Болдырмау"},
		// ── scopeReply ──
		"medit.scope_q":          {"ru": "Эта встреча или вся серия (эта и далее)?", "en": "This meeting or the whole series (this and later)?", "kk": "Осы кездесу ме әлде бүкіл серия ма (осы және кейінгі)?"},
		"medit.btn_scope_one":    {"ru": "📍 Эта встреча", "en": "📍 This meeting", "kk": "📍 Осы кездесу"},
		"medit.btn_scope_series": {"ru": "🔁 Вся серия (эта и далее)", "en": "🔁 Whole series (this and later)", "kk": "🔁 Бүкіл серия (осы және кейінгі)"},
		// ── confirmDelete / doDelete / deleteErr ──
		"medit.delete_one_q":      {"ru": "Удалить эту встречу?", "en": "Delete this meeting?", "kk": "Осы кездесуді өшіру керек пе?"},
		"medit.delete_series_q":   {"ru": "Удалить всю серию (эту и далее)? Это отменит все будущие встречи серии.", "en": "Delete the whole series (this and later)? This cancels all future meetings in the series.", "kk": "Бүкіл серияны өшіру керек пе (осы және кейінгі)? Бұл серияның барлық болашақ кездесулерін болдырмайды."},
		"medit.btn_delete_yes":    {"ru": "✅ Да, удалить", "en": "✅ Yes, delete", "kk": "✅ Иә, өшіру"},
		"medit.series_deleted":    {"ru": "Удалено встреч серии: %[1]d ❌", "en": "Series meetings deleted: %[1]d ❌", "kk": "Серия кездесулері өшірілді: %[1]d ❌"},
		"medit.meeting_deleted":   {"ru": "Встреча удалена ❌", "en": "Meeting deleted ❌", "kk": "Кездесу өшірілді ❌"},
		"medit.meeting_unavailable": {"ru": "Встреча больше недоступна.", "en": "The meeting is no longer available.", "kk": "Кездесу енді қолжетімді емес."},
		"medit.delete_failed":     {"ru": "Не удалось удалить, попробуй позже.", "en": "Couldn't delete, try later.", "kk": "Өшіру мүмкін болмады, кейінірек көр."},
		// ── recReply / recLabel ──
		"medit.pick_recurrence": {"ru": "Выбери частоту:", "en": "Pick a recurrence:", "kk": "Жиілікті таңда:"},
		"medit.rec.once":        {"ru": "Однократно", "en": "Once", "kk": "Бір рет"},
		"medit.rec.daily":       {"ru": "Ежедневно", "en": "Daily", "kk": "Күн сайын"},
		"medit.rec.weekly":      {"ru": "Еженедельно", "en": "Weekly", "kk": "Апта сайын"},
		"medit.rec.biweekly":    {"ru": "Раз в 2 недели", "en": "Every 2 weeks", "kk": "2 аптада бір"},
		"medit.rec.monthly":     {"ru": "Ежемесячно", "en": "Monthly", "kk": "Ай сайын"},
		// ── menuText ──
		"medit.menu_series_header": {"ru": "Редактирование всей серии с %[1]s (★ — изменено):\n", "en": "Editing the whole series from %[1]s (★ = changed):\n", "kk": "%[1]s бастап бүкіл серияны өңдеу (★ — өзгертілген):\n"},
		"medit.menu_one_header":    {"ru": "Редактирование встречи (★ — изменено):\n", "en": "Editing the meeting (★ = changed):\n", "kk": "Кездесуді өңдеу (★ — өзгертілген):\n"},
		"medit.lbl_time":           {"ru": "Время", "en": "Time", "kk": "Уақыт"},
		"medit.lbl_datetime":       {"ru": "Дата/время", "en": "Date/time", "kk": "Күн/уақыт"},
		"medit.lbl_dept":           {"ru": "Отдел", "en": "Department", "kk": "Бөлім"},
		"medit.lbl_type":           {"ru": "Тип", "en": "Type", "kk": "Түрі"},
		"medit.lbl_host":           {"ru": "Ведущий", "en": "Host", "kk": "Жүргізуші"},
		"medit.lbl_description":    {"ru": "Описание", "en": "Description", "kk": "Сипаттама"},
		"medit.lbl_recurrence":     {"ru": "Частота", "en": "Recurrence", "kk": "Жиілік"},
	})
}
```

- [ ] **Step 2: Verify build + coverage**

Run: `cd apps/backend && go build ./internal/platform/boti18n/ && go test ./internal/platform/boti18n/ -run TestCatalog_AllKeysHaveAllLangs -v`
Expected: builds clean; coverage test PASS (all `medit.*` keys have ru/en/kk).

- [ ] **Step 3: Commit**

```bash
git add apps/backend/internal/platform/boti18n/catalog_meetingedit.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): meetingedit translation catalog (medit.* ru/en/kk)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

### Task 2: Thread `lang` through `meetingedit` + dispatcher + tests

The threading is atomic (fully connected call graph), so this task replaces `service.go` wholesale, wires the 3 dispatcher sites, and updates/adds tests.

**Files:**
- Modify: `apps/backend/internal/platform/meetingedit/service.go` (full rewrite below)
- Modify: `apps/backend/internal/infrastructure/telegram/multitenant.go` (3 call sites)
- Test: `apps/backend/internal/platform/meetingedit/service_test.go`

**Interfaces:**
- Consumes (Task 1): all `medit.*` keys. (sub-project A) `boti18n.T`, dispatcher `h.resolveLang(ctx, *models.User) string`.
- Produces: `Start(ctx,id,lang)`, `OnCallback(ctx,id,data,lang)`, `OnText(ctx,id,text,lang)` and every helper with a trailing `lang`.

- [ ] **Step 1: Add a localized test to `service_test.go`, run it red**

First read the existing `service_test.go` to learn its fake names (`Backend`/`sessions` fakes, the `New(...)` constructor, and how a session/meeting is seeded). Then add a test that drives `Start` and the menu, asserting ru≠en + an English phrase + a localized button label. Adapt the seeding to the existing fakes:

```go
func TestMeetingEdit_Localized(t *testing.T) {
	// Use the existing fakes/constructor from this file.
	ru := newTestService(t).Start(context.Background(), 1, "ru")
	en := newTestService(t).Start(context.Background(), 1, "en")
	if ru.Text == en.Text {
		t.Fatalf("Start must differ by language; both = %q", ru.Text)
	}
	if !strings.Contains(en.Text, "Pick a meeting") && !strings.Contains(en.Text, "No upcoming") {
		t.Errorf("en Start = %q", en.Text)
	}
}
```

> Replace `newTestService(t)` with whatever the file's existing constructor/fakes are (e.g. `New(fakeBackend{...}, newFakeSessions())`). If the fake `ListEditableMeetings` returns meetings, assert against `"Pick a meeting"`; if empty, against `"No upcoming"`. Pick the branch the existing fakes produce. Add `"strings"`/`"context"` to imports if absent.

Run: `cd apps/backend && go test ./internal/platform/meetingedit/ -run TestMeetingEdit_Localized 2>&1 | head` — Expected: FAIL (arity: `Start` takes no `lang`).

- [ ] **Step 2: Replace `service.go` with the localized version**

Replace the entire body of `apps/backend/internal/platform/meetingedit/service.go` with the following. (Imports add `boti18n`; `fieldPrompts` map becomes the `fieldPrompt(f, lang)` function; everything else is mechanically threaded.)

```go
package meetingedit

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

type Backend interface {
	ListEditableMeetings(ctx context.Context, telegramID int64) ([]postgres.MeetingWithTZ, error)
	UpdateMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in application.UpdateMeetingInput) (postgres.Meeting, error)
	UpdateSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID, in application.SeriesUpdateInput) (int, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
	SearchEmployees(ctx context.Context, organizationID uuid.UUID, query string) ([]postgres.Employee, error)
	AddParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error
	RemoveParticipant(ctx context.Context, organizationID, userID, meetingID uuid.UUID, email string) error
	CancelMeeting(ctx context.Context, organizationID, userID, meetingID uuid.UUID) error
	CancelSeries(ctx context.Context, organizationID, userID, meetingID uuid.UUID) (int, error)
	MeetingUpdateConflicts(ctx context.Context, organizationID, meetingID uuid.UUID, in application.UpdateMeetingInput) ([]application.Conflict, error)
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

func (s *Service) Start(ctx context.Context, telegramID int64, lang string) Reply {
	ms, err := s.backend.ListEditableMeetings(ctx, telegramID)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.list_failed")}
	}
	if len(ms) == 0 {
		return Reply{Text: boti18n.T(lang, "medit.none_editable")}
	}
	var rows [][]Button
	for _, m := range ms {
		rows = append(rows, []Button{{Text: m.Name, Data: "medit:pick:" + m.ID.String()}})
	}
	return Reply{Text: boti18n.T(lang, "medit.pick_meeting"), Keyboard: rows}
}

func (s *Service) OnCallback(ctx context.Context, telegramID int64, data, lang string) (Reply, bool) {
	switch {
	case strings.HasPrefix(data, "medit:pick:"):
		return s.pick(ctx, telegramID, strings.TrimPrefix(data, "medit:pick:"), lang), true
	case strings.HasPrefix(data, "medit:field:"):
		return s.field(ctx, telegramID, strings.TrimPrefix(data, "medit:field:"), lang), true
	case strings.HasPrefix(data, "medit:set:rec:"):
		return s.setRec(ctx, telegramID, strings.TrimPrefix(data, "medit:set:rec:"), lang), true
	case data == "medit:apply":
		return s.apply(ctx, telegramID, lang), true
	case data == "medit:applyforce":
		return s.applyForce(ctx, telegramID, lang), true
	case data == "medit:cancel":
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.cancelled"), Edit: true}, true
	case data == "medit:menu":
		return s.backToMenu(ctx, telegramID, lang), true
	case data == "medit:parts":
		return s.parts(ctx, telegramID, lang), true
	case data == "medit:padd":
		return s.padd(ctx, telegramID, lang), true
	case strings.HasPrefix(data, "medit:padd:"):
		return s.paddPick(ctx, telegramID, strings.TrimPrefix(data, "medit:padd:"), lang), true
	case data == "medit:premc":
		return s.premConfirm(ctx, telegramID, lang), true
	case strings.HasPrefix(data, "medit:prem:"):
		return s.prem(ctx, telegramID, strings.TrimPrefix(data, "medit:prem:"), lang), true
	case data == "medit:scope:one":
		return s.setScope(ctx, telegramID, "one", lang), true
	case data == "medit:scope:series":
		return s.setScope(ctx, telegramID, "series", lang), true
	case data == "medit:delete":
		return s.confirmDelete(ctx, telegramID, lang), true
	case data == "medit:delconf":
		return s.doDelete(ctx, telegramID, lang), true
	}
	return Reply{}, false
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text, lang string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.Step != stepAwaiting {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	if st.AwaitingField == "participant" {
		return s.searchParticipant(ctx, telegramID, st, text, lang), true
	}
	if st.AwaitingField == "datetime" {
		if st.Scope == "series" {
			start, end, perr := parseTimeRange(text)
			if perr != nil {
				return Reply{Text: boti18n.T(lang, "medit.bad_time_range")}, true
			}
			st.Overrides["start"] = start
			st.Overrides["end"] = end
		} else {
			d, start, end, perr := parseDateTime(text)
			if perr != nil {
				return Reply{Text: boti18n.T(lang, "medit.bad_datetime")}, true
			}
			st.Overrides["date"] = d
			st.Overrides["start"] = start
			st.Overrides["end"] = end
		}
	} else {
		if text == "" {
			return Reply{Text: boti18n.T(lang, "medit.empty_value")}, true
		}
		st.Overrides[st.AwaitingField] = text
	}
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)

	return menuReply(*st, false, lang), true
}

func (s *Service) pick(ctx context.Context, telegramID int64, idStr, lang string) Reply {
	mid, err := uuid.Parse(idStr)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.unknown_meeting")}
	}
	ms, err := s.backend.ListEditableMeetings(ctx, telegramID)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.get_meeting_failed")}
	}
	var found *postgres.MeetingWithTZ
	for i := range ms {
		if ms[i].ID == mid {
			found = &ms[i]
			break
		}
	}
	if found == nil || found.OrganizerUserID == nil {
		return Reply{Text: boti18n.T(lang, "medit.not_editable")}
	}
	loc := loadLoc(found.TZ)
	st := State{
		Step:           stepMenu,
		MeetingID:      mid.String(),
		OrganizationID: found.OrganizationID.String(),
		UserID:         found.OrganizerUserID.String(),
		Cur:            snapshot(found.Meeting, loc),
		Overrides:      map[string]string{},
	}
	if found.SeriesID != nil {
		st.SeriesID = found.SeriesID.String()
		_ = s.sessions.Set(ctx, telegramID, st)
		return scopeReply(lang)
	}
	st.Scope = "one"
	_ = s.sessions.Set(ctx, telegramID, st)
	return menuReply(st, true, lang)
}

func (s *Service) field(ctx context.Context, telegramID int64, f, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	if f == "rec" {
		if st.Scope == "series" {
			return menuReply(*st, true, lang)
		}
		return recReply(lang)
	}
	prompt, ok := fieldPrompt(f, lang)
	if !ok {
		return Reply{}
	}
	if f == "datetime" && st.Scope == "series" {
		prompt = boti18n.T(lang, "medit.prompt_time_series")
	}
	st.Step = stepAwaiting
	st.AwaitingField = f
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: prompt}
}

func (s *Service) setRec(ctx context.Context, telegramID int64, val, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	if !meeting.Recurrence(val).Valid() {
		return Reply{}
	}
	st.Overrides["recurrence"] = val
	st.Step = stepMenu
	st.AwaitingField = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, true, lang)
}

func (s *Service) apply(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	if st.SeriesID != "" && st.Scope == "" {
		return Reply{Text: boti18n.T(lang, "medit.choose_scope_first"), Keyboard: scopeReply(lang).Keyboard, Edit: true}
	}
	if len(st.Overrides) == 0 {
		return Reply{Text: boti18n.T(lang, "medit.no_changes"), Keyboard: menuKeyboard(st.Scope, lang), Edit: true}
	}

	if st.Scope != "series" {
		if _, ok := st.Overrides["date"]; ok {
			orgID, _ := uuid.Parse(st.OrganizationID)
			mid, _ := uuid.Parse(st.MeetingID)
			conflicts, cerr := s.backend.MeetingUpdateConflicts(ctx, orgID, mid, toInput(st.Overrides))
			if cerr == nil && len(conflicts) > 0 {
				return Reply{Text: formatConflictWarning(conflicts, lang), Keyboard: conflictKeyboard(lang), Edit: true}
			}
		}
	}
	return s.doApply(ctx, telegramID, st, lang)
}

func (s *Service) applyForce(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	return s.doApply(ctx, telegramID, st, lang)
}

func (s *Service) doApply(ctx context.Context, telegramID int64, st *State, lang string) Reply {
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if st.Scope == "series" {
		n, err := s.backend.UpdateSeries(ctx, orgID, uid, mid, seriesInput(st.Overrides))
		if err != nil {
			switch {
			case errors.Is(err, application.ErrInvalidInput):
				return Reply{Text: boti18n.T(lang, "medit.invalid_data")}
			case errors.Is(err, application.ErrForbidden):
				_ = s.sessions.Del(ctx, telegramID)
				return Reply{Text: boti18n.T(lang, "medit.forbidden")}
			case errors.Is(err, postgres.ErrMeetingNotEditable):
				_ = s.sessions.Del(ctx, telegramID)
				return Reply{Text: boti18n.T(lang, "medit.series_not_editable")}
			default:
				return Reply{Text: boti18n.T(lang, "medit.update_series_failed")}
			}
		}
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.series_updated", n)}
	}
	m, err := s.backend.UpdateMeeting(ctx, orgID, uid, mid, toInput(st.Overrides))
	if err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			return Reply{Text: boti18n.T(lang, "medit.invalid_data")}
		case errors.Is(err, application.ErrForbidden):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.forbidden")}
		case errors.Is(err, postgres.ErrMeetingNotEditable):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.meeting_not_editable")}
		default:
			return Reply{Text: boti18n.T(lang, "medit.update_failed")}
		}
	}
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: boti18n.T(lang, "medit.updated_done") + "\n" + summary(m)}
}

func formatConflictWarning(cs []application.Conflict, lang string) string {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		loc = time.FixedZone("Almaty", 5*60*60)
	}
	var b strings.Builder
	b.WriteString(boti18n.T(lang, "medit.conflict_header"))
	for _, c := range cs {
		fmt.Fprintf(&b, "- %s — «%s» (%s–%s)\n",
			c.PersonName, c.MeetingName, c.Start.In(loc).Format("15:04"), c.End.In(loc).Format("15:04"))
	}
	b.WriteString(boti18n.T(lang, "medit.conflict_apply_q"))
	return b.String()
}

func conflictKeyboard(lang string) [][]Button {
	return [][]Button{{
		{Text: boti18n.T(lang, "medit.btn_apply_yes"), Data: "medit:applyforce"},
		{Text: boti18n.T(lang, "medit.btn_change_time"), Data: "medit:field:datetime"},
	}}
}

func (s *Service) backToMenu(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	return menuReply(*st, true, lang)
}

func (s *Service) parts(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	mid, _ := uuid.Parse(st.MeetingID)
	ps, err := s.backend.ListParticipants(ctx, mid)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.get_parts_failed")}
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
	return partsReply(emails, true, lang)
}

func partsReply(emails []string, edit bool, lang string) Reply {
	var rows [][]Button
	for i, e := range emails {
		rows = append(rows, []Button{{Text: "✖ " + e, Data: fmt.Sprintf("medit:prem:%d", i)}})
	}
	rows = append(rows, []Button{{Text: boti18n.T(lang, "medit.btn_add"), Data: "medit:padd"}})
	rows = append(rows, []Button{{Text: boti18n.T(lang, "medit.btn_back"), Data: "medit:menu"}})
	text := boti18n.T(lang, "medit.parts_title")
	if len(emails) == 0 {
		text = boti18n.T(lang, "medit.parts_empty")
	}
	return Reply{Text: text, Keyboard: rows, Edit: edit}
}

func (s *Service) padd(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	st.Step = stepAwaiting
	st.AwaitingField = "participant"
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "medit.padd_prompt")}
}

func (s *Service) searchParticipant(ctx context.Context, telegramID int64, st *State, query, lang string) Reply {
	orgID, _ := uuid.Parse(st.OrganizationID)
	emps, err := s.backend.SearchEmployees(ctx, orgID, query)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "medit.search_failed")}
	}
	var cands []string
	var rows [][]Button
	seen := map[string]bool{}
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		if len(cands) >= 10 {
			break
		}
		seen[e.Email] = true
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("medit:padd:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if addr, perr := mail.ParseAddress(strings.TrimSpace(query)); perr == nil {
		email := strings.ToLower(addr.Address)
		if !seen[email] {
			rows = append(rows, []Button{{Text: boti18n.T(lang, "medit.btn_add_email", email), Data: fmt.Sprintf("medit:padd:%d", len(cands))}})
			cands = append(cands, email)
		}
	}
	if len(cands) == 0 {
		return Reply{Text: boti18n.T(lang, "medit.search_none")}
	}
	st.PartCands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "medit.padd_pick"), Keyboard: rows}
}

func (s *Service) paddPick(ctx context.Context, telegramID int64, idxStr, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	email, ok := indexInto(st.PartCands, idxStr)
	if !ok {
		return Reply{Text: boti18n.T(lang, "medit.cand_not_found")}
	}
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if err := s.backend.AddParticipant(ctx, orgID, uid, mid, email); err != nil {
		switch {
		case errors.Is(err, application.ErrInvalidInput):
			return Reply{Text: boti18n.T(lang, "medit.already_or_invalid")}
		case errors.Is(err, application.ErrForbidden):
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.forbidden")}
		default:
			return Reply{Text: boti18n.T(lang, "medit.add_failed")}
		}
	}
	return s.parts(ctx, telegramID, lang)
}

func (s *Service) prem(ctx context.Context, telegramID int64, idxStr, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	email, ok := indexInto(st.PartList, idxStr)
	if !ok {
		return Reply{Text: boti18n.T(lang, "medit.part_not_found")}
	}
	st.PendingRemove = email
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{
		Text: boti18n.T(lang, "medit.remove_confirm", email),
		Edit: true,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_yes"), Data: "medit:premc"}},
			{{Text: boti18n.T(lang, "medit.btn_cancel_back"), Data: "medit:parts"}},
		},
	}
}

func (s *Service) premConfirm(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	email := st.PendingRemove
	if email == "" {
		return Reply{Text: boti18n.T(lang, "medit.nothing_to_remove")}
	}
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)
	if err := s.backend.RemoveParticipant(ctx, orgID, uid, mid, email); err != nil {
		if errors.Is(err, application.ErrForbidden) {
			_ = s.sessions.Del(ctx, telegramID)
			return Reply{Text: boti18n.T(lang, "medit.forbidden")}
		}
		return Reply{Text: boti18n.T(lang, "medit.remove_failed")}
	}
	st.PendingRemove = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return s.parts(ctx, telegramID, lang)
}

func indexInto(list []string, idxStr string) (string, bool) {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(list) {
		return "", false
	}
	return list[i], true
}

func fieldPrompt(f, lang string) (string, bool) {
	switch f {
	case "dept":
		return boti18n.T(lang, "medit.prompt_dept"), true
	case "type":
		return boti18n.T(lang, "medit.prompt_type"), true
	case "host":
		return boti18n.T(lang, "medit.prompt_host"), true
	case "description":
		return boti18n.T(lang, "medit.prompt_description"), true
	case "datetime":
		return boti18n.T(lang, "medit.prompt_datetime"), true
	}
	return "", false
}

func menuKeyboard(scope, lang string) [][]Button {
	if scope == "series" {
		return [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_time"), Data: "medit:field:datetime"}},
			{{Text: boti18n.T(lang, "medit.btn_dept"), Data: "medit:field:dept"}, {Text: boti18n.T(lang, "medit.btn_type"), Data: "medit:field:type"}},
			{{Text: boti18n.T(lang, "medit.btn_host"), Data: "medit:field:host"}, {Text: boti18n.T(lang, "medit.btn_description"), Data: "medit:field:description"}},
			{{Text: boti18n.T(lang, "medit.btn_delete"), Data: "medit:delete"}},
			{{Text: boti18n.T(lang, "medit.btn_apply"), Data: "medit:apply"}, {Text: boti18n.T(lang, "medit.btn_cancel"), Data: "medit:cancel"}},
		}
	}
	return [][]Button{
		{{Text: boti18n.T(lang, "medit.btn_datetime"), Data: "medit:field:datetime"}},
		{{Text: boti18n.T(lang, "medit.btn_dept"), Data: "medit:field:dept"}, {Text: boti18n.T(lang, "medit.btn_type"), Data: "medit:field:type"}},
		{{Text: boti18n.T(lang, "medit.btn_host"), Data: "medit:field:host"}, {Text: boti18n.T(lang, "medit.btn_description"), Data: "medit:field:description"}},
		{{Text: boti18n.T(lang, "medit.btn_recurrence"), Data: "medit:field:rec"}},
		{{Text: boti18n.T(lang, "medit.btn_participants"), Data: "medit:parts"}},
		{{Text: boti18n.T(lang, "medit.btn_delete"), Data: "medit:delete"}},
		{{Text: boti18n.T(lang, "medit.btn_apply"), Data: "medit:apply"}, {Text: boti18n.T(lang, "medit.btn_cancel"), Data: "medit:cancel"}},
	}
}

func scopeReply(lang string) Reply {
	return Reply{
		Text: boti18n.T(lang, "medit.scope_q"),
		Edit: true,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_scope_one"), Data: "medit:scope:one"}},
			{{Text: boti18n.T(lang, "medit.btn_scope_series"), Data: "medit:scope:series"}},
		},
	}
}

func (s *Service) confirmDelete(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	text := boti18n.T(lang, "medit.delete_one_q")
	if st.Scope == "series" {
		text = boti18n.T(lang, "medit.delete_series_q")
	}
	return Reply{
		Text: text,
		Edit: true,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "medit.btn_delete_yes"), Data: "medit:delconf"}},
			{{Text: boti18n.T(lang, "medit.btn_cancel_back"), Data: "medit:menu"}},
		},
	}
}

func (s *Service) doDelete(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	orgID, _ := uuid.Parse(st.OrganizationID)
	uid, _ := uuid.Parse(st.UserID)
	mid, _ := uuid.Parse(st.MeetingID)

	if st.Scope == "series" {
		n, err := s.backend.CancelSeries(ctx, orgID, uid, mid)
		if err != nil {
			return s.deleteErrReply(ctx, telegramID, err, lang)
		}
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.series_deleted", n)}
	}
	if err := s.backend.CancelMeeting(ctx, orgID, uid, mid); err != nil {
		return s.deleteErrReply(ctx, telegramID, err, lang)
	}
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: boti18n.T(lang, "medit.meeting_deleted")}
}

func (s *Service) deleteErrReply(ctx context.Context, telegramID int64, err error, lang string) Reply {
	switch {
	case errors.Is(err, application.ErrForbidden):
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.forbidden")}
	case errors.Is(err, postgres.ErrMeetingNotEditable):
		_ = s.sessions.Del(ctx, telegramID)
		return Reply{Text: boti18n.T(lang, "medit.meeting_unavailable")}
	default:
		return Reply{Text: boti18n.T(lang, "medit.delete_failed")}
	}
}

func (s *Service) setScope(ctx context.Context, telegramID int64, scope, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "medit.session_expired")}
	}
	st.Scope = scope
	_ = s.sessions.Set(ctx, telegramID, *st)
	return menuReply(*st, true, lang)
}

func recReply(lang string) Reply {
	return Reply{Text: boti18n.T(lang, "medit.pick_recurrence"), Edit: true, Keyboard: [][]Button{
		{{Text: boti18n.T(lang, "medit.rec.once"), Data: "medit:set:rec:once"}},
		{{Text: boti18n.T(lang, "medit.rec.daily"), Data: "medit:set:rec:daily"}},
		{{Text: boti18n.T(lang, "medit.rec.weekly"), Data: "medit:set:rec:weekly"}},
		{{Text: boti18n.T(lang, "medit.rec.biweekly"), Data: "medit:set:rec:biweekly"}},
		{{Text: boti18n.T(lang, "medit.rec.monthly"), Data: "medit:set:rec:monthly"}},
	}}
}

func menuReply(st State, edit bool, lang string) Reply {
	return Reply{Text: menuText(st, lang), Keyboard: menuKeyboard(st.Scope, lang), Edit: edit}
}

func menuText(st State, lang string) string {
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
	if st.Scope == "series" {
		b.WriteString(boti18n.T(lang, "medit.menu_series_header", st.Cur["date"]))
		tmark := ""
		if _, ok := st.Overrides["start"]; ok {
			tmark = " ★"
		}
		fmt.Fprintf(&b, "• %s: %s–%s%s\n", boti18n.T(lang, "medit.lbl_time"), eff("start"), eff("end"), tmark)
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_dept"), eff("dept"), mark("dept"))
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_type"), eff("type"), mark("type"))
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_host"), eff("host"), mark("host"))
		fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_description"), eff("description"), mark("description"))
		return b.String()
	}
	b.WriteString(boti18n.T(lang, "medit.menu_one_header"))
	dmark := ""
	if _, ok := st.Overrides["date"]; ok {
		dmark = " ★"
	}
	fmt.Fprintf(&b, "• %s: %s %s–%s%s\n", boti18n.T(lang, "medit.lbl_datetime"), eff("date"), eff("start"), eff("end"), dmark)
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_dept"), eff("dept"), mark("dept"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_type"), eff("type"), mark("type"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_host"), eff("host"), mark("host"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_description"), eff("description"), mark("description"))
	fmt.Fprintf(&b, "• %s: %s%s\n", boti18n.T(lang, "medit.lbl_recurrence"), recLabel(eff("recurrence"), lang), mark("recurrence"))
	return b.String()
}

func recLabel(v, lang string) string {
	if v == "once" || v == "" {
		return boti18n.T(lang, "medit.rec.once")
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

func seriesInput(ov map[string]string) application.SeriesUpdateInput {
	var in application.SeriesUpdateInput
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

> The package var `fieldPrompts` is removed and replaced by the `fieldPrompt(f, lang)` function above. Confirm no other file references `fieldPrompts` (it is package-private to `meetingedit`; a quick `grep fieldPrompts` should show only this file — now gone).

- [ ] **Step 3: Run the localized test + existing tests; fix existing arity**

Run: `cd apps/backend && go test ./internal/platform/meetingedit/ 2>&1 | head -40`
Expected: compile errors in `service_test.go` for any existing test calling `Start`/`OnText`/`OnCallback` (or a helper like `menuReply`/`partsReply`/`menuText`) without `lang`. For each, add the `lang` argument — pass `"ru"` — and keep the existing assertion (ru output is unchanged). Re-run until the package is green, including `TestMeetingEdit_Localized`.

- [ ] **Step 4: Update the 3 dispatcher call sites in `multitenant.go`**

- Line 85 (OnText): `if reply, handled := h.editor.OnText(ctx, from.ID, text, h.resolveLang(ctx, from)); handled {`
- Line 130 (Start): `h.sendEditorReply(ctx, b, chatID, 0, h.editor.Start(ctx, from.ID, h.resolveLang(ctx, from)))`
- Line 198 (OnCallback): `if reply, handled := h.editor.OnCallback(ctx, cq.From.ID, cq.Data, h.resolveLang(ctx, &cq.From)); handled && cq.Message.Message != nil {`

- [ ] **Step 5: Build + vet + coverage**

Run: `cd apps/backend && go build ./... && go vet ./internal/platform/meetingedit/ ./internal/infrastructure/telegram/ && go test ./internal/platform/meetingedit/ ./internal/platform/boti18n/`
Expected: build clean, vet clean, both test packages PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/backend/internal/platform/meetingedit/service.go apps/backend/internal/platform/meetingedit/service_test.go apps/backend/internal/infrastructure/telegram/multitenant.go
git commit -m "$(cat <<'EOF'
feat(bot-i18n): localize meetingedit FSM (ru/en/kk)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_019dEzoeRPoXyA4wm1kuDfti
EOF
)"
```

---

## Self-Review

**Spec coverage:**
- `medit.*` catalog ru/en/kk → Task 1. ✓
- `lang` threaded through all entry methods + every text-emitting helper; dispatcher passes resolveLang (3 sites, callback `&cq.From`) → Task 2 (full rewrite + dispatcher). ✓
- Reply text AND button labels localized; `menuText` field summary + headers localized; conflict warning localized → Task 2 (`menuKeyboard`/`scopeReply`/`recReply`/`partsReply`/`conflictKeyboard`/`menuText`/`formatConflictWarning`). ✓
- Parse errors localized at FSM level (`medit.bad_datetime`/`medit.bad_time_range`); `parse.go` untouched → Task 2 OnText. ✓
- `fieldPrompts` map → `fieldPrompt(f, lang)` function (values now localized) → Task 2. ✓
- Out of scope kept as-is: domain `.Label()` for non-`once` (recLabel), `loadLoc`, `summary`, glyphs/dates → Task 2 (recLabel only localizes the `once` branch; summary/loadLoc/snapshot unchanged). ✓
- Coverage test enforces all `medit.*` keys → Task 1 Step 2 + Task 2 Step 5. ✓

**Placeholder scan:** No TBD/TODO. Task 2 provides the complete file; the only adapt-on-the-spot is wiring `TestMeetingEdit_Localized` to the file's existing fakes (explicitly flagged) and adding `"ru"` to whatever existing tests call now-localized funcs (mechanical, the failing compile lists them).

**Type consistency:** Every method/helper signature in the rewritten `service.go` carries `lang` consistently and every internal call passes it (verified by the file compiling as a unit in Step 5). Catalog keys referenced in `service.go` (`medit.*`) all exist in `catalog_meetingedit.go` from Task 1. Dispatcher passes `*models.User` (`from` / `&cq.From`) to `resolveLang`, matching its signature.
