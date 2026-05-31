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
	// New message (not Edit): we can't turn the user's text message into the menu,
	// so the refreshed menu is sent below. The previous menu's buttons still work
	// (FSM state lives in Redis, not the message).
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
	// IDs come from our own session (set in pick from uuid.UUID.String()); a parse
	// failure would yield a zero UUID that UpdateMeeting rejects as ErrForbidden —
	// safe degradation, so the parse errors are intentionally ignored.
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
