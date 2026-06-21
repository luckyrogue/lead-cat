package checker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

var _ Backend = (*fakeBackend)(nil)
var _ sessions = (*fakeSessions)(nil)

type fakeBackend struct {
	employees    []postgres.Employee
	freeSlots    []application.FreeSlot
	searchCalled string // last query passed to SearchEmployeesGlobal
	slotsCalled  bool
}

func (f *fakeBackend) SearchEmployeesGlobal(_ context.Context, query string) ([]postgres.Employee, error) {
	f.searchCalled = query
	return f.employees, nil
}

func (f *fakeBackend) FreeSlots(_ context.Context, _ string, _ []string, _, _ time.Time, _ int) ([]application.FreeSlot, error) {
	f.slotsCalled = true
	return f.freeSlots, nil
}

type fakeSessions struct {
	store map[int64]State
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{store: make(map[int64]State)}
}

func (f *fakeSessions) Get(_ context.Context, telegramID int64) (*State, error) {
	s, ok := f.store[telegramID]
	if !ok {
		return nil, nil
	}
	copy := s
	return &copy, nil
}

func (f *fakeSessions) Set(_ context.Context, telegramID int64, s State) error {
	f.store[telegramID] = s
	return nil
}

func (f *fakeSessions) Del(_ context.Context, telegramID int64) error {
	delete(f.store, telegramID)
	return nil
}

func newService(b *fakeBackend, sess *fakeSessions) *Service {
	return New(b, sess)
}

func TestStart_SetsParticipantsStep(t *testing.T) {
	t.Parallel()
	sess := newFakeSessions()
	svc := newService(&fakeBackend{}, sess)
	ctx := context.Background()
	const tid int64 = 1001

	reply := svc.Start(ctx, tid, "ru")

	if reply.Text == "" {
		t.Fatal("Start: expected non-empty Reply.Text")
	}
	st, err := sess.Get(ctx, tid)
	if err != nil {
		t.Fatalf("sess.Get error: %v", err)
	}
	if st == nil {
		t.Fatal("Start: session not stored")
	}
	if st.Step != stepParticipants {
		t.Fatalf("Start: expected step %q, got %q", stepParticipants, st.Step)
	}
}

func TestOnText_Search_ReturnsMatches(t *testing.T) {
	t.Parallel()
	fb := &fakeBackend{
		employees: []postgres.Employee{
			{FullName: "Ivan Petrov", Email: "ivan@example.com"},
		},
	}
	sess := newFakeSessions()
	svc := newService(fb, sess)
	ctx := context.Background()
	const tid int64 = 1002

	svc.Start(ctx, tid, "ru")

	reply, handled := svc.OnText(ctx, tid, "ivan", "ru", almaty())

	if !handled {
		t.Fatal("OnText: expected handled=true")
	}
	if reply.Text == "" {
		t.Fatal("OnText: expected non-empty Reply.Text")
	}
	if len(reply.Keyboard) == 0 {
		t.Fatal("OnText: expected at least one keyboard row")
	}
	if fb.searchCalled != "ivan" {
		t.Fatalf("OnText: expected SearchEmployeesGlobal called with %q, got %q", "ivan", fb.searchCalled)
	}
}

func TestOnText_NoSession_NotHandled(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeBackend{}, newFakeSessions())
	ctx := context.Background()
	const tid int64 = 9999

	reply, handled := svc.OnText(ctx, tid, "hello", "ru", almaty())

	if handled {
		t.Fatal("OnText: expected handled=false for unknown session")
	}
	if reply.Text != "" || reply.Keyboard != nil {
		t.Fatalf("OnText: expected zero Reply, got %+v", reply)
	}
}

