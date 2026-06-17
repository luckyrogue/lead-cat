package meeting

import (
	"errors"
	"testing"
	"time"
)

func validInput() Input {
	return Input{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		StartsAt: ts(2026, 6, 1, 10, 0), EndsAt: ts(2026, 6, 1, 11, 0),
		Recurrence: Once,
	}
}

func TestValidate_OK(t *testing.T) {
	if err := validInput().Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidate_EndBeforeStart(t *testing.T) {
	in := validInput()
	in.EndsAt = in.StartsAt.Add(-time.Minute)
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for end<=start")
	}
}

func TestValidate_MissingFields(t *testing.T) {
	in := validInput()
	in.Dept = ""
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for missing dept")
	}
}

func TestValidate_CustomNeedsDays(t *testing.T) {
	in := validInput()
	in.Recurrence = Custom
	if err := in.Validate(); !errors.Is(err, ErrRecurrenceDays) {
		t.Fatalf("want ErrRecurrenceDays, got %v", err)
	}
}

func TestValidate_WeekdayRange(t *testing.T) {
	in := validInput()
	in.Recurrence = Custom
	in.RecurrenceDays = []int{0}
	if err := in.Validate(); err == nil {
		t.Fatal("expected error for weekday out of 1..7")
	}
}

func TestRecurrence_Valid(t *testing.T) {
	if !Daily.Valid() || Recurrence("nope").Valid() {
		t.Fatal("Valid() wrong")
	}
}
