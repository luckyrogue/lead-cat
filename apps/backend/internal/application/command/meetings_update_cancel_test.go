package command

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func strPtr(s string) *string { return &s }

// seedStored puts a scheduled meeting (with a calendar event id) in the fake
// store, organized by organizerID, and sets the org owner.
func seedStored(fs *fakeStore, organizerID uuid.UUID) (uuid.UUID, model.Organization) {
	owner := uuid.New()
	org := model.Organization{TZ: "Asia/Almaty", OwnerUserID: &owner}
	fs.org = org
	id := uuid.New()
	org2 := organizerID
	fs.meetings[id] = model.Meeting{
		ID: id, Dept: "Eng", Type: "Sync", Host: "Mia",
		StartsAt:        time.Date(2026, 6, 1, 5, 0, 0, 0, time.UTC),
		EndsAt:          time.Date(2026, 6, 1, 5, 30, 0, 0, time.UTC),
		Recurrence:      "once",
		Status:          "scheduled",
		GoogleEventID:   "evt-seed",
		OrganizerUserID: &org2,
	}
	return id, org
}

func TestUpdateMeeting_HappyPath_ByOrganizer(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	out, err := c.UpdateMeeting(context.Background(), uuid.New(), organizer, id, UpdateInput{Dept: strPtr("Sales")})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if out.Dept != "Sales" {
		t.Fatalf("want Dept=Sales, got %q", out.Dept)
	}
	if len(fcs.updated) != 1 || fcs.updated[0] != "evt-seed" {
		t.Fatalf("calendar UpdateEvent not called for seeded event: %+v", fcs.updated)
	}
	if len(fs.updated) != 1 {
		t.Fatalf("Store.UpdateMeeting not called: %d", len(fs.updated))
	}
	if len(fq.updatedEnq) != 1 {
		t.Fatalf("updated job not enqueued: %d", len(fq.updatedEnq))
	}
}

func TestUpdateMeeting_PermissionDenied(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	stranger := uuid.New()
	_, err := c.UpdateMeeting(context.Background(), uuid.New(), stranger, id, UpdateInput{Dept: strPtr("Sales")})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if len(fcs.updated) != 0 || len(fs.updated) != 0 || len(fq.updatedEnq) != 0 {
		t.Fatalf("no mutation should occur on permission denial")
	}
}

func TestCancelMeeting_HappyPath_ByOrganizer(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	if err := c.CancelMeeting(context.Background(), uuid.New(), organizer, id); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(fcs.deleted) != 1 || fcs.deleted[0] != "evt-seed" {
		t.Fatalf("calendar DeleteEvent not called: %+v", fcs.deleted)
	}
	if len(fs.cancelled) != 1 {
		t.Fatalf("Store.CancelMeeting not called: %d", len(fs.cancelled))
	}
	if len(fq.cancelledEnq) != 1 {
		t.Fatalf("cancelled job not enqueued: %d", len(fq.cancelledEnq))
	}
}

func TestCancelMeeting_PermissionDenied(t *testing.T) {
	fs := newFakeStore()
	organizer := uuid.New()
	id, _ := seedStored(fs, organizer)
	fcs := &fakeCalService{}
	fq := &fakeQueue{}
	c := &Meetings{Store: fs, Calendar: &fakeCalProvider{svc: fcs}, Queue: fq, Log: zap.NewNop()}

	if err := c.CancelMeeting(context.Background(), uuid.New(), uuid.New(), id); !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
	if len(fs.cancelled) != 0 || len(fq.cancelledEnq) != 0 {
		t.Fatalf("no mutation should occur on permission denial")
	}
}