func TestParseRange_ValidAndErrors(t *testing.T) {
	t.Parallel()
	loc := time.UTC

	t.Run("valid range", func(t *testing.T) {
		from, to, err := parseRange("2026-06-01..2026-06-03", loc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !from.Before(to) {
			t.Fatalf("expected from (%v) before to (%v)", from, to)
		}
	})

	t.Run("no separator", func(t *testing.T) {
		_, _, err := parseRange("2026-06-01", loc)
		if err == nil {
			t.Fatal("expected error for input without '..'")
		}
	})

	t.Run("bad left date", func(t *testing.T) {
		_, _, err := parseRange("bad..2026-06-03", loc)
		if err == nil {
			t.Fatal("expected error for bad left date")
		}
	})

	t.Run("reversed range", func(t *testing.T) {
		_, _, err := parseRange("2026-06-03..2026-06-01", loc)
		if err == nil {
			t.Fatal("expected error for reversed date range")
		}
	})
}

func TestDayLabel(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	day := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	label := dayLabel(day, loc)

	if !strings.Contains(label, "01.06") {
		t.Fatalf("dayLabel: expected date part '01.06' in %q", label)
	}
	if !strings.Contains(label, "Пн") {
		t.Fatalf("dayLabel: expected weekday abbreviation 'Пн' in %q", label)
	}
}

func TestOnCallback_DoneTransition(t *testing.T) {
	t.Parallel()
	sess := newFakeSessions()
	svc := newService(&fakeBackend{}, sess)
	ctx := context.Background()
	const tid int64 = 1003

	_ = sess.Set(ctx, tid, State{
		Step:   stepParticipants,
		Emails: []string{"alice@example.com"},
		Cands:  []string{"alice@example.com"},
	})

	reply, handled := svc.OnCallback(ctx, tid, "chk:done", "ru", almaty())

	if !handled {
		t.Fatal("OnCallback chk:done: expected handled=true")
	}
	if reply.Text == "" {
		t.Fatal("OnCallback chk:done: expected non-empty Reply.Text")
	}
	st, err := sess.Get(ctx, tid)
	if err != nil {
		t.Fatalf("sess.Get error: %v", err)
	}
	if st == nil {
		t.Fatal("OnCallback chk:done: session missing after done")
	}
	if st.Step != stepRange {
		t.Fatalf("OnCallback chk:done: expected step %q, got %q", stepRange, st.Step)
	}
}

func TestChecker_Localized(t *testing.T) {
	be := &fakeBackend{}
	svc := New(be, newFakeSessions())
	ctx := context.Background()
	ru := svc.Start(ctx, 1, "ru")
	en := svc.Start(ctx, 2, "en")
	if ru.Text == en.Text {
		t.Fatalf("Start must differ by language; both = %q", ru.Text)
	}
	if !strings.Contains(en.Text, "Enter a participant") {
		t.Errorf("en Start = %q", en.Text)
	}
}

func TestChecker_CallbackSessionExpired_Localized(t *testing.T) {
	svc := New(&fakeBackend{}, newFakeSessions())
	r, _ := svc.OnCallback(context.Background(), 99, "chk:done", "en", almaty())
	if !strings.Contains(r.Text, "Session expired") {
		t.Fatalf("en session-expired = %q", r.Text)
	}
}

func TestChecker_SetRange_UsesLoc(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	svc := New(&fakeBackend{}, newFakeSessions())
	ctx := context.Background()
	_ = svc.Start(ctx, 1, "ru")
	from, _, err := parseRange("2026-06-15..2026-06-15", london)
	if err != nil {
		t.Fatal(err)
	}
	fromAlmaty, _, _ := parseRange("2026-06-15..2026-06-15", almaty())
	if from.Equal(fromAlmaty) {
		t.Fatal("expected London and Almaty parse of the same date to differ in absolute time")
	}
	if from.UTC().Hour() != 23 { // 2026-06-15 00:00 BST == 2026-06-14 23:00 UTC
		t.Errorf("London 2026-06-15 midnight should be 23:00 prev-day UTC, got %v", from.UTC())
	}
}
