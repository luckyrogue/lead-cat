package application

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

// almatyLoc is the base timezone for free-slot windows (UTC+5). §4.8
var almatyLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Almaty", 5*60*60)
	}
	return loc
}()

const (
	workStartHour = 9  // 09:00 working-day start (Almaty). §4.8
	workEndHour   = 18 // 18:00 working-day end (Almaty).
)

// Conflict is one participant's overlapping meeting. §4.7.2
type Conflict struct {
	Email       string
	PersonName  string
	MeetingName string
	Start, End  time.Time // UTC
}

// FreeSlot is a window where all queried participants are free. §4.8.4
type FreeSlot struct {
	Day        time.Time // start-of-day in Almaty
	Start, End time.Time // UTC
	Mins       int
}

// MeetingConflicts returns overlaps with [start,end) across emails (participants +
// organizer of each overlapping meeting), excluding excludeMeetingID (uuid.Nil =
// none). Attribution is done in Go: per overlapping meeting we load its participants
// and organizer email and keep those in the queried set. Global by email. §4.7
func (s *Services) MeetingConflicts(ctx context.Context, emails []string, start, end time.Time, excludeMeetingID uuid.UUID) ([]Conflict, error) {
	if len(emails) == 0 {
		return nil, nil
	}
	ms, err := s.Store.ListMeetingsOverlapping(ctx, emails, start, end)
	if err != nil {
		return nil, err
	}
	want := make(map[string]bool, len(emails))
	for _, e := range emails {
		want[e] = true
	}
	var out []Conflict
	for _, m := range ms {
		if m.ID == excludeMeetingID {
			continue
		}

		if !meeting.Overlaps(start, end, m.StartsAt, m.EndsAt) {
			continue
		}
		hit := map[string]bool{}
		parts, perr := s.Store.ListParticipants(ctx, m.ID)
		if perr != nil {
			return nil, perr
		}
		for _, p := range parts {
			if want[p.Email] {
				hit[p.Email] = true
			}
		}
		if m.OrganizerUserID != nil {
			if u, uerr := s.Store.GetUserByID(ctx, *m.OrganizerUserID); uerr == nil && want[u.Email] {
				hit[u.Email] = true
			}
		}
		for email := range hit {
			out = append(out, Conflict{
				Email:       email,
				PersonName:  s.personName(ctx, email),
				MeetingName: m.Name,
				Start:       m.StartsAt,
				End:         m.EndsAt,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Start.Equal(out[j].Start) {
			return out[i].Email < out[j].Email
		}
		return out[i].Start.Before(out[j].Start)
	})
	return out, nil
}

// MeetingUpdateConflicts resolves a pending single-meeting edit's effective time,
// participants and organizer, then checks §4.7 conflicts (excluding the meeting
// itself). Returns nil when the edit does not change the time (overlap unchanged).
func (s *Services) MeetingUpdateConflicts(ctx context.Context, organizationID, meetingID uuid.UUID, in UpdateMeetingInput) ([]Conflict, error) {
	if in.Date == nil || in.Start == nil || in.End == nil {

		return nil, nil
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(orDefault(w.TZ, "Asia/Almaty"))
	if err != nil {
		loc = almatyLoc
	}
	start, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.Start, loc)
	if err != nil {
		return nil, nil
	}
	end, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.End, loc)
	if err != nil {
		return nil, nil
	}
	emails, err := s.meetingEmails(ctx, organizationID, meetingID)
	if err != nil {
		return nil, err
	}
	return s.MeetingConflicts(ctx, emails, start.UTC(), end.UTC(), meetingID)
}

// meetingEmails returns a meeting's participant emails plus its organizer email.
func (s *Services) meetingEmails(ctx context.Context, organizationID, meetingID uuid.UUID) ([]string, error) {
	m, err := s.Store.GetMeeting(ctx, organizationID, meetingID)
	if err != nil {
		return nil, err
	}
	parts, err := s.Store.ListParticipants(ctx, meetingID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var emails []string
	add := func(e string) {
		if e != "" && !seen[e] {
			seen[e] = true
			emails = append(emails, e)
		}
	}
	for _, p := range parts {
		add(p.Email)
	}
	if m.OrganizerUserID != nil {
		if u, uerr := s.Store.GetUserByID(ctx, *m.OrganizerUserID); uerr == nil {
			add(u.Email)
		}
	}
	return emails, nil
}

// personName resolves a display name for an email (best-effort; falls back to email).
func (s *Services) personName(ctx context.Context, email string) string {
	matches, err := s.Store.SearchEmployeesGlobal(ctx, email)
	if err == nil {
		for _, e := range matches {
			if e.Email == email {
				return e.FullName
			}
		}
	}
	return email
}

// FreeSlots finds windows where ALL emails are free within [from,to) (day-exclusive),
// Mon–Fri, workStartHour–workEndHour Almaty, gaps >= durMins. Global by email. §4.8
// Callers should pass start-of-day Almaty boundaries for from/to so day windows align.
func (s *Services) FreeSlots(ctx context.Context, emails []string, from, to time.Time, durMins int) ([]FreeSlot, error) {
	if len(emails) == 0 || durMins <= 0 {
		return nil, nil
	}
	ms, err := s.Store.ListMeetingsOverlapping(ctx, emails, from, to)
	if err != nil {
		return nil, err
	}
	busy := make([]meeting.Span, 0, len(ms))
	for _, m := range ms {
		busy = append(busy, meeting.Span{Start: m.StartsAt, End: m.EndsAt})
	}
	minDur := time.Duration(durMins) * time.Minute

	var out []FreeSlot
	for day := from.In(almatyLoc); day.Before(to); day = day.AddDate(0, 0, 1) {
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		sod := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, almatyLoc)
		winStart := time.Date(day.Year(), day.Month(), day.Day(), workStartHour, 0, 0, 0, almatyLoc)
		winEnd := time.Date(day.Year(), day.Month(), day.Day(), workEndHour, 0, 0, 0, almatyLoc)
		var dayBusy []meeting.Span
		for _, b := range busy {
			if meeting.Overlaps(b.Start, b.End, winStart, winEnd) {
				dayBusy = append(dayBusy, b)
			}
		}
		for _, f := range meeting.FreeSlots(dayBusy, winStart, winEnd, minDur) {
			out = append(out, FreeSlot{Day: sod, Start: f.Start, End: f.End, Mins: int(f.End.Sub(f.Start).Minutes())})
		}
	}
	return out, nil
}
