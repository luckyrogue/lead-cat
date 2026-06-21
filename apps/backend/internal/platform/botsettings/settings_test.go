package botsettings

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

func TestParse_DedupSortDropInvalid(t *testing.T) {
	got := Parse(" 60, 15 ,15, x, 30,, 60 ")
	want := []int{15, 30, 60}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %v, want %v", got, want)
	}
	if len(Parse("")) != 0 {
		t.Fatalf("empty csv should parse to empty")
	}
}

func TestFormat_SortedCompactCSV(t *testing.T) {
	if got := Format([]int{60, 15, 15, 30}); got != "15,30,60" {
		t.Fatalf("Format = %q", got)
	}
	if got := Format(nil); got != "" {
		t.Fatalf("Format(nil) = %q, want empty", got)
	}
}

func TestToggle_AddAndRemove(t *testing.T) {
	on := toggle([]int{15, 60}, 30)
	if got := format(on); got != "15,30,60" {
		t.Fatalf("toggle add = %q", got)
	}
	off := toggle([]int{15, 30, 60}, 30)
	if got := format(off); got != "15,60" {
		t.Fatalf("toggle remove = %q", got)
	}
}

type fakeStore struct {
	user     postgres.BotUser
	savedCSV string
	saveErr  error
}

func (f *fakeStore) GetBotUserByTelegramID(_ context.Context, _ int64) (postgres.BotUser, error) {
	return f.user, nil
}
func (f *fakeStore) SetReminderMinutes(_ context.Context, _ int64, csv string) error {
	f.savedCSV = csv
	return f.saveErr
}

var _ store = (*fakeStore)(nil)

func TestService_Toggle_Persists(t *testing.T) {
	fs := &fakeStore{user: postgres.BotUser{ReminderMinutes: "15"}}
	s := New(fs)
	text, kb, err := s.Toggle(context.Background(), 1, 60, "ru")
	if err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if fs.savedCSV != "15,60" {
		t.Fatalf("persisted CSV = %q, want 15,60", fs.savedCSV)
	}
	if text == "" || len(kb) == 0 {
		t.Fatalf("expected rendered text + keyboard")
	}
}

func TestService_Settings_RendersCurrent(t *testing.T) {
	fs := &fakeStore{user: postgres.BotUser{ReminderMinutes: "30"}}
	text, kb, err := New(fs).Settings(context.Background(), 1, "ru")
	if err != nil || text == "" || len(kb) == 0 {
		t.Fatalf("settings: %v text=%q kb=%d", err, text, len(kb))
	}
}

func TestRender_Localized(t *testing.T) {
	ru, _ := render([]int{}, "ru")
	en, _ := render([]int{}, "en")
	if ru == en {
		t.Fatalf("render must differ by language; both = %q", ru)
	}
	if !strings.Contains(en, "Meeting reminders") || !strings.Contains(en, "currently off") {
		t.Errorf("en render = %q", en)
	}
	// interval label localized in the keyboard
	_, kb := render([]int{}, "en")
	if kb[0][0].Text != "10m" {
		t.Errorf("en first interval label = %q", kb[0][0].Text)
	}
}
