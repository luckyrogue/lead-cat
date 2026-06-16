package scheduler_agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

// --- self-contained test doubles ---

type stubBackend struct{}

func (stubBackend) SearchEmployeesGlobal(context.Context, string) ([]model.Employee, error) {
	return nil, nil
}
func (stubBackend) FreeSlots(context.Context, []string, time.Time, time.Time, int) ([]application.FreeSlot, error) {
	return nil, nil
}
func (stubBackend) MeetingConflicts(context.Context, []string, time.Time, time.Time, uuid.UUID) ([]application.Conflict, error) {
	return nil, nil
}

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

type scriptPlanner struct {
	turns []application.AgentTurn
	calls int
}

func (p *scriptPlanner) Plan(_ context.Context, _ string, _ []application.AgentMessage, _ []application.AgentTool) (application.AgentTurn, error) {
	turn := p.turns[p.calls]
	p.calls++
	return turn, nil
}

type fakeBooker struct {
	called  bool
	got     PendingBooking
	gotTgID int64
	result  string
	err     error
}

func (b *fakeBooker) Book(_ context.Context, telegramID int64, pb PendingBooking) (string, error) {
	b.called = true
	b.gotTgID = telegramID
	b.got = pb
	if b.err != nil {
		return "", b.err
	}
	return b.result, nil
}

// --- tests ---

func TestService_ProposeMeeting_ShowsConfirmCard(t *testing.T) {
	planner := &scriptPlanner{turns: []application.AgentTurn{
		{ToolCalls: []application.AgentToolCall{{ID: "p1", Name: "propose_meeting",
			Input: []byte(`{"type":"Sync","date":"2026-06-22","start":"10:00","end":"10:30","emails":["mia@co.com"]}`)}}},
	}}
	booker := &fakeBooker{}
	sess := newMemSessions()
	svc := NewWithBooker(planner, stubBackend{}, booker, sess)

	reply, handled := svc.OnText(context.Background(), 5, "book sync with mia tomorrow 10:00")
	if !handled {
		t.Fatal("handled=false")
	}
	if booker.called {
		t.Fatal("Booker must NOT be called until the user confirms")
	}
	if len(reply.Keyboard) == 0 {
		t.Fatal("expected a confirm keyboard")
	}
	if !strings.Contains(reply.Text, "2026-06-22") || !strings.Contains(reply.Text, "mia@co.com") {
		t.Fatalf("confirm card missing details: %q", reply.Text)
	}
	st, _ := sess.Get(context.Background(), 5)
	if st == nil || st.Pending == nil || st.Pending.Type != "Sync" {
		t.Fatal("expected Pending booking stored in session")
	}
	if planner.calls != 1 {
		t.Fatalf("planner calls = %d, want 1", planner.calls)
	}
}

func TestService_OnCallback_ConfirmBooks(t *testing.T) {
	booker := &fakeBooker{result: "Встреча создана ✅"}
	sess := newMemSessions()
	_ = sess.Set(context.Background(), 5, State{Pending: &PendingBooking{
		Type: "Sync", Date: "2026-06-22", Start: "10:00", End: "10:30", Emails: []string{"mia@co.com"},
	}})
	svc := NewWithBooker(&scriptPlanner{}, stubBackend{}, booker, sess)

	reply, handled := svc.OnCallback(context.Background(), 5, "agent:book:yes")
	if !handled {
		t.Fatal("handled=false")
	}
	if !booker.called || booker.gotTgID != 5 || booker.got.Type != "Sync" {
		t.Fatalf("booker not called correctly: %+v", booker)
	}
	if !strings.Contains(reply.Text, "создана") {
		t.Fatalf("reply = %q", reply.Text)
	}
	st, _ := sess.Get(context.Background(), 5)
	if st != nil && st.Pending != nil {
		t.Fatal("Pending should be cleared after confirm")
	}
}

func TestService_OnCallback_Cancel(t *testing.T) {
	booker := &fakeBooker{}
	sess := newMemSessions()
	_ = sess.Set(context.Background(), 5, State{Pending: &PendingBooking{Type: "Sync"}})
	svc := NewWithBooker(&scriptPlanner{}, stubBackend{}, booker, sess)

	_, handled := svc.OnCallback(context.Background(), 5, "agent:book:no")
	if !handled {
		t.Fatal("handled=false")
	}
	if booker.called {
		t.Fatal("Booker must not be called on cancel")
	}
	st, _ := sess.Get(context.Background(), 5)
	if st != nil && st.Pending != nil {
		t.Fatal("Pending should be cleared on cancel")
	}
}

func TestService_OnCallback_NotOurs(t *testing.T) {
	svc := NewWithBooker(&scriptPlanner{}, stubBackend{}, &fakeBooker{}, newMemSessions())
	_, handled := svc.OnCallback(context.Background(), 5, "chk:done")
	if handled {
		t.Fatal("must not handle callbacks that aren't agent:book:*")
	}
}
