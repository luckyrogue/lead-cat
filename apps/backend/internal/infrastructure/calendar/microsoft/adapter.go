package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

const graphTimeLayout = "2006-01-02T15:04:05"

type adapter struct {
	httpClient *http.Client
	baseURL    string
}

func newAdapter(httpClient *http.Client, baseURL string) *adapter {
	return &adapter{httpClient: httpClient, baseURL: baseURL}
}

func graphTime(t time.Time) graphDateTime {
	return graphDateTime{DateTime: t.UTC().Format(graphTimeLayout), TimeZone: "UTC"}
}

func attendees(emails []string) []graphAttendee {
	out := make([]graphAttendee, 0, len(emails))
	for _, e := range emails {
		out = append(out, graphAttendee{EmailAddress: graphEmail{Address: e}, Type: "required"})
	}
	return out
}

func (a *adapter) CreateEvent(ctx context.Context, e docalendar.CalendarEvent) (docalendar.CalendarResult, error) {
	body := graphEvent{
		Subject:               e.Title,
		Body:                  &graphBody{ContentType: "text", Content: e.Description},
		Start:                 graphTime(e.Start),
		End:                   graphTime(e.End),
		Attendees:             attendees(e.AttendeeEmails),
		IsOnlineMeeting:       true,
		OnlineMeetingProvider: "teamsForBusiness",
	}
	var resp graphEvent
	if err := a.doJSON(ctx, http.MethodPost, "/me/events", body, &resp); err != nil {
		return docalendar.CalendarResult{}, err
	}
	link := ""
	if resp.OnlineMeeting != nil {
		link = resp.OnlineMeeting.JoinURL
	}
	return docalendar.CalendarResult{EventID: resp.ID, MeetLink: link}, nil
}

func (a *adapter) UpdateEvent(ctx context.Context, eventID string, e docalendar.CalendarEvent) error {
	body := graphEvent{Subject: e.Title, Body: &graphBody{ContentType: "text", Content: e.Description}, Start: graphTime(e.Start), End: graphTime(e.End)}
	return a.doJSON(ctx, http.MethodPatch, "/me/events/"+eventID, body, nil)
}

func (a *adapter) UpdateAttendees(ctx context.Context, eventID string, emails []string) error {
	return a.doJSON(ctx, http.MethodPatch, "/me/events/"+eventID, map[string]any{"attendees": attendees(emails)}, nil)
}

func (a *adapter) DeleteEvent(ctx context.Context, eventID string) error {
	return a.doJSON(ctx, http.MethodDelete, "/me/events/"+eventID, nil, nil)
}

func (a *adapter) BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]docalendar.Interval, error) {
	body := map[string]any{
		"schedules":                emails,
		"startTime":                graphTime(from),
		"endTime":                  graphTime(to),
		"availabilityViewInterval": 30,
	}
	var resp graphScheduleResponse
	if err := a.doJSON(ctx, http.MethodPost, "/me/calendar/getSchedule", body, &resp); err != nil {
		return nil, err
	}
	out := make(map[string][]docalendar.Interval, len(resp.Value))
	for _, s := range resp.Value {
		for _, it := range s.ScheduleItems {
			start, _ := time.Parse(graphTimeLayout, trimFraction(it.Start.DateTime))
			end, _ := time.Parse(graphTimeLayout, trimFraction(it.End.DateTime))
			out[s.ScheduleID] = append(out[s.ScheduleID], docalendar.Interval{Start: start, End: end})
		}
	}
	return out, nil
}

func (a *adapter) doJSON(ctx context.Context, method, path string, in, out any) error {
	var reader io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.baseURL+path, reader)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var ge graphErrorEnvelope
		raw, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(raw, &ge)
		return fmt.Errorf("graph %s: %s", resp.Status, ge.Error.Code)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

var _ docalendar.Service = (*adapter)(nil)
var _ docalendar.BusyReader = (*adapter)(nil)
