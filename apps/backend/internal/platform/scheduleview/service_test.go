package scheduleview

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeSessions struct {
	m map[int64]*State
}

var _ sessions = (*fakeSessions)(nil)

func newFakeSessions() *fakeSessions { return &fakeSessions{m: make(map[int64]*State)} }

func (f *fakeSessions) Get(_ context.Context, id int64) (*State, error) {
	s, ok := f.m[id]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (f *fakeSessions) Set(_ context.Context, id int64, s State) error {
	cp := s
	f.m[id] = &cp
	return nil
}

func (f *fakeSessions) Del(_ context.Context, id int64) error {
	delete(f.m, id)
	return nil
}

type fakeBackend struct {
	employees     []postgres.Employee
	meetings      []postgres.Meeting
	searchCalls   []string
	scheduleCalls []struct {
		email    string
		from, to time.Time
	}
}

var _ Backend = (*fakeBackend)(nil)

func (f *fakeBackend) SearchEmployeesGlobal(_ context.Context, query string) ([]postgres.Employee, error) {
	f.searchCalls = append(f.searchCalls, query)
	return f.employees, nil
}

func (f *fakeBackend) EmployeeSchedule(_ context.Context, email string, from, to time.Time) ([]postgres.Meeting, error) {
	f.scheduleCalls = append(f.scheduleCalls, struct {
		email    string
		from, to time.Time
	}{email, from, to})
	return f.meetings, nil
}

func newService(be *fakeBackend, sess *fakeSessions) *Service {
	return New(be, sess)
}

func TestStart_SetsAwaitSearch(t *testing.T) {
	sess := newFakeSessions()
	svc := newService(&fakeBackend{}, sess)

	const tid = int64(42)
	reply := svc.Start(context.Background(), tid, "ru")

	if reply.Text == "" {
		t.Fatal("Start: expected non-empty reply text")
	}

	st, err := sess.Get(context.Background(), tid)
	if err != nil {
		t.Fatalf("sessions.Get error: %v", err)
	}
	if st == nil {
		t.Fatal("Start: session not saved")
	}
	if st.Step != stepAwait {
		t.Errorf("Step: want %q, got %q", stepAwait, st.Step)
	}
	if st.AwaitingKind != awaitSearch {
		t.Errorf("AwaitingKind: want %q, got %q", awaitSearch, st.AwaitingKind)
	}
}

func TestOnText_Search(t *testing.T) {
	sess := newFakeSessions()
	be := &fakeBackend{
		employees: []postgres.Employee{
			{FullName: "Ivan Petrov", Email: "ivan@example.com"},
		},
	}
	svc := newService(be, sess)

	const tid = int64(7)
	svc.Start(context.Background(), tid, "ru")

	reply, handled := svc.OnText(context.Background(), tid, "ivan", "ru", almaty())

	if !handled {
		t.Fatal("OnText: want handled=true")
	}
	if reply.Text == "" {
		t.Fatal("OnText: want non-empty reply text")
	}
	if len(be.searchCalls) != 1 || be.searchCalls[0] != "ivan" {
		t.Errorf("SearchEmployeesGlobal call: want [\"ivan\"], got %v", be.searchCalls)
	}
	found := false
	for _, row := range reply.Keyboard {
		for _, btn := range row {
			if strings.HasPrefix(btn.Data, "sched:pick:") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("OnText search: expected a sched:pick: keyboard button, keyboard=%v", reply.Keyboard)
	}
}

func TestParseDate_ValidAndError(t *testing.T) {
	loc := time.UTC

	d, err := parseDate("2026-06-01", loc)
	if err != nil {
		t.Fatalf("parseDate valid: unexpected error: %v", err)
	}
	if d.Year() != 2026 || d.Month() != 6 || d.Day() != 1 {
		t.Errorf("parseDate valid: got %v", d)
	}

	_, err = parseDate("nope", loc)
	if err == nil {
		t.Fatal("parseDate invalid: expected error, got nil")
	}
}

func TestParseRange_EndExclusive(t *testing.T) {
	loc := time.UTC

	from, to, err := parseRange("2026-06-01..2026-06-03", loc)
	if err != nil {
		t.Fatalf("parseRange: unexpected error: %v", err)
	}
	wantFrom := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	wantTo := time.Date(2026, 6, 4, 0, 0, 0, 0, loc) // d2.AddDate(0,0,1) → end-exclusive
	if !from.Equal(wantFrom) {
		t.Errorf("parseRange from: want %v, got %v", wantFrom, from)
	}
	if !to.Equal(wantTo) {
		t.Errorf("parseRange to (end-exclusive): want %v, got %v", wantTo, to)
	}

	_, _, err = parseRange("2026-06-03..2026-06-01", loc)
	if err == nil {
		t.Fatal("parseRange reversed: expected error, got nil")
	}
}

func TestDayWindow(t *testing.T) {
	loc := time.UTC
	now := time.Date(2026, 6, 10, 14, 30, 0, 0, loc)
	sod := time.Date(2026, 6, 10, 0, 0, 0, 0, loc)

	from, to, ok := dayWindow(now, "today", loc)
	if !ok {
		t.Fatal("dayWindow today: ok=false")
	}
	if !from.Equal(sod) {
		t.Errorf("today from: want %v, got %v", sod, from)
	}
	if !to.Equal(sod.AddDate(0, 0, 1)) {
		t.Errorf("today to: want %v, got %v", sod.AddDate(0, 0, 1), to)
	}

	from, to, ok = dayWindow(now, "tomorrow", loc)
	if !ok {
		t.Fatal("dayWindow tomorrow: ok=false")
	}
	if !from.Equal(sod.AddDate(0, 0, 1)) {
		t.Errorf("tomorrow from: want %v, got %v", sod.AddDate(0, 0, 1), from)
	}
	if !to.Equal(sod.AddDate(0, 0, 2)) {
		t.Errorf("tomorrow to: want %v, got %v", sod.AddDate(0, 0, 2), to)
	}

	from, to, ok = dayWindow(now, "upcoming", loc)
	if !ok {
		t.Fatal("dayWindow upcoming: ok=false")
	}
	if !from.Equal(now) {
		t.Errorf("upcoming from: want %v, got %v", now, from)
	}
	if !to.Equal(now.AddDate(1, 0, 0)) {
		t.Errorf("upcoming to: want %v, got %v", now.AddDate(1, 0, 0), to)
	}

	_, _, ok = dayWindow(now, "week", loc)
	if ok {
		t.Fatal("dayWindow unknown kind: expected ok=false, got true")
	}
}

func TestStatusEmoji(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)

	future := now.Add(time.Hour)
	if got := statusEmoji(future, now); got != "🔜" {
		t.Errorf("future: want 🔜, got %q", got)
	}

	past := now.Add(-time.Hour)
	if got := statusEmoji(past, now); got != "✅" {
		t.Errorf("past: want ✅, got %q", got)
	}
}

func TestOnCallback_Back(t *testing.T) {
	sess := newFakeSessions()
	svc := newService(&fakeBackend{}, sess)

	const tid = int64(99)

	reply, handled := svc.OnCallback(context.Background(), tid, "sched:back", "ru", almaty())

	if !handled {
		t.Fatal("OnCallback sched:back: want handled=true")
	}
	if reply.Text == "" {
		t.Fatal("OnCallback sched:back: want non-empty reply text")
	}

	st, err := sess.Get(context.Background(), tid)
	if err != nil {
		t.Fatalf("sessions.Get error: %v", err)
	}
	if st == nil {
		t.Fatal("sched:back: session not saved")
	}
	if st.Step != stepAwait {
		t.Errorf("sched:back Step: want %q, got %q", stepAwait, st.Step)
	}
	if st.AwaitingKind != awaitSearch {
		t.Errorf("sched:back AwaitingKind: want %q, got %q", awaitSearch, st.AwaitingKind)
	}
}

func TestSchedule_Localized(t *testing.T) {
	svc := newService(&fakeBackend{}, newFakeSessions())
	ru := svc.Start(context.Background(), 1, "ru")
	en := svc.Start(context.Background(), 2, "en")
	if ru.Text == en.Text {
		t.Fatalf("Start must differ by language; both = %q", ru.Text)
	}
	if !strings.Contains(en.Text, "Whose schedule") {
		t.Errorf("en Start = %q", en.Text)
	}
}

func TestSchedule_DayWindow_UsesLoc(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	almatyLoc, _ := time.LoadLocation("Asia/Almaty")
	now := time.Date(2026, 6, 15, 2, 0, 0, 0, time.UTC) // 07:00 Almaty, 03:00 London
	fL, _, ok := dayWindow(now, "today", london)
	if !ok {
		t.Fatal("today window")
	}
	fA, _, _ := dayWindow(now, "today", almatyLoc)
	if fL.Equal(fA) {
		t.Fatal("today start should differ between London and Almaty for the same instant")
	}
}

func TestSchedule_ParseDate_UsesLoc(t *testing.T) {
	london, _ := time.LoadLocation("Europe/London")
	d, err := parseDate("2026-06-15", london)
	if err != nil {
		t.Fatal(err)
	}
	if d.UTC().Hour() != 23 {
		t.Errorf("London 2026-06-15 midnight should be 23:00 prev-day UTC, got %v", d.UTC())
	}
}
