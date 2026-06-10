package application

import (
	"context"
	"errors"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type fakeUserSettingsStore struct {
	bu        postgres.BotUser
	getErr    error
	setCSV    string
	setErr    error
	setCalled bool
}

func (f *fakeUserSettingsStore) GetBotUserByTelegramID(_ context.Context, _ int64) (postgres.BotUser, error) {
	if f.getErr != nil {
		return postgres.BotUser{}, f.getErr
	}
	return f.bu, nil
}

func (f *fakeUserSettingsStore) SetReminderMinutes(_ context.Context, _ int64, csv string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.setCSV = csv
	f.setCalled = true
	return nil
}

func TestSetUserReminderMinutes_WhitelistOK(t *testing.T) {
	f := &fakeUserSettingsStore{}
	err := setUserReminderMinutes(context.Background(), f, 42, []int{10, 30})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !f.setCalled || f.setCSV != "10,30" {
		t.Fatalf("expected CSV '10,30', got '%s' (called=%v)", f.setCSV, f.setCalled)
	}
}

func TestSetUserReminderMinutes_RejectsNonWhitelist(t *testing.T) {
	f := &fakeUserSettingsStore{}
	err := setUserReminderMinutes(context.Background(), f, 42, []int{7})
	if !errors.Is(err, ErrInvalidReminderMinute) {
		t.Fatalf("expected ErrInvalidReminderMinute, got %v", err)
	}
	if f.setCalled {
		t.Fatalf("store must not be written on validation failure")
	}
}

func TestSetUserReminderMinutes_DedupeSort(t *testing.T) {
	f := &fakeUserSettingsStore{}
	if err := setUserReminderMinutes(context.Background(), f, 42, []int{30, 10, 30}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if f.setCSV != "10,30" {
		t.Fatalf("expected '10,30', got '%s'", f.setCSV)
	}
}

func TestSetUserReminderMinutes_Empty(t *testing.T) {
	f := &fakeUserSettingsStore{}
	if err := setUserReminderMinutes(context.Background(), f, 42, []int{}); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !f.setCalled || f.setCSV != "" {
		t.Fatalf("expected empty CSV, got '%s' (called=%v)", f.setCSV, f.setCalled)
	}
}

func TestGetUserSettings_ParsesCSV(t *testing.T) {
	f := &fakeUserSettingsStore{bu: postgres.BotUser{ReminderMinutes: "15,60"}}
	got, err := getUserSettings(context.Background(), f, 42)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got.ReminderMinutes) != 2 || got.ReminderMinutes[0] != 15 || got.ReminderMinutes[1] != 60 {
		t.Fatalf("expected [15,60], got %v", got.ReminderMinutes)
	}
}
