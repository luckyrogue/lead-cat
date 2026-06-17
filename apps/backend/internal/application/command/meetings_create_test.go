package command

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func newMeetingsCmd(fs *fakeStore, fcp *fakeCalProvider, fq *fakeQueue) *Meetings {
	return &Meetings{Store: fs, Calendar: fcp, Queue: fq, Log: zap.NewNop()}
}

func ownerOrg() (model.Organization, uuid.UUID) {
	owner := uuid.New()
	return model.Organization{TZ: "Asia/Almaty", OwnerUserID: &owner}, owner
}

func TestCreateMeeting_Once_HappyPath(t *testing.T) {
	fs := newFakeStore()
	org, owner := ownerOrg()
	fs.org = org
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, fq)

	m, err := c.CreateMeeting(context.Background(), uuid.New(), owner, CreateInput{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "once",
		Participants: []model.MeetingParticipant{{Email: "a@x.io"}},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(fcs.created) != 1 {
		t.Fatalf("want 1 calendar event, got %d", len(fcs.created))
	}
	if len(fs.created) != 1 || fs.created[0].GoogleEventID == "" || fs.created[0].MeetLink == "" {
		t.Fatalf("meeting not persisted with calendar ids: %+v", fs.created)
	}
	if len(fq.createdEnq) != 1 {
		t.Fatalf("want 1 enqueue, got %d", len(fq.createdEnq))
	}
	if m.StartsAt.Location() != time.UTC {
		t.Fatalf("starts_at should be stored UTC, got %v", m.StartsAt.Location())
	}
}

func TestCreateMeeting_Recurring_Series(t *testing.T) {
	fs := newFakeStore()
	org, owner := ownerOrg()
	fs.org = org
	fcs := &fakeCalService{}
	c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, &fakeQueue{})

	_, err := c.CreateMeeting(context.Background(), uuid.New(), owner, CreateInput{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		Date: "2026-06-01", Start: "10:00", End: "10:30",
		Recurrence: "daily", RecurrenceUntil: "2026-06-03",
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if len(fcs.created) != 3 {
		t.Fatalf("want 3 calendar events (Jun 1-3), got %d", len(fcs.created))
	}
	if len(fs.series) != 1 || len(fs.series[0]) != 3 {
		t.Fatalf("want CreateMeetingSeries with 3 meetings, got %+v", fs.series)
	}
}

func TestCreateMeeting_InvalidInput(t *testing.T) {
	cases := map[string]CreateInput{
		"bad-start":      {Dept: "E", Type: "S", Host: "M", Date: "2026-06-01", Start: "99:99", End: "10:30", Recurrence: "once"},
		"bad-recurrence": {Dept: "E", Type: "S", Host: "M", Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "fortnightly"},
		"bad-until":      {Dept: "E", Type: "S", Host: "M", Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "daily", RecurrenceUntil: "nope"},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			fs := newFakeStore()
			org, owner := ownerOrg()
			fs.org = org
			fcs := &fakeCalService{}
			fq := &fakeQueue{}
			c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, fq)
			_, err := c.CreateMeeting(context.Background(), uuid.New(), owner, in)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
			if len(fs.created) != 0 || len(fcs.created) != 0 || len(fq.createdEnq) != 0 {
				t.Fatalf("nothing should be created/enqueued on invalid input")
			}
		})
	}
}

func TestCreateMeeting_CalendarFailure(t *testing.T) {
	fs := newFakeStore()
	org, owner := ownerOrg()
	fs.org = org
	fcs := &fakeCalService{failCreate: true}
	fq := &fakeQueue{}
	c := newMeetingsCmd(fs, &fakeCalProvider{svc: fcs}, fq)

	_, err := c.CreateMeeting(context.Background(), uuid.New(), owner, CreateInput{
		Dept: "Eng", Type: "Sync", Host: "Mia",
		Date: "2026-06-01", Start: "10:00", End: "10:30", Recurrence: "once",
	})
	if err == nil || !strings.Contains(err.Error(), "calendar") {
		t.Fatalf("want calendar-wrapped error, got %v", err)
	}
	if len(fs.created) != 0 || len(fq.createdEnq) != 0 {
		t.Fatalf("nothing should persist/enqueue when calendar fails")
	}
}
