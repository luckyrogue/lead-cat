// Package google is the real Google Calendar adapter. It impersonates a
// workspace's configured subject (domain-wide delegation) and creates events
// with a Google Meet conference link.
package google

import (
	"context"
	"time"

	"github.com/google/uuid"
	calendar "google.golang.org/api/calendar/v3"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

type adapter struct {
	svc        *calendar.Service
	calendarID string
}

// buildEvent maps a transport-agnostic CalendarEvent to a Google event with a
// Meet create-request. requestID must be unique per insert.
func buildEvent(e application.CalendarEvent, requestID string) *calendar.Event {
	var attendees []*calendar.EventAttendee
	for _, em := range e.AttendeeEmails {
		attendees = append(attendees, &calendar.EventAttendee{Email: em})
	}
	return &calendar.Event{
		Summary:     e.Title,
		Description: e.Description,
		Start:       &calendar.EventDateTime{DateTime: e.Start.Format(time.RFC3339)},
		End:         &calendar.EventDateTime{DateTime: e.End.Format(time.RFC3339)},
		Attendees:   attendees,
		ConferenceData: &calendar.ConferenceData{
			CreateRequest: &calendar.CreateConferenceRequest{
				RequestId:             requestID,
				ConferenceSolutionKey: &calendar.ConferenceSolutionKey{Type: "hangoutsMeet"},
			},
		},
	}
}

func (a *adapter) CreateEvent(ctx context.Context, e application.CalendarEvent) (application.CalendarResult, error) {
	created, err := a.svc.Events.
		Insert(a.calendarID, buildEvent(e, uuid.NewString())).
		ConferenceDataVersion(1).
		Context(ctx).
		Do()
	if err != nil {
		return application.CalendarResult{}, err
	}
	link := created.HangoutLink
	if link == "" && created.ConferenceData != nil {
		for _, ep := range created.ConferenceData.EntryPoints {
			if ep.EntryPointType == "video" {
				link = ep.Uri
				break
			}
		}
	}
	return application.CalendarResult{EventID: created.Id, MeetLink: link}, nil
}

func (a *adapter) DeleteEvent(ctx context.Context, eventID string) error {
	return a.svc.Events.Delete(a.calendarID, eventID).Context(ctx).Do()
}

// buildPatch maps a CalendarEvent to a partial Google event for Events.Patch.
// It sets only the fields edited here; omitting Attendees and ConferenceData
// leaves the guest list and the Meet link untouched.
func buildPatch(e application.CalendarEvent) *calendar.Event {
	return &calendar.Event{
		Summary:     e.Title,
		Description: e.Description,
		Start:       &calendar.EventDateTime{DateTime: e.Start.Format(time.RFC3339)},
		End:         &calendar.EventDateTime{DateTime: e.End.Format(time.RFC3339)},
	}
}

func (a *adapter) UpdateEvent(ctx context.Context, eventID string, e application.CalendarEvent) error {
	_, err := a.svc.Events.
		Patch(a.calendarID, eventID, buildPatch(e)).
		SendUpdates("all").
		Context(ctx).
		Do()
	return err
}
