package checker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

type Backend interface {
	SearchEmployeesGlobal(ctx context.Context, query string) ([]postgres.Employee, error)
	FreeSlots(ctx context.Context, requesterEmail string, emails []string, from, to time.Time, durMins int) ([]application.FreeSlot, error)
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
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepParticipants})
	return Reply{Text: boti18n.T(lang, "checker.start")}
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text, lang string, loc *time.Location) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{}, false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepParticipants:
		return s.search(ctx, telegramID, st, text, lang), true
	case stepRange:
		return s.setRange(ctx, telegramID, st, text, lang, loc), true
	}
	return Reply{}, false
}

func (s *Service) OnCallback(ctx context.Context, telegramID int64, data, lang string, loc *time.Location) (Reply, bool) {
	if !strings.HasPrefix(data, "chk:") {
		return Reply{}, false
	}
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return Reply{Text: boti18n.T(lang, "checker.session_expired")}, true
	}
	switch {
	case strings.HasPrefix(data, "chk:add:"):
		return s.add(ctx, telegramID, st, strings.TrimPrefix(data, "chk:add:"), lang), true
	case data == "chk:done":
		return s.done(ctx, telegramID, st, lang), true
	case strings.HasPrefix(data, "chk:dur:"):
		return s.duration(ctx, telegramID, st, strings.TrimPrefix(data, "chk:dur:"), lang, loc), true
	}
	return Reply{}, true
}

func (s *Service) search(ctx context.Context, telegramID int64, st *State, query, lang string) Reply {
	emps, err := s.backend.SearchEmployeesGlobal(ctx, query)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "checker.search_failed")}
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
		return Reply{Text: boti18n.T(lang, "checker.none_found")}
	}
	st.Cands = cands
	_ = s.sessions.Set(ctx, telegramID, *st)
	if len(st.Emails) > 0 {
		rows = append(rows, []Button{{Text: boti18n.T(lang, "checker.btn_done"), Data: "chk:done"}})
	}
	return Reply{Text: boti18n.T(lang, "checker.pick"), Keyboard: rows}
}

func (s *Service) add(ctx context.Context, telegramID int64, st *State, idxStr, lang string) Reply {
	i, err := strconv.Atoi(idxStr)
	if err != nil || i < 0 || i >= len(st.Cands) {
		return Reply{Text: boti18n.T(lang, "checker.not_found_retry")}
	}
	email := st.Cands[i]
	for _, e := range st.Emails {
		if e == email {
			return Reply{Text: boti18n.T(lang, "checker.already_added"),
				Keyboard: [][]Button{{{Text: boti18n.T(lang, "checker.btn_done"), Data: "chk:done"}}}}
		}
	}
	st.Emails = append(st.Emails, email)
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{
		Text:     boti18n.T(lang, "checker.added", email, len(st.Emails)),
		Keyboard: [][]Button{{{Text: boti18n.T(lang, "checker.btn_done"), Data: "chk:done"}}},
	}
}

func (s *Service) done(ctx context.Context, telegramID int64, st *State, lang string) Reply {
	if len(st.Emails) == 0 {
		return Reply{Text: boti18n.T(lang, "checker.need_one")}
	}
	st.Step = stepRange
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "checker.enter_range")}
}

func (s *Service) setRange(ctx context.Context, telegramID int64, st *State, text, lang string, loc *time.Location) Reply {
	from, to, err := parseRange(text, loc)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "checker.bad_range")}
	}
	st.From = from.Format("2006-01-02")
	st.To = to.Format("2006-01-02")
	st.Step = stepDuration
	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: boti18n.T(lang, "checker.pick_duration"), Keyboard: durationKeyboard(lang)}
}

func (s *Service) duration(ctx context.Context, telegramID int64, st *State, durStr, lang string, loc *time.Location) Reply {
	durMins, err := strconv.Atoi(durStr)
	if err != nil || durMins <= 0 {
		return Reply{Text: boti18n.T(lang, "checker.bad_duration")}
	}
	from, _ := time.ParseInLocation("2006-01-02", st.From, loc)
	toIncl, _ := time.ParseInLocation("2006-01-02", st.To, loc)
	slots, err := s.backend.FreeSlots(ctx, "", st.Emails, from, toIncl.AddDate(0, 0, 1), durMins)
	if err != nil {
		return Reply{Text: boti18n.T(lang, "checker.search_failed_later")}
	}
	n := len(st.Emails)
	_ = s.sessions.Del(ctx, telegramID)
	if len(slots) == 0 {
		return Reply{Text: boti18n.T(lang, "checker.no_slots")}
	}
	return Reply{Text: formatSlots(slots, n, loc, lang)}
}

func durationKeyboard(lang string) [][]Button {
	return [][]Button{
		{{Text: boti18n.T(lang, "checker.dur.15m"), Data: "chk:dur:15"}, {Text: boti18n.T(lang, "checker.dur.30m"), Data: "chk:dur:30"}, {Text: boti18n.T(lang, "checker.dur.45m"), Data: "chk:dur:45"}},
		{{Text: boti18n.T(lang, "checker.dur.1h"), Data: "chk:dur:60"}, {Text: boti18n.T(lang, "checker.dur.1_5h"), Data: "chk:dur:90"}, {Text: boti18n.T(lang, "checker.dur.2h"), Data: "chk:dur:120"}},
	}
}

func formatSlots(slots []application.FreeSlot, n int, loc *time.Location, lang string) string {
	var b strings.Builder
	b.WriteString(boti18n.T(lang, "checker.slots_header", n))
	for _, sl := range slots {
		fmt.Fprintf(&b, "📅 %s — %s–%s (%s)\n",
			dayLabel(sl.Day, loc), sl.Start.In(loc).Format("15:04"), sl.End.In(loc).Format("15:04"),
			boti18n.T(lang, "checker.slot_mins", sl.Mins))
	}
	return b.String()
}
