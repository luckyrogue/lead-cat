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

type fakeBackend struct {
	employees []model.Employee
	slots     []application.FreeSlot
	conflicts []application.Conflict
}

func (f fakeBackend) SearchEmployeesGlobal(_ context.Context, _ string) ([]model.Employee, error) {
	return f.employees, nil
}
func (f fakeBackend) FreeSlots(_ context.Context, _ []string, _, _ time.Time, _ int) ([]application.FreeSlot, error) {
	return f.slots, nil
}
func (f fakeBackend) MeetingConflicts(_ context.Context, _ []string, _, _ time.Time, _ uuid.UUID) ([]application.Conflict, error) {
	return f.conflicts, nil
}

func TestDispatch_SearchPeople(t *testing.T) {
	be := fakeBackend{employees: []model.Employee{
		{FullName: "Mia Cat", Email: "mia@co.com"},
		{FullName: "Alex Paw", Email: "alex@co.com"},
	}}
	out, err := Dispatch(context.Background(), be, "search_people", []byte(`{"query":"a"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "mia@co.com") || !strings.Contains(out, "alex@co.com") {
		t.Fatalf("expected both emails in output, got:\n%s", out)
	}
}

func TestDispatch_FindFreeSlots(t *testing.T) {
	day := time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC)
	be := fakeBackend{slots: []application.FreeSlot{
		{Day: day, Start: day.Add(9 * time.Hour), End: day.Add(10 * time.Hour), Mins: 60},
	}}
	out, err := Dispatch(context.Background(), be, "find_free_slots",
		[]byte(`{"emails":["mia@co.com"],"from":"2026-06-22","to":"2026-06-26","duration_mins":30}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(out, "2026-06-22") {
		t.Fatalf("expected the slot day in output, got:\n%s", out)
	}
}

func TestDispatch_CheckConflicts_None(t *testing.T) {
	be := fakeBackend{}
	out, err := Dispatch(context.Background(), be, "check_conflicts",
		[]byte(`{"emails":["mia@co.com"],"start":"2026-06-22 10:00","end":"2026-06-22 10:30"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "no conflicts") {
		t.Fatalf("expected a no-conflicts message, got:\n%s", out)
	}
}

func TestDispatch_UnknownTool(t *testing.T) {
	_, err := Dispatch(context.Background(), fakeBackend{}, "delete_everything", []byte(`{}`))
	if err == nil {
		t.Fatal("expected an error for an unknown tool")
	}
}

func TestDispatch_BadArgs(t *testing.T) {
	_, err := Dispatch(context.Background(), fakeBackend{}, "find_free_slots", []byte(`{"from":"not-a-date"}`))
	if err == nil {
		t.Fatal("expected an error for unparseable arguments")
	}
}

func TestToolSpecs_Shape(t *testing.T) {
	specs := ToolSpecs()
	names := map[string]bool{}
	for _, s := range specs {
		names[s.Name] = true
		if s.Description == "" || s.InputSchema == nil {
			t.Fatalf("tool %q missing description or schema", s.Name)
		}
	}
	for _, want := range []string{"search_people", "find_free_slots", "check_conflicts"} {
		if !names[want] {
			t.Fatalf("missing tool spec %q", want)
		}
	}
}
