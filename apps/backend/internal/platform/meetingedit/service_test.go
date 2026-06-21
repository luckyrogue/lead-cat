package meetingedit

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// ---------------------------------------------------------------------------
// fakeBackend
// ---------------------------------------------------------------------------

type fakeBackend struct {
	mu       sync.Mutex
	meetings []postgres.MeetingWithTZ
	listErr  error

	canceled     []uuid.UUID
	added        []string
	removed      []string
	participants []postgres.MeetingParticipant
}

var _ Backend = (*fakeBackend)(nil)

func (f *fakeBackend) ListEditableMeetings(_ context.Context, _ int64) ([]postgres.MeetingWithTZ, error) {
	return f.meetings, f.listErr
}

func (f *fakeBackend) UpdateMeeting(_ context.Context, _, _, _ uuid.UUID, _ application.UpdateMeetingInput) (postgres.Meeting, error) {
	return postgres.Meeting{}, nil
}

func (f *fakeBackend) UpdateSeries(_ context.Context, _, _, _ uuid.UUID, _ application.SeriesUpdateInput) (int, error) {
	return 0, nil
}

func (f *fakeBackend) ListParticipants(_ context.Context, _ uuid.UUID) ([]postgres.MeetingParticipant, error) {
	return f.participants, nil
}

func (f *fakeBackend) SearchEmployees(_ context.Context, _ uuid.UUID, _ string) ([]postgres.Employee, error) {
	return nil, nil
}

func (f *fakeBackend) AddParticipant(_ context.Context, _, _, _ uuid.UUID, email string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.added = append(f.added, email)
	return nil
}

func (f *fakeBackend) RemoveParticipant(_ context.Context, _, _, _ uuid.UUID, email string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, email)
	return nil
}

func (f *fakeBackend) CancelMeeting(_ context.Context, _, _, meetingID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.canceled = append(f.canceled, meetingID)
	return nil
}

func (f *fakeBackend) CancelSeries(_ context.Context, _, _, meetingID uuid.UUID) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.canceled = append(f.canceled, meetingID)
	return 1, nil
}

func (f *fakeBackend) MeetingUpdateConflicts(_ context.Context, _, _ uuid.UUID, _ application.UpdateMeetingInput) ([]application.Conflict, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// fakeSessions
// ---------------------------------------------------------------------------

type fakeSessions struct {
	mu   sync.Mutex
	data map[int64]*State
}

var _ sessions = (*fakeSessions)(nil)

func newFakeSessions() *fakeSessions {
	return &fakeSessions{data: make(map[int64]*State)}
}

func (f *fakeSessions) Get(_ context.Context, telegramID int64) (*State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.data[telegramID], nil
}

func (f *fakeSessions) Set(_ context.Context, telegramID int64, s State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := s
	f.data[telegramID] = &cp
	return nil
}

func (f *fakeSessions) Del(_ context.Context, telegramID int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, telegramID)
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func makeMeetingWithTZ(id uuid.UUID, name string) postgres.MeetingWithTZ {
	orgID := uuid.New()
	userID := uuid.New()
	return postgres.MeetingWithTZ{
		Meeting: postgres.Meeting{
			ID:              id,
			OrganizationID:  orgID,
			OrganizerUserID: &userID,
			Name:            name,
			StartsAt:        time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
			EndsAt:          time.Date(2026, 7, 1, 11, 0, 0, 0, time.UTC),
			Recurrence:      "once",
		},
		TZ: "Asia/Almaty",
	}
}

// ---------------------------------------------------------------------------
// Tests: Start
// ---------------------------------------------------------------------------

func TestStart_ListsEditableMeetings(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	fb := &fakeBackend{
		meetings: []postgres.MeetingWithTZ{
			makeMeetingWithTZ(id1, "Meeting Alpha"),
			makeMeetingWithTZ(id2, "Meeting Beta"),
		},
	}
	svc := New(fb, newFakeSessions())

	reply := svc.Start(context.Background(), 12345, "ru")

	if reply.Text == "" {
		t.Fatal("expected non-empty Text")
	}
	if len(reply.Keyboard) != 2 {
		t.Fatalf("expected 2 keyboard rows, got %d", len(reply.Keyboard))
	}

	wantIDs := map[string]bool{
		id1.String(): false,
		id2.String(): false,
	}
	for _, row := range reply.Keyboard {
		if len(row) != 1 {
			t.Fatalf("expected 1 button per row, got %d", len(row))
		}
		btn := row[0]
		if !strings.HasPrefix(btn.Data, "medit:pick:") {
			t.Errorf("button Data %q does not start with medit:pick:", btn.Data)
		}
		btnID := strings.TrimPrefix(btn.Data, "medit:pick:")
		if _, ok := wantIDs[btnID]; !ok {
			t.Errorf("unexpected meeting ID in button: %q", btnID)
		}
		wantIDs[btnID] = true
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Errorf("meeting ID %s not found in keyboard buttons", id)
		}
	}
}

