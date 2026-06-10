package application

import (
	"errors"
	"testing"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func occ() model.Meeting {
	loc, _ := time.LoadLocation("Asia/Almaty")
	return model.Meeting{
		Dept: "Разработка", Type: "Планёрка", Host: "Иванов",
		StartsAt:   time.Date(2026, 6, 8, 14, 0, 0, 0, loc).UTC(),
		EndsAt:     time.Date(2026, 6, 8, 15, 0, 0, 0, loc).UTC(),
		Recurrence: "weekly", Description: "d", Name: "old",
	}
}

func TestApplySeriesUpdate_FieldOnly(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applySeriesUpdate(occ(), SeriesUpdateInput{Dept: strp("Маркетинг")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if out.Dept != "Маркетинг" {
		t.Fatalf("dept=%q", out.Dept)
	}
	if out.Name != "Маркетинг | Планёрка | Иванов | 2026-06-08 | Еженедельно" {
		t.Fatalf("name=%q", out.Name)
	}
	if !out.StartsAt.Equal(occ().StartsAt) {
		t.Fatal("times must be unchanged when no time override")
	}
}

func TestApplySeriesUpdate_TimeOfDayKeepsDate(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	out, err := applySeriesUpdate(occ(), SeriesUpdateInput{Start: strp("10:00"), End: strp("11:00")}, loc)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, 6, 8, 10, 0, 0, 0, loc).UTC()
	if !out.StartsAt.Equal(wantStart) {
		t.Fatalf("start=%v want %v", out.StartsAt, wantStart)
	}
	if !out.EndsAt.Equal(time.Date(2026, 6, 8, 11, 0, 0, 0, loc).UTC()) {
		t.Fatalf("end=%v", out.EndsAt)
	}
}

func TestApplySeriesUpdate_BadTime(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	_, err := applySeriesUpdate(occ(), SeriesUpdateInput{Start: strp("15:00"), End: strp("14:00")}, loc)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("want ErrInvalidInput, got %v", err)
	}
}
