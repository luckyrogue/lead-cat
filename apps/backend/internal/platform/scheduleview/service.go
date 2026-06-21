package scheduleview

import (
	"context"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

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

func (s *Service) Start(ctx context.Context, telegramID int64, lang string) Reply {
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepAwait, AwaitingKind: awaitSearch})
	return Reply{Text: boti18n.T(lang, "sched.start")}
}

func (s *Service) OnCallback(ctx context.Context, telegramID int64, data, lang string, loc *time.Location) (Reply, bool) {
	switch {
	case strings.HasPrefix(data, "sched:pick:"):
		return s.pick(ctx, telegramID, strings.TrimPrefix(data, "sched:pick:"), lang), true
	case data == "sched:periods":
		return s.periods(ctx, telegramID, lang), true
	case data == "sched:back":
		return s.Start(ctx, telegramID, lang), true
	case strings.HasPrefix(data, "sched:d:"):
		return s.period(ctx, telegramID, strings.TrimPrefix(data, "sched:d:"), lang, loc), true
	}
	return Reply{}, false
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text, lang string, loc *time.Location) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.Step != stepAwait {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	switch st.AwaitingKind {
	case awaitSearch:
		return s.search(ctx, telegramID, st, text, lang), true
	case awaitDate:
		d, perr := parseDate(text, loc)
		if perr != nil {
			return Reply{Text: boti18n.T(lang, "sched.bad_date")}, true
		}
		return s.list(ctx, st, d, d.AddDate(0, 0, 1), text, false, lang, loc), true
	case awaitRange:
		from, to, perr := parseRange(text, loc)
		if perr != nil {
			return Reply{Text: boti18n.T(lang, "sched.bad_range")}, true
		}
		return s.list(ctx, st, from, to, text, false, lang, loc), true
	}
	return Reply{}, false
}

func (s *Service) search(ctx context.Context, telegramID int64, st *State, query, lang string) Reply {
	emps, err := s.backend.SearchEmployeesGlobal(ctx, query)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "sched.search_failed")}
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
			rows = append(rows, []Button{{Text: boti18n.T(lang, "sched.schedule_btn", email), Data: fmt.Sprintf("sched:pick:%d", len(cands))}})
			cands = append(cands, email)
		}
	}
	if len(cands) == 0 {
		return Reply{Text: boti18n.T(lang, "sched.none_found")}
	}
	st.Cands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "sched.pick"), Keyboard: rows}
}

func (s *Service) pick(ctx context.Context, telegramID int64, idxStr, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "sched.session_expired")}
	}
	email, ok := indexInto(st.Cands, idxStr)
	if !ok {
		return Reply{Text: boti18n.T(lang, "sched.not_found_retry")}
	}
	st.EmployeeEmail = email
	st.AwaitingKind = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return periodReply(email, true, lang)
}

func (s *Service) periods(ctx context.Context, telegramID int64, lang string) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.EmployeeEmail == "" {
		return Reply{Text: boti18n.T(lang, "sched.session_expired")}
	}
	st.AwaitingKind = ""
	_ = s.sessions.Set(ctx, telegramID, *st)
	return periodReply(st.EmployeeEmail, true, lang)
}

func (s *Service) period(ctx context.Context, telegramID int64, kind, lang string, loc *time.Location) Reply {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil || st.EmployeeEmail == "" {
		return Reply{Text: boti18n.T(lang, "sched.session_expired")}
	}
	switch kind {
	case "date":
		st.AwaitingKind = awaitDate
		_ = s.sessions.Set(ctx, telegramID, *st)
		return Reply{Text: boti18n.T(lang, "sched.enter_date")}
	case "range":
		st.AwaitingKind = awaitRange
		_ = s.sessions.Set(ctx, telegramID, *st)
		return Reply{Text: boti18n.T(lang, "sched.enter_range")}
	}
	from, to, ok := dayWindow(time.Now(), kind, loc)
	if !ok {
		return Reply{}
	}
	return s.list(ctx, st, from, to, periodLabel(kind, lang), true, lang, loc)
}

func (s *Service) list(ctx context.Context, st *State, from, to time.Time, period string, edit bool, lang string, loc *time.Location) Reply {
	ms, err := s.backend.EmployeeSchedule(ctx, st.EmployeeEmail, from, to)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "sched.get_failed")}
	}
	text := scheduleText(st.EmployeeEmail, period, ms, time.Now(), loc, lang)
	return Reply{Text: text, Keyboard: [][]Button{{{Text: boti18n.T(lang, "sched.btn.periods"), Data: "sched:periods"}}}, Edit: edit}
}

func periodReply(email string, edit bool, lang string) Reply {
	return Reply{
		Text: boti18n.T(lang, "sched.pick_period", email),
		Edit: edit,
		Keyboard: [][]Button{
			{{Text: boti18n.T(lang, "sched.btn.today"), Data: "sched:d:today"}, {Text: boti18n.T(lang, "sched.btn.tomorrow"), Data: "sched:d:tomorrow"}},
			{{Text: boti18n.T(lang, "sched.btn.upcoming"), Data: "sched:d:upcoming"}},
			{{Text: boti18n.T(lang, "sched.btn.date"), Data: "sched:d:date"}, {Text: boti18n.T(lang, "sched.btn.range"), Data: "sched:d:range"}},
			{{Text: boti18n.T(lang, "sched.btn.back"), Data: "sched:back"}},
		},
	}
}

func periodLabel(kind, lang string) string {
	switch kind {
	case "today":
		return boti18n.T(lang, "sched.period.today")
	case "tomorrow":
		return boti18n.T(lang, "sched.period.tomorrow")
	case "upcoming":
		return boti18n.T(lang, "sched.period.upcoming")
	}
	return kind
}

func scheduleText(email, period string, ms []postgres.Meeting, now time.Time, loc *time.Location, lang string) string {
	var b strings.Builder
	b.WriteString(boti18n.T(lang, "sched.header", email, period))
	if len(ms) == 0 {
		b.WriteString(boti18n.T(lang, "sched.no_meetings"))
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
