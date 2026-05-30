package botsettings

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

func TestParse(t *testing.T) {
	got := parse(" 60, 15 ,15, x, ,30 ")
	want := []int{15, 30, 60}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if len(parse("")) != 0 {
		t.Fatal("empty csv -> empty slice")
	}
}

func TestFormat(t *testing.T) {
	if format([]int{60, 15, 15}) != "15,60" {
		t.Fatalf("got %q", format([]int{60, 15, 15}))
	}
	if format(nil) != "" {
		t.Fatal("nil -> empty string")
	}
}

func TestToggle(t *testing.T) {
	if got := format(toggle([]int{15, 60}, 30)); got != "15,30,60" {
		t.Fatalf("add: %q", got)
	}
	if got := format(toggle([]int{15, 60}, 15)); got != "60" {
		t.Fatalf("remove: %q", got)
	}
}

func TestRender(t *testing.T) {
	text, kb := render([]int{15})
	if !strings.Contains(text, "Напоминан") {
		t.Fatalf("text: %q", text)
	}
	var flat []Button
	for _, row := range kb {
		flat = append(flat, row...)
	}
	if len(flat) != 6 {
		t.Fatalf("want 6 buttons, got %d", len(flat))
	}
	var on15 Button
	for _, b := range flat {
		if b.Data == "rem:15" {
			on15 = b
		}
	}
	if !strings.HasPrefix(on15.Text, "✓") {
		t.Fatalf("15 should be checked: %q", on15.Text)
	}
	emptyText, _ := render(nil)
	if !strings.Contains(emptyText, "выключены") {
		t.Fatalf("empty state text: %q", emptyText)
	}
}

type fakeStore struct {
	csv string
	err error
}

func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, _ int64) (postgres.BotUser, error) {
	if f.err != nil {
		return postgres.BotUser{}, f.err
	}
	return postgres.BotUser{ReminderMinutes: f.csv}, nil
}
func (f *fakeStore) SetReminderMinutes(_ context.Context, _ int64, csv string) error {
	f.csv = csv
	return nil
}

func TestServiceToggle(t *testing.T) {
	fs := &fakeStore{csv: "15"}
	svc := New(fs)
	if _, _, err := svc.Toggle(context.Background(), 1, 60); err != nil {
		t.Fatal(err)
	}
	if fs.csv != "15,60" {
		t.Fatalf("persisted %q", fs.csv)
	}
}

func TestServiceSettingsError(t *testing.T) {
	svc := New(&fakeStore{err: errors.New("db")})
	if _, _, err := svc.Settings(context.Background(), 1); err == nil {
		t.Fatal("expected error propagated")
	}
}
