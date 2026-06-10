package application

import (
	"fmt"
	"time"

	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// UpdateMeetingInput carries optional field overrides (nil = leave unchanged).
// Date/Start/End must be supplied together to change the time.
// Participant changes are not handled here (separate increment).
type UpdateMeetingInput struct {
	Dept        *string
	Type        *string
	Host        *string
	Date        *string // YYYY-MM-DD
	Start       *string // HH:MM
	End         *string // HH:MM
	Recurrence  *string
	Description *string
}

// applyMeetingUpdate applies non-nil overrides to cur, parsing date/time in loc,
// validating via the domain, and recomputing the name (from the LOCAL start, as
// CreateMeeting does). Pure; returns a patched copy or ErrInvalidInput.
func applyMeetingUpdate(cur postgres.Meeting, in UpdateMeetingInput, loc *time.Location) (postgres.Meeting, error) {
	dept := orStr(in.Dept, cur.Dept)
	typ := orStr(in.Type, cur.Type)
	host := orStr(in.Host, cur.Host)
	desc := orStr(in.Description, cur.Description)
	rec := meeting.Recurrence(orStr(in.Recurrence, cur.Recurrence))

	startLocal := cur.StartsAt.In(loc)
	startsAt := cur.StartsAt
	endsAt := cur.EndsAt

	hasDate := in.Date != nil || in.Start != nil || in.End != nil
	allDate := in.Date != nil && in.Start != nil && in.End != nil
	if hasDate && !allDate {
		return postgres.Meeting{}, fmt.Errorf("%w: date, start, and end must be supplied together", ErrInvalidInput)
	}

	if allDate {
		s, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.Start, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad start time", ErrInvalidInput)
		}
		e, err := time.ParseInLocation("2006-01-02 15:04", *in.Date+" "+*in.End, loc)
		if err != nil {
			return postgres.Meeting{}, fmt.Errorf("%w: bad end time", ErrInvalidInput)
		}
		startLocal = s
		startsAt = s.UTC()
		endsAt = e.UTC()
	}

	dom := meeting.Input{Dept: dept, Type: typ, Host: host, StartsAt: startsAt, EndsAt: endsAt, Recurrence: rec, Description: desc}
	if err := dom.Validate(); err != nil {
		return postgres.Meeting{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	out := cur
	out.Dept, out.Type, out.Host = dept, typ, host
	out.Description = desc
	out.Recurrence = string(rec)
	out.StartsAt, out.EndsAt = startsAt, endsAt
	out.Name = meeting.GenerateName(dept, typ, host, startLocal, rec)
	return out, nil
}

func orStr(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}
