package scheduler_agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application"
)

const maxIterations = 6

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	planner  application.Planner
	backend  Backend
	booker   Booker
	sessions sessions
	tools    []application.AgentTool
	bookMu   sync.Mutex
}

func New(planner application.Planner, backend Backend, sess sessions) *Service {
	return NewWithBooker(planner, backend, nil, sess)
}

func NewWithBooker(planner application.Planner, backend Backend, booker Booker, sess sessions) *Service {
	return &Service{planner: planner, backend: backend, booker: booker, sessions: sess, tools: ToolSpecs()}
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (Reply, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		st = &State{}
	}
	st.History = append(st.History, application.AgentMessage{Role: "user", Text: text})

	for i := 0; i < maxIterations; i++ {
		turn, perr := s.planner.Plan(ctx, systemPrompt, st.History, s.tools)
		if perr != nil {
			return Reply{Text: "Не получилось обработать запрос, попробуй ещё раз чуть позже 🐾"}, true
		}
		if len(turn.ToolCalls) == 0 {
			st.History = append(st.History, application.AgentMessage{Role: "assistant", Text: turn.Text})
			_ = s.sessions.Set(ctx, telegramID, *st)
			return Reply{Text: turn.Text}, true
		}
		st.History = append(st.History, application.AgentMessage{
			Role:              "assistant",
			Text:              turn.Text,
			Thinking:          turn.Thinking,
			ThinkingSignature: turn.ThinkingSignature,
			ToolCalls:         turn.ToolCalls,
		})
		// Build one tool_result user message covering every tool_use in the turn.
		var pending *PendingBooking
		results := make([]application.AgentToolResult, 0, len(turn.ToolCalls))
		for _, call := range turn.ToolCalls {
			if call.Name == "propose_meeting" {
				pb, perr := parsePending(call.Input)
				if perr != nil {
					results = append(results, application.AgentToolResult{ID: call.ID, Content: perr.Error(), IsError: true})
					continue
				}
				pb2 := pb
				pending = &pb2
				results = append(results, application.AgentToolResult{ID: call.ID, Content: "Предложение показано пользователю, ждём подтверждения."})
				continue
			}
			out, derr := Dispatch(ctx, s.backend, call.Name, call.Input)
			if derr != nil {
				results = append(results, application.AgentToolResult{ID: call.ID, Content: derr.Error(), IsError: true})
				continue
			}
			results = append(results, application.AgentToolResult{ID: call.ID, Content: out})
		}
		st.History = append(st.History, application.AgentMessage{Role: "user", ToolResults: results})
		if pending != nil {
			st.Pending = pending
			_ = s.sessions.Set(ctx, telegramID, *st)
			return Reply{
				Text: describeBooking(*pending),
				Keyboard: [][]Button{{
					{Text: "Подтвердить ✅", Data: "agent:book:yes"},
					{Text: "Отмена", Data: "agent:book:no"},
				}},
			}, true
		}
		// no proposal: continue the loop to re-plan with the tool results
	}

	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Это оказалось сложновато 🐾 Попробуй переформулировать или уточнить участников и даты."}, true
}

func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: "Спроси меня про расписание — например: «когда у Миа и Алекса есть общий час на следующей неделе?» 🐾"}
}

func parsePending(args []byte) (PendingBooking, error) {
	var in struct {
		Type   string   `json:"type"`
		Dept   string   `json:"dept"`
		Date   string   `json:"date"`
		Start  string   `json:"start"`
		End    string   `json:"end"`
		Emails []string `json:"emails"`
		Desc   string   `json:"desc"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return PendingBooking{}, fmt.Errorf("bad proposal arguments: %w", err)
	}
	if in.Type == "" || in.Date == "" || in.Start == "" || in.End == "" || len(in.Emails) == 0 {
		return PendingBooking{}, fmt.Errorf("proposal missing required fields (type, date, start, end, emails)")
	}
	if _, err := time.ParseInLocation("2006-01-02", in.Date, time.UTC); err != nil {
		return PendingBooking{}, fmt.Errorf("bad date (want YYYY-MM-DD)")
	}
	if _, err := time.ParseInLocation("15:04", in.Start, time.UTC); err != nil {
		return PendingBooking{}, fmt.Errorf("bad start time (want HH:MM)")
	}
	if _, err := time.ParseInLocation("15:04", in.End, time.UTC); err != nil {
		return PendingBooking{}, fmt.Errorf("bad end time (want HH:MM)")
	}
	return PendingBooking{Dept: in.Dept, Type: in.Type, Date: in.Date, Start: in.Start, End: in.End, Emails: in.Emails, Desc: in.Desc}, nil
}

// OnCallback handles confirm/cancel taps. Returns handled=false for callbacks
// that aren't ours so the dispatcher can fall through.
func (s *Service) OnCallback(ctx context.Context, telegramID int64, data string) (Reply, bool) {
	switch data {
	case "agent:book:yes":
		s.bookMu.Lock()
		st, err := s.sessions.Get(ctx, telegramID)
		if err != nil || st == nil || st.Pending == nil {
			s.bookMu.Unlock()
			return Reply{Text: "Предложение устарело 🐾 Попроси заново.", Edit: true}, true
		}
		pb := *st.Pending
		st.Pending = nil
		_ = s.sessions.Set(ctx, telegramID, *st)
		s.bookMu.Unlock()
		if s.booker == nil {
			return Reply{Text: "Бронирование сейчас недоступно.", Edit: true}, true
		}
		msg, berr := s.booker.Book(ctx, telegramID, pb)
		if berr != nil {
			return Reply{Text: berr.Error(), Edit: true}, true
		}
		return Reply{Text: msg, Edit: true}, true
	case "agent:book:no":
		st, err := s.sessions.Get(ctx, telegramID)
		if err == nil && st != nil {
			st.Pending = nil
			_ = s.sessions.Set(ctx, telegramID, *st)
		}
		return Reply{Text: "Хорошо, не бронирую 🐾", Edit: true}, true
	default:
		return Reply{}, false
	}
}