func TestStart_Empty(t *testing.T) {
	fb := &fakeBackend{meetings: nil}
	svc := New(fb, newFakeSessions())

	reply := svc.Start(context.Background(), 99, "ru")

	if reply.Text == "" {
		t.Fatal("expected non-empty Text for empty meetings list")
	}
	if len(reply.Keyboard) != 0 {
		t.Errorf("expected no keyboard buttons for empty list, got %d rows", len(reply.Keyboard))
	}
}

func TestStart_BackendError(t *testing.T) {
	fb := &fakeBackend{listErr: errors.New("db down")}
	svc := New(fb, newFakeSessions())

	reply := svc.Start(context.Background(), 77, "ru")

	if reply.Text == "" {
		t.Fatal("expected non-empty error Text")
	}
	if len(reply.Keyboard) != 0 {
		t.Errorf("expected no keyboard buttons on error, got %d rows", len(reply.Keyboard))
	}
}

// ---------------------------------------------------------------------------
// Test: pick → edit menu (medit:pick:<id>)
// ---------------------------------------------------------------------------

func TestPick_OpensEditMenu(t *testing.T) {
	id := uuid.New()
	m := makeMeetingWithTZ(id, "Stand-up Daily")
	// Non-series meeting: no SeriesID set, so scope is auto-set to "one".

	fb := &fakeBackend{meetings: []postgres.MeetingWithTZ{m}}
	sess := newFakeSessions()
	svc := New(fb, sess)

	reply, handled := svc.OnCallback(context.Background(), 42, "medit:pick:"+id.String(), "ru")

	if !handled {
		t.Fatal("expected OnCallback to handle medit:pick:*")
	}
	if reply.Text == "" {
		t.Fatal("expected non-empty menu Text after pick")
	}

	// The reply should contain the menu keyboard (apply / cancel buttons are canonical).
	found := false
	for _, row := range reply.Keyboard {
		for _, btn := range row {
			if btn.Data == "medit:apply" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected medit:apply button in edit menu keyboard")
	}

	// Session should be persisted with the meeting ID and scope "one".
	st, err := sess.Get(context.Background(), 42)
	if err != nil {
		t.Fatalf("session Get error: %v", err)
	}
	if st == nil {
		t.Fatal("expected session to be saved after pick")
	}
	if st.MeetingID != id.String() {
		t.Errorf("session MeetingID = %q; want %q", st.MeetingID, id.String())
	}
	if st.Scope != "one" {
		t.Errorf("session Scope = %q; want \"one\"", st.Scope)
	}
}

// TODO(WS2c): deeper medit:* flow (field → apply → confirm → series scope) not covered here.
// The pick→apply chain requires a fully-seeded State.Cur map (snapshot of a real Meeting)
// and the UpdateMeeting/UpdateSeries fakes to return shaped postgres.Meeting values.
// Drive those tests once the fake's UpdateMeeting is extended to return a canned meeting.

// ---------------------------------------------------------------------------
// Test: lang threading — ru vs en output differs
// ---------------------------------------------------------------------------

func TestMeetingEdit_Localized(t *testing.T) {
	id := uuid.New()
	m := makeMeetingWithTZ(id, "Meeting Alpha")

	newSvc := func() *Service {
		return New(&fakeBackend{meetings: []postgres.MeetingWithTZ{m}}, newFakeSessions())
	}

	ru := newSvc().Start(context.Background(), 1, "ru")
	en := newSvc().Start(context.Background(), 1, "en")
	if ru.Text == en.Text {
		t.Fatalf("Start must differ by language; both = %q", ru.Text)
	}
	if !strings.Contains(en.Text, "Pick a meeting") && !strings.Contains(en.Text, "No upcoming") {
		t.Errorf("en Start = %q", en.Text)
	}
}
