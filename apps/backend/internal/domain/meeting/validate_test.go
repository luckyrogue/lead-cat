package meeting

import (
	"errors"
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

func TestValidate_CustomRequiresDays(t *testing.T) {
	in := base()
	in.Recurrence = Custom
	in.RecurrenceDays = nil
	if !errors.Is(in.Validate(), ErrRecurrenceDays) {
		t.Fatal("expected ErrRecurrenceDays")
	}
}

func TestValidate_CustomDaysOutOfRange(t *testing.T) {
	in := base()
	in.Recurrence = Custom
	in.RecurrenceDays = []int{0, 3}
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for day out of range")
	}
	in.RecurrenceDays = []int{1, 8}
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for day out of range")
	}
}
