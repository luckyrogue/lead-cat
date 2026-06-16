package scheduler_agent

import (
	"context"

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
	sessions sessions
	tools    []application.AgentTool
}

func New(planner application.Planner, backend Backend, sess sessions) *Service {
	return &Service{planner: planner, backend: backend, sessions: sess, tools: ToolSpecs()}
}

// OnText handles a free-text message. It always returns handled=true (the agent
// is the catch-all), so wire it LAST in the dispatcher fallback chain.
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
		st.History = append(st.History, application.AgentMessage{Role: "assistant", Text: turn.Text, ToolCalls: turn.ToolCalls})
		results := make([]application.AgentToolResult, 0, len(turn.ToolCalls))
		for _, call := range turn.ToolCalls {
			out, derr := Dispatch(ctx, s.backend, call.Name, call.Input)
			if derr != nil {
				results = append(results, application.AgentToolResult{ID: call.ID, Content: derr.Error(), IsError: true})
				continue
			}
			results = append(results, application.AgentToolResult{ID: call.ID, Content: out})
		}
		st.History = append(st.History, application.AgentMessage{Role: "user", ToolResults: results})
	}

	_ = s.sessions.Set(ctx, telegramID, *st)
	return Reply{Text: "Это оказалось сложновато 🐾 Попробуй переформулировать или уточнить участников и даты."}, true
}

// Start resets the conversation and greets.
func (s *Service) Start(ctx context.Context, telegramID int64) Reply {
	_ = s.sessions.Del(ctx, telegramID)
	return Reply{Text: "Спроси меня про расписание — например: «когда у Миа и Алекса есть общий час на следующей неделе?» 🐾"}
}
