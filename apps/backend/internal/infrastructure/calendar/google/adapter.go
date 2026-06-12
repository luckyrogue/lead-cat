package google

import (
	"context"
	"time"

	"github.com/google/uuid"
	calendar "google.golang.org/api/calendar/v3"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type adapter struct {
	svc        *calendar.Service
	calendarID string
}

func buildEvent(e docalendar.CalendarEvent, requestID string) *calendar.Event {
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

func (a *adapter) CreateEvent(ctx context.Context, e docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	created, err := a.svc.Events.
		Insert(a.calendarID, buildEvent(e, uuid.NewString())).
		ConferenceDataVersion(1).
		Context(ctx).
		Do()
	if err != nil {
		return docalendar.CalendarResult{}, err
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
	return docalendar.CalendarResult{EventID: created.Id, MeetLink: link}, nil
}

func (a *adapter) DeleteEvent(ctx context.Context, eventID string) error {
	return a.svc.Events.Delete(a.calendarID, eventID).Context(ctx).Do()
}

func buildPatch(e docalendar.CalendarEvent) *calendar.Event {
	return &calendar.Event{
		Summary:     e.Title,
		Description: e.Description,
		Start:       &calendar.EventDateTime{DateTime: e.Start.Format(time.RFC3339)},
		End:         &calendar.EventDateTime{DateTime: e.End.Format(time.RFC3339)},
	}
}

func (a *adapter) UpdateEvent(ctx context.Context, eventID string, e docalendar.CalendarEvent) error {
	_, err := a.svc.Events.
		Patch(a.calendarID, eventID, buildPatch(e)).
		SendUpdates("all").
		Context(ctx).
		Do()
	return err
}

func attendeeList(emails []string) []*calendar.EventAttendee {
	var as []*calendar.EventAttendee
	for _, e := range emails {
		as = append(as, &calendar.EventAttendee{Email: e})
	}
	return as
}

func (a *adapter) UpdateAttendees(ctx context.Context, eventID string, emails []string) error {
	ev := &calendar.Event{Attendees: attendeeList(emails), ForceSendFields: []string{"Attendees"}}
	_, err := a.svc.Events.Patch(a.calendarID, eventID, ev).SendUpdates("all").Context(ctx).Do()
	return err
}
