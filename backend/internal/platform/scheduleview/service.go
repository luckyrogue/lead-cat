package scheduleview

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Backend is the application surface the FSM needs (satisfied by *application.Services).
type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error)
	EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]postgres.Meeting, error)
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

// Start handles /schedule: prompts for the employee to look up.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepAwait, AwaitingKind: awaitSearch})
	return Reply{Text: "Чьё расписание показать? Введи email сотрудника или часть имени:"}
}

// OnCallback handles sched:* taps. The bool is false for non-sched data.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	switch {
	case strings.HasPrefix(data, "sched:pick:"):
		return s.pick(ctx, telegramID, strings.TrimPrefix(data, "sched:pick:")), true
	case data == "sched:periods":
		return s.periods(ctx, telegramID), true
	case data == "sched:back":
		return s.Start(ctx, telegramID), true
	case strings.HasPrefix(data, "sched:d:"):
		return s.period(ctx, telegramID, strings.TrimPrefix(data, "sched:d:")), true
	}
	return Reply{}, false
}

// OnText feeds free text into the active awaiting state. The bool is false when
// there is no active schedule session.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.Step != stepAwait {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	switch st.AwaitingKind {
	case awaitSearch:
		return s.search(ctx, telegramID, st, text), true
	case awaitDate:
		d, perr := parseDate(text, almaty())
		if perr != nil {
			return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
		}
		return s.list(ctx, st, d, d.AddDate(0, 0, 1), text), true
	case awaitRange:
		from, to, perr := parseRange(text, almaty())
		if perr != nil {
			return Reply{Text: perr.Error() + "\nПопробуй ещё раз:"}, true
		}
		return s.list(ctx, st, from, to, text), true
	}
	return Reply{}, false
}

func (s *Service) search(ctx context.Context, telegramID int64, st *State, query string) Reply {
	emps, err := s.backend.SearchEmployeesGlobal(ctx, query)
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
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("sched:pick:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if addr, perr := mail.ParseAddress(query); perr == nil {
		email := strings.ToLower(addr.Address)
		if !seen[email] {
			rows = append(rows, []Button{{Text: "Расписание " + email, Data: fmt.Sprintf("sched:pick:%d", len(cands))}})
			cands = append(cands, email)
		}
	}
	if len(cands) == 0 {
		return Reply{Text: "Ничего не найдено. Введи корректный email или часть имени:"}
	}
	st.Cands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Выбери сотрудника:", Keyboard: rows}
}

func (s *Service) pick(ctx context.Context, telegramID int64, idxStr string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /schedule"}
	}
	email, ok := indexInto(st.Cands, idxStr)
	if !ok {
		return Reply{Text: "Не найдено, начни заново: /schedule"}
	}
	st.EmployeeEmail = email
	st.AwaitingKind = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return periodReply(email, true)
}

func (s *Service) periods(ctx context.Context, telegramID int64) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.EmployeeEmail == "" {
		return Reply{Text: "Сессия истекла. Начни заново: /schedule"}
	}
	return periodReply(st.EmployeeEmail, true)
}

func (s *Service) period(ctx context.Context, telegramID int64, kind string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.EmployeeEmail == "" {
		return Reply{Text: "Сессия истекла. Начни заново: /schedule"}
	}
	switch kind {
	case "date":
		st.AwaitingKind = awaitDate
		_ = s.sessions.Set(ctx, telegramID, *st)
		return Reply{Text: "Введи дату ГГГГ-ММ-ДД:"}
	case "range":
		st.AwaitingKind = awaitRange
		_ = s.sessions.Set(ctx, telegramID, *st)
		return Reply{Text: "Введи диапазон ГГГГ-ММ-ДД..ГГГГ-ММ-ДД:"}
	}
	from, to, ok := dayWindow(time.Now(), kind, almaty())
	if !ok {
		return Reply{}
	}
	return s.list(ctx, st, from, to, periodLabel(kind))
}

func (s *Service) list(ctx context.Context, st *State, from, to time.Time, period string) Reply {
	ms, err := s.backend.EmployeeSchedule(ctx, st.EmployeeEmail, from, to)
	if err != nil {
		return Reply{Text: "Не удалось получить расписание, попробуй позже."}
	}
	text := scheduleText(st.EmployeeEmail, period, ms, time.Now(), almaty())
	return Reply{Text: text, Keyboard: [][]Button{{{Text: "⬅ Периоды", Data: "sched:periods"}}}}
}

func periodReply(email string, edit bool) Reply {
	return Reply{
		Text: "Расписание " + email + ". Выбери период:",
		Edit: edit,
		Keyboard: [][]Button{
			{{Text: "Сегодня", Data: "sched:d:today"}, {Text: "Завтра", Data: "sched:d:tomorrow"}},
			{{Text: "Все предстоящие", Data: "sched:d:upcoming"}},
			{{Text: "Конкретная дата", Data: "sched:d:date"}, {Text: "Диапазон", Data: "sched:d:range"}},
			{{Text: "⬅ Другой сотрудник", Data: "sched:back"}},
		},
	}
}

func periodLabel(kind string) string {
	switch kind {
	case "today":
		return "сегодня"
	case "tomorrow":
		return "завтра"
	case "upcoming":
		return "все предстоящие"
	}
	return kind
}

func scheduleText(email, period string, ms []postgres.Meeting, now time.Time, loc *time.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Расписание %s: %s\n", email, period)
	if len(ms) == 0 {
		b.WriteString("Встреч нет.")
		return b.String()
	}
	for _, m := range ms {
		s := m.StartsAt.In(loc)
		e := m.EndsAt.In(loc)
		fmt.Fprintf(&b, "%s «%s» — %s %s–%s\n", statusEmoji(m.StartsAt, now), m.Name, s.Format("02.01.2006"), s.Format("15:04"), e.Format("15:04"))
	}
	return b.String()
}

func indexInto(list []string, idxStr string) (string, bool) {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(list) {
		return "", false
	}
	return list[i], true
}

func almaty() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.UTC
	}
	return loc
}
