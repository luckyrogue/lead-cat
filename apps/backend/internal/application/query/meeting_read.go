package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type MiniAppMeeting struct {
	ID              string
	Type            string
	Dept            string
	Host            string
	Date            string
	Start           string
	End             string
	Rec             string
	SeriesID        string
	RecurrenceUntil string
	Organizer       string
	Participants    []string
	Desc            string
	MeetLink        string
	Status          string
}

type meetingStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error)
}

func MeetingDTO(ctx context.Context, store meetingStore, m model.Meeting, loc *time.Location) MiniAppMeeting {
	s := m.StartsAt.In(loc)
	e := m.EndsAt.In(loc)
	organizer := ""
	if m.OrganizerUserID != nil {
		if u, err := store.GetUserByID(ctx, *m.OrganizerUserID); err == nil {
			organizer = u.Email
		}
	}
	emails := []string{}
	if parts, err := store.ListParticipants(ctx, m.ID); err == nil {
		for _, p := range parts {
			if p.Email != "" {
				emails = append(emails, p.Email)
			}
		}
	}
	seriesID := ""
	if m.SeriesID != nil {
		seriesID = m.SeriesID.String()
	}
	recurrenceUntil := ""
	if m.RecurrenceUntil != nil {
		recurrenceUntil = m.RecurrenceUntil.In(loc).Format("2006-01-02")
	}
	return MiniAppMeeting{
		ID: m.ID.String(), Type: m.Type, Dept: m.Dept, Host: m.Host,
		Date: s.Format("2006-01-02"), Start: s.Format("15:04"), End: e.Format("15:04"),
		Rec: m.Recurrence, SeriesID: seriesID, RecurrenceUntil: recurrenceUntil,
		Organizer: organizer, Participants: emails,
		Desc: m.Description, MeetLink: m.MeetLink, Status: m.Status,
	}
}
