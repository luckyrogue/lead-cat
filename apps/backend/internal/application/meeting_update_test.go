package application

import (
	"errors"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func strp(s string) *string { return &s }

func baseMeeting() model.Meeting {
	loc, _ := time.LoadLocation("Asia/Almaty")
	return model.Meeting{
		Dept: "Разработка", Type: "Планёрка", Host: "Иванов А.А.",
		StartsAt:   time.Date(2026, 6, 1, 14, 0, 0, 0, loc).UTC(),
		EndsAt:     time.Date(2026, 6, 1, 15, 0, 0, 0, loc).UTC(),
		Recurrence: "once", Description: "old", Name: "old name",
	}
}

func TestApplyMeetingUpdate_DeptOnly(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := command.ApplyMeetingUpdate(baseMeeting(), UpdateMeetingInput{Dept: strp("Маркетинг")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Dept != "Маркетинг" {
		t.Fatalf("dept = %q", out.Dept)
	}
	if out.Name != "Маркетинг | Планёрка | Иванов А.А. | 2026-06-01" {
		t.Fatalf("name = %q", out.Name)
	}
	if !out.StartsAt.Equal(baseMeeting().StartsAt) {
		t.Fatalf("start changed unexpectedly")
	}
}

func TestApplyMeetingUpdate_DateTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := command.ApplyMeetingUpdate(baseMeeting(), UpdateMeetingInput{
		Date: strp("2026-06-02"), Start: strp("10:00"), End: strp("11:00"),
	}, loc)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 6, 2, 10, 0, 0, 0, loc).UTC()
	if !out.StartsAt.Equal(wantStart) {
		t.Fatalf("start = %v want %v", out.StartsAt, wantStart)
	}
	if out.Name != "Разработка | Планёрка | Иванов А.А. | 2026-06-02" {
		t.Fatalf("name = %q", out.Name)
	}
}

func TestApplyMeetingUpdate_EndBeforeStart(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	_, err := command.ApplyMeetingUpdate(baseMeeting(), UpdateMeetingInput{
		Date: strp("2026-06-02"), Start: strp("11:00"), End: strp("10:00"),
	}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestApplyMeetingUpdate_BadRecurrence(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	_, err := command.ApplyMeetingUpdate(baseMeeting(), UpdateMeetingInput{Recurrence: strp("hourly")}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}

func TestApplyMeetingUpdate_RecurrenceLabelInName(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := command.ApplyMeetingUpdate(baseMeeting(), UpdateMeetingInput{Recurrence: strp("weekly")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Разработка | Планёрка | Иванов А.А. | 2026-06-01 | Еженедельно" {
		t.Fatalf("name = %q", out.Name)
	}
}

func TestApplyMeetingUpdate_NameUsesLocalDate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	m := baseMeeting()

	m.StartsAt = time.Date(2026, 6, 1, 1, 0, 0, 0, loc).UTC()
	m.EndsAt = time.Date(2026, 6, 1, 2, 0, 0, 0, loc).UTC()
	out, err := command.ApplyMeetingUpdate(m, UpdateMeetingInput{Dept: strp("Маркетинг")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Name != "Маркетинг | Планёрка | Иванов А.А. | 2026-06-01" {
		t.Fatalf("name must use local date 2026-06-01, got %q", out.Name)
	}
}

func TestApplyMeetingUpdate_PartialDateTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")

	_, err := command.ApplyMeetingUpdate(baseMeeting(), UpdateMeetingInput{
		Date: strp("2026-06-02"), Start: strp("10:00"),
	}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput for partial date/time, got %v", err)
	}
}
