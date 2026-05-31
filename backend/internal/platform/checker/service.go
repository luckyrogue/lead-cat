package checker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Backend is the application surface the checker FSM needs (satisfied by *application.Services).
type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error)
	FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
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

// Start handles /checker: prompts for the first participant.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepParticipants})
	return Reply{Text: "Поиск общего свободного времени.\nВведи имя или email участника:"}
}

// OnText feeds free text into the active step. bool=false when no active session.
func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepParticipants:
		return s.search(ctx, telegramID, st, text), true
	case stepRange:
		return s.setRange(ctx, telegramID, st, text), true
	}
	return Reply{}, false
}

// OnCallback handles chk:* taps. bool=false for non-chk data.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	if !strings.HasPrefix(data, "chk:") {
		return Reply{}, false
	}
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: "Сессия истекла. Начни заново: /checker"}, true
	}
	switch {
	case strings.HasPrefix(data, "chk:add:"):
		return s.add(ctx, telegramID, st, strings.TrimPrefix(data, "chk:add:")), true
	case data == "chk:done":
		return s.done(ctx, telegramID, st), true
	case strings.HasPrefix(data, "chk:dur:"):
		return s.duration(ctx, telegramID, st, strings.TrimPrefix(data, "chk:dur:")), true
	}
	return Reply{}, true
}

func (s *Service) search(ctx context.Context, telegramID int64, st *State, query string) Reply {
	emps, err := s.backend.SearchEmployeesGlobal(ctx, query)
	if err != nil {
		return Reply{Text: "Не удалось выполнить поиск, попробуй ещё раз:"}
	}
	var rows [][]Button
	var cands []string
	seen := map[string]bool{}
	for _, e := range emps {
		if e.Email == "" || seen[e.Email] {
			continue
		}
		seen[e.Email] = true
		rows = append(rows, []Button{{Text: e.FullName + " — " + e.Email, Data: fmt.Sprintf("chk:add:%d", len(cands))}})
		cands = append(cands, e.Email)
	}
	if len(cands) == 0 {
		return Reply{Text: "Ничего не найдено. Введи другой запрос:"}
	}
	st.Cands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	if len(st.Emails) > 0 {
		rows = append(rows, []Button{{Text: "Готово ✅", Data: "chk:done"}})
	}
	return Reply{Text: "Выбери участника (можно несколько):", Keyboard: rows}
}

func (s *Service) add(ctx context.Context, telegramID int64, st *State, idxStr string) Reply {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(st.Cands) {
		return Reply{Text: "Не найдено, поищи ещё раз:"}
	}
	email := st.Cands[i]
	for _, e := range st.Emails {
		if e == email {
			return Reply{Text: "Уже добавлен. Ищи ещё или нажми «Готово».",
				Keyboard: [][]Button{{{Text: "Готово ✅", Data: "chk:done"}}}}
		}
	}
	st.Emails = append(st.Emails, email)
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{
		Text:     fmt.Sprintf("Добавлен: %s\nУчастников: %d. Ищи ещё или нажми «Готово».", email, len(st.Emails)),
		Keyboard: [][]Button{{{Text: "Готово ✅", Data: "chk:done"}}},
	}
}

func (s *Service) done(ctx context.Context, telegramID int64, st *State) Reply {
	if len(st.Emails) == 0 {
		return Reply{Text: "Добавь хотя бы одного участника."}
	}
	st.Step = stepRange
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Введи диапазон дат: ГГГГ-ММ-ДД..ГГГГ-ММ-ДД"}
}

func (s *Service) setRange(ctx context.Context, telegramID int64, st *State, text string) Reply {
	from, to, err := parseRange(text, almaty())
	if err != nil {
		return Reply{Text: err.Error() + "\nПопробуй ещё раз:"}
	}
	st.From = from.Format("2006-01-02")
	st.To = to.Format("2006-01-02")
	st.Step = stepDuration
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Выбери длительность встречи:", Keyboard: durationKeyboard()}
}

func (s *Service) duration(ctx context.Context, telegramID int64, st *State, durStr string) Reply {
	durMins, err := strconv.Atoi(durStr)
	if err != nil || durMins <= 0 {
		return Reply{Text: "Неверная длительность."}
	}
	loc := almaty()
	from, _ := time.ParseInLocation("2006-01-02", st.From, loc)
	toIncl, _ := time.ParseInLocation("2006-01-02", st.To, loc)
	slots, err := s.backend.FreeSlots(ctx, st.Emails, from, toIncl.AddDate(0, 0, 1), durMins)
	if err != nil {
		return Reply{Text: "Не удалось выполнить поиск, попробуй позже."}
	}
	n := len(st.Emails)
	_ = s.sessions.Del(ctx, telegramID)
	if len(slots) == 0 {
		return Reply{Text: "Общих свободных слотов в выбранном диапазоне не найдено.\n" +
			"Попробуй: расширить диапазон дат / уменьшить длительность / изменить состав участников."}
	}
	return Reply{Text: formatSlots(slots, n, loc)}
}

func durationKeyboard() [][]Button {
	return [][]Button{
		{{Text: "15 мин", Data: "chk:dur:15"}, {Text: "30 мин", Data: "chk:dur:30"}, {Text: "45 мин", Data: "chk:dur:45"}},
		{{Text: "1 час", Data: "chk:dur:60"}, {Text: "1.5 часа", Data: "chk:dur:90"}, {Text: "2 часа", Data: "chk:dur:120"}},
	}
}

func formatSlots(slots []application.FreeSlot, n int, loc *time.Location) string {
	var b strings.Builder
	fmt.Fprintf(&b, "✅ Общее свободное время для %d участников:\n\n", n)
	for _, sl := range slots {
		fmt.Fprintf(&b, "📅 %s — %s–%s (%d мин свободно)\n",
			dayLabel(sl.Day, loc), sl.Start.In(loc).Format("15:04"), sl.End.In(loc).Format("15:04"), sl.Mins)
	}
	return b.String()
}
