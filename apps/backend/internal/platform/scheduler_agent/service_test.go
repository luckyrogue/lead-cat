package scheduler_agent

import (
	"context"
	"strings"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

// memSessions is an in-memory sessions store for tests.
type memSessions struct{ m map[int64]State }

func newMemSessions() *memSessions { return &memSessions{m: map[int64]State{}} }
func (s *memSessions) Get(_ context.Context, id int64) (*State, error) {
	st, ok := s.m[id]
	if !ok {
		return nil, nil
	}
	cp := st
	return &cp, nil
}
func (s *memSessions) Set(_ context.Context, id int64, st State) error { s.m[id] = st; return nil }
func (s *memSessions) Del(_ context.Context, id int64) error           { delete(s.m, id); return nil }

// scriptPlanner returns pre-scripted turns, one per call.
type scriptPlanner struct {
	turns []application.AgentTurn
	calls int
	got   []int // number of history messages seen on each call
}

func (p *scriptPlanner) Plan(_ context.Context, _ string, history []application.AgentMessage, _ []application.AgentTool) (application.AgentTurn, error) {
	p.got = append(p.got, len(history))
	turn := p.turns[p.calls]
	p.calls++
	return turn, nil
}

func TestService_OnText_DirectAnswer(t *testing.T) {
	planner := &scriptPlanner{turns: []application.AgentTurn{{Text: "Привет! Кого ищем?"}}}
	svc := New(planner, fakeBackend{}, newMemSessions())

	reply, handled := svc.OnText(context.Background(), 42, "привет")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if reply.Text != "Привет! Кого ищем?" {
		t.Fatalf("text = %q", reply.Text)
	}
	if planner.calls != 1 {
		t.Fatalf("planner called %d times, want 1", planner.calls)
	}
}

func TestService_OnText_RunsToolThenAnswers(t *testing.T) {
	be := fakeBackend{employees: []model.Employee{{FullName: "Mia", Email: "mia@co.com"}}}
	planner := &scriptPlanner{turns: []application.AgentTurn{
		{ToolCalls: []application.AgentToolCall{{ID: "t1", Name: "search_people", Input: []byte(`{"query":"mia"}`)}}},
		{Text: "Нашёл Mia <mia@co.com>."},
	}}
	svc := New(planner, be, newMemSessions())

	reply, handled := svc.OnText(context.Background(), 7, "найди mia")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if !strings.Contains(reply.Text, "mia@co.com") {
		t.Fatalf("text = %q", reply.Text)
	}
	if planner.calls != 2 {
		t.Fatalf("planner called %d times, want 2", planner.calls)
	}
	// Second call must see: user msg, assistant tool-call msg, user tool-result msg.
	if planner.got[1] != 3 {
		t.Fatalf("second Plan saw %d history messages, want 3", planner.got[1])
	}
}

func TestService_OnText_UnknownToolBecomesErrorResult(t *testing.T) {
	planner := &scriptPlanner{turns: []application.AgentTurn{
		{ToolCalls: []application.AgentToolCall{{ID: "t1", Name: "nope", Input: []byte(`{}`)}}},
		{Text: "Извини, не получилось."},
	}}
	svc := New(planner, fakeBackend{}, newMemSessions())

	reply, _ := svc.OnText(context.Background(), 9, "do x")
	if reply.Text != "Извини, не получилось." {
		t.Fatalf("text = %q", reply.Text)
	}
	if planner.calls != 2 {
		t.Fatalf("planner called %d times, want 2", planner.calls)
	}
}

func TestService_OnText_IterationCap(t *testing.T) {
	loopTurn := application.AgentTurn{ToolCalls: []application.AgentToolCall{{ID: "t", Name: "search_people", Input: []byte(`{"query":"x"}`)}}}
	turns := make([]application.AgentTurn, 20)
	for i := range turns {
		turns[i] = loopTurn
	}
	planner := &scriptPlanner{turns: turns}
	svc := New(planner, fakeBackend{}, newMemSessions())

	reply, handled := svc.OnText(context.Background(), 1, "loop")
	if !handled {
		t.Fatal("expected handled=true")
	}
	if reply.Text == "" {
		t.Fatal("expected a graceful fallback message, got empty")
	}
	if planner.calls > maxIterations {
		t.Fatalf("planner called %d times, exceeds cap %d", planner.calls, maxIterations)
	}
}
