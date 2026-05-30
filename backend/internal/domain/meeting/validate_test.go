package meeting

import (
	"testing"
	"time"
)

func base() Input {
	start := time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC)
	return Input{
		Dept:       "Разработка",
		Type:       "Планёрка",
		Host:       "Иванов А.А.",
		StartsAt:   start,
		EndsAt:     start.Add(time.Hour),
		Recurrence: Once,
	}
}

func TestValidate_OK(t *testing.T) {
	if err := base().Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidate_EndBeforeStart(t *testing.T) {
	in := base()
	in.EndsAt = in.StartsAt.Add(-time.Minute)
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for end <= start")
	}
}

func TestValidate_MissingDept(t *testing.T) {
	in := base()
	in.Dept = ""
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for missing dept")
	}
}

func TestValidate_BadRecurrence(t *testing.T) {
	in := base()
	in.Recurrence = "yearly"
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for unknown recurrence")
	}
}
