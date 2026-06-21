package checker

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// ── compile-time interface assertions ────────────────────────────────────────

var _ Backend = (*fakeBackend)(nil)
var _ sessions = (*fakeSessions)(nil)

// ── fakeBackend ───────────────────────────────────────────────────────────────

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

// ── fakeSessions ──────────────────────────────────────────────────────────────

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

// ── helpers ───────────────────────────────────────────────────────────────────

func newService(b *fakeBackend, sess *fakeSessions) *Service {
	return New(b, sess)
}

// ── TestStart_SetsParticipantsStep ────────────────────────────────────────────

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

// ── TestOnText_Search_ReturnsMatches ──────────────────────────────────────────

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

	// Seed session via Start (as a real flow would do)
	svc.Start(ctx, tid, "ru")

	reply, handled := svc.OnText(ctx, tid, "ivan", "ru")

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

// ── TestOnText_NoSession_NotHandled ───────────────────────────────────────────

func TestOnText_NoSession_NotHandled(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeBackend{}, newFakeSessions())
	ctx := context.Background()
	const tid int64 = 9999

	reply, handled := svc.OnText(ctx, tid, "hello", "ru")

	if handled {
		t.Fatal("OnText: expected handled=false for unknown session")
	}
	if reply.Text != "" || reply.Keyboard != nil {
		t.Fatalf("OnText: expected zero Reply, got %+v", reply)
	}
}

// ── TestParseRange_ValidAndErrors ─────────────────────────────────────────────

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

// ── TestDayLabel ──────────────────────────────────────────────────────────────

func TestDayLabel(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	// 2026-06-01 is a Monday → "Пн"
	day := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	label := dayLabel(day, loc)

	// The format is "%s, %02d.%02d" → "Пн, 01.06"
	if !strings.Contains(label, "01.06") {
		t.Fatalf("dayLabel: expected date part '01.06' in %q", label)
	}
	if !strings.Contains(label, "Пн") {
		t.Fatalf("dayLabel: expected weekday abbreviation 'Пн' in %q", label)
	}
}

// ── TestOnCallback_DoneTransition ─────────────────────────────────────────────
//
// Drive the "chk:done" callback from a session that already has emails set
// at stepParticipants. This verifies the done() path that advances to stepRange.

func TestOnCallback_DoneTransition(t *testing.T) {
	t.Parallel()
	sess := newFakeSessions()
	svc := newService(&fakeBackend{}, sess)
	ctx := context.Background()
	const tid int64 = 1003

	// Manually seed a session with an email already added (simulates after add)
	_ = sess.Set(ctx, tid, State{
		Step:   stepParticipants,
		Emails: []string{"alice@example.com"},
		Cands:  []string{"alice@example.com"},
	})

	reply, handled := svc.OnCallback(ctx, tid, "chk:done", "ru")

	if !handled {
		t.Fatal("OnCallback chk:done: expected handled=true")
	}
	if reply.Text == "" {
		t.Fatal("OnCallback chk:done: expected non-empty Reply.Text")
	}
	// Session should now be at stepRange
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

// ── TestChecker_Localized ─────────────────────────────────────────────────────

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

// ── TestChecker_CallbackSessionExpired_Localized ──────────────────────────────

func TestChecker_CallbackSessionExpired_Localized(t *testing.T) {
	svc := New(&fakeBackend{}, newFakeSessions())
	r, _ := svc.OnCallback(context.Background(), 99, "chk:done", "en")
	if !strings.Contains(r.Text, "Session expired") {
		t.Fatalf("en session-expired = %q", r.Text)
	}
}

// TODO(WS2c): deeper chk: callback branches
// The chk:add: and chk:dur: paths require careful coordination between Cands
// index state and callback data. Cover them in a follow-up with table-driven
// tests seeding State.Cands/Emails directly.
