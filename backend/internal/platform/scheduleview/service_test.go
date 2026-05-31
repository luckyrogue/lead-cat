package scheduleview

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

type fakeBackend struct {
	employees []postgres.Employee
	meetings  []postgres.Meeting
	gotEmail  string
	gotFrom   time.Time
	gotTo     time.Time
}

func (f *fakeBackend) SearchEmployeesGlobal(_ context.Context, query string) ([]postgres.Employee, error) {
	var out []postgres.Employee
	for _, e := range f.employees {
		if strings.Contains(strings.ToLower(e.FullName), strings.ToLower(query)) || strings.Contains(strings.ToLower(e.Email), strings.ToLower(query)) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (f *fakeBackend) EmployeeSchedule(_ context.Context, email string, from, to time.Time) ([]postgres.Meeting, error) {
	f.gotEmail, f.gotFrom, f.gotTo = email, from, to
	return f.meetings, nil
}

type memSessions struct{ m map[int64]*State }

func newMemSessions() *memSessions                                     { return &memSessions{m: map[int64]*State{}} }
func (s *memSessions) Get(_ context.Context, tg int64) (*State, error) { return s.m[tg], nil }
func (s *memSessions) Set(_ context.Context, tg int64, st State) error {
	c := st
	s.m[tg] = &c
	return nil
}
func (s *memSessions) Del(_ context.Context, tg int64) error { delete(s.m, tg); return nil }

func TestScheduleFlow_PickAndToday(t *testing.T) {
	ctx := context.Background()
	loc, _ := time.LoadLocation("Asia/Almaty")
	be := &fakeBackend{
		employees: []postgres.Employee{{FullName: "Иван Иванов", Email: "ivan@corp.kz"}},
		meetings: []postgres.Meeting{{
			Name: "Планёрка", StartsAt: time.Date(2026, 6, 2, 14, 0, 0, 0, loc).UTC(), EndsAt: time.Date(2026, 6, 2, 15, 0, 0, 0, loc).UTC(),
		}},
	}
	svc := New(be, newMemSessions())
	const tg = int64(70)

	if r := svc.Start(ctx, tg); !strings.Contains(r.Text, "email") {
		t.Fatalf("start: %+v", r)
	}
	if r, ok := svc.OnText(ctx, tg, "иван"); !ok || len(r.Keyboard) == 0 {
		t.Fatalf("search: %+v ok=%v", r, ok)
	}
	if r, ok := svc.OnCallback(ctx, tg, "sched:pick:0"); !ok || len(r.Keyboard) == 0 {
		t.Fatalf("pick: %+v ok=%v", r, ok)
	}
	r, ok := svc.OnCallback(ctx, tg, "sched:d:today")
	if !ok || !strings.Contains(r.Text, "Планёрка") {
		t.Fatalf("today list: %+v ok=%v", r, ok)
	}
	if be.gotEmail != "ivan@corp.kz" {
		t.Fatalf("schedule queried for %q", be.gotEmail)
	}
	if !be.gotTo.After(be.gotFrom) {
		t.Fatalf("window not valid: from=%v to=%v", be.gotFrom, be.gotTo)
	}
}

func TestScheduleFlow_RawEmailThenDate(t *testing.T) {
	ctx := context.Background()
	be := &fakeBackend{}
	svc := New(be, newMemSessions())
	const tg = int64(71)
	svc.Start(ctx, tg)
	svc.OnText(ctx, tg, "bob@corp.kz")
	svc.OnCallback(ctx, tg, "sched:pick:0")
	svc.OnCallback(ctx, tg, "sched:d:date")
	if r, ok := svc.OnText(ctx, tg, "2026-06-02"); !ok || !strings.Contains(r.Text, "Расписание") {
		t.Fatalf("date list: %+v ok=%v", r, ok)
	}
	if be.gotEmail != "bob@corp.kz" {
		t.Fatalf("schedule queried for %q", be.gotEmail)
	}
}

func TestScheduleFlow_BadDate(t *testing.T) {
	ctx := context.Background()
	be := &fakeBackend{}
	svc := New(be, newMemSessions())
	const tg = int64(72)
	svc.Start(ctx, tg)
	svc.OnText(ctx, tg, "bob@corp.kz")
	svc.OnCallback(ctx, tg, "sched:pick:0")
	svc.OnCallback(ctx, tg, "sched:d:date")
	if r, ok := svc.OnText(ctx, tg, "nope"); !ok || !strings.Contains(r.Text, "дата") {
		t.Fatalf("expected date error, got %+v ok=%v", r, ok)
	}
}

func TestScheduleFlow_NoResults(t *testing.T) {
	ctx := context.Background()
	svc := New(&fakeBackend{}, newMemSessions())
	const tg = int64(73)
	svc.Start(ctx, tg)
	if r, ok := svc.OnText(ctx, tg, "zzz"); !ok || !strings.Contains(r.Text, "Ничего не найдено") {
		t.Fatalf("expected no-results, got %+v ok=%v", r, ok)
	}
}

func TestScheduleFlow_BadIndex(t *testing.T) {
	ctx := context.Background()
	be := &fakeBackend{employees: []postgres.Employee{{FullName: "Иван", Email: "ivan@corp.kz"}}}
	svc := New(be, newMemSessions())
	const tg = int64(74)
	svc.Start(ctx, tg)
	svc.OnText(ctx, tg, "иван")
	if r, _ := svc.OnCallback(ctx, tg, "sched:pick:99"); !strings.Contains(r.Text, "Не найдено") {
		t.Fatalf("expected not-found for OOB index, got %+v", r)
	}
}

func TestScheduleFlow_SessionExpired(t *testing.T) {
	ctx := context.Background()
	svc := New(&fakeBackend{}, newMemSessions())
	const tg = int64(75)
	for _, data := range []string{"sched:pick:0", "sched:d:today", "sched:periods"} {
		if r, ok := svc.OnCallback(ctx, tg, data); !ok || !strings.Contains(r.Text, "истекла") {
			t.Fatalf("%s: expected session-expired, got %+v ok=%v", data, r, ok)
		}
	}
}

func TestScheduleFlow_Range(t *testing.T) {
	ctx := context.Background()
	be := &fakeBackend{}
	svc := New(be, newMemSessions())
	const tg = int64(76)
	svc.Start(ctx, tg)
	svc.OnText(ctx, tg, "bob@corp.kz")
	svc.OnCallback(ctx, tg, "sched:pick:0")
	svc.OnCallback(ctx, tg, "sched:d:range")
	if r, ok := svc.OnText(ctx, tg, "2026-06-01..2026-06-03"); !ok || !strings.Contains(r.Text, "Расписание") {
		t.Fatalf("range list: %+v ok=%v", r, ok)
	}
	if be.gotEmail != "bob@corp.kz" || !be.gotTo.After(be.gotFrom) {
		t.Fatalf("range query: email=%q from=%v to=%v", be.gotEmail, be.gotFrom, be.gotTo)
	}
}
