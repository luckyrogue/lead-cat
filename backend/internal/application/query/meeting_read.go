package query

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// MiniAppMeeting is the Mini App list/detail meeting shape.
type MiniAppMeeting struct {
	ID           string
	Type         string
	Dept         string
	Host         string
	Date         string
	Start        string
	End          string
	Rec          string
	Organizer    string
	Participants []string
	Desc         string
	MeetLink     string
	Status       string
}

type meetingStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (postgres.User, error)
	ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]postgres.MeetingParticipant, error)
}

// MeetingDTO maps a meeting row to the TMA UI shape, resolving organizer and participant emails.
func MeetingDTO(ctx context.Context, store meetingStore, m postgres.Meeting, loc *time.Location) MiniAppMeeting {
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
	return MiniAppMeeting{
		ID: m.ID.String(), Type: m.Type, Dept: m.Dept, Host: m.Host,
		Date: s.Format("2006-01-02"), Start: s.Format("15:04"), End: e.Format("15:04"),
		Rec: m.Recurrence, Organizer: organizer, Participants: emails,
		Desc: m.Description, MeetLink: m.MeetLink, Status: m.Status,
	}
}
