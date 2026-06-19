package application

import (
	"context"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
)

type BookingSlot struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

func loadLoc(name string) *time.Location {
	if name == "" {
		return almatyLoc
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return almatyLoc
}

func weekdaySet(days []int) map[int]bool {
	m := make(map[int]bool, len(days))
	for _, d := range days {
		m[d] = true
	}
	return m
}

func (s *Services) BookingAvailability(ctx context.Context, et model.BookingEventType, from, to time.Time) ([]BookingSlot, error) {
	if et.DurationMins <= 0 {
		return nil, nil
	}
	host, ok, err := s.Store.GetPlatformUserByID(ctx, et.HostUserID)
	if err != nil || !ok || host.Email == "" {
		return nil, err
	}
	loc := loadLoc(et.Timezone)
	allowed := weekdaySet(et.AvailWeekdays)
	dur := time.Duration(et.DurationMins) * time.Minute

	ms, err := s.Store.ListMeetingsOverlapping(ctx, []string{host.Email}, from, to)
	if err != nil {
		return nil, err
	}
	busy := make([]meeting.Span, 0, len(ms))
	for _, m := range ms {
		busy = append(busy, meeting.Span{Start: m.StartsAt, End: m.EndsAt})
	}
	busy = append(busy, s.gatherExternalBusy(ctx, host.Email, []string{host.Email}, from, to)[host.Email]...)

	now := from
	var out []BookingSlot
	for day := from.In(loc); day.Before(to); day = day.AddDate(0, 0, 1) {
		iso := int(day.Weekday())
		if iso == 0 {
			iso = 7
		}
		if !allowed[iso] {
			continue
		}
		y, mo, d := day.Date()
		winStart := time.Date(y, mo, d, 0, 0, 0, 0, loc).Add(time.Duration(et.AvailStartMinute) * time.Minute)
		winEnd := time.Date(y, mo, d, 0, 0, 0, 0, loc).Add(time.Duration(et.AvailEndMinute) * time.Minute)
		var dayBusy []meeting.Span
		for _, b := range busy {
			if meeting.Overlaps(b.Start, b.End, winStart, winEnd) {
				dayBusy = append(dayBusy, b)
			}
		}
		for _, f := range meeting.FreeSlots(dayBusy, winStart, winEnd, dur) {
			if !f.Start.After(now) {
				continue
			}
			out = append(out, BookingSlot{Start: f.Start.UTC(), End: f.End.UTC()})
		}
	}
	return out, nil
}
