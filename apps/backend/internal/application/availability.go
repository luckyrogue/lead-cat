package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/domain/meeting"
	"github.com/luckyrogue/lead-cat/internal/platform/fanio"
)

func (s *Services) gatherExternalBusy(ctx context.Context, requesterEmail string, emails []string, from, to time.Time) map[string][]meeting.Span {
	out := map[string][]meeting.Span{}
	if s.Busy == nil || len(emails) == 0 {
		return out
	}
	var mu sync.Mutex
	add := func(busy map[string][]docalendar.Interval) {
		mu.Lock()
		defer mu.Unlock()
		for email, ivs := range busy {
			for _, iv := range ivs {
				out[email] = append(out[email], meeting.Span{Start: iv.Start, End: iv.End})
			}
		}
	}
	if requesterEmail != "" {
		if reader, ok := s.Busy.ReaderFor(ctx, requesterEmail); ok {
			if busy, err := reader.BusyTimes(ctx, emails, from, to); err == nil {
				add(busy)
			} else {
				s.logExternalBusyFail(requesterEmail, err)
			}
		}
	}
	fanio.AllBestEffort(ctx, 4, len(emails), func(ctx context.Context, i int) {
		email := emails[i]
		reader, ok := s.Busy.ReaderFor(ctx, email)
		if !ok {
			return
		}
		busy, err := reader.BusyTimes(ctx, []string{email}, from, to)
		if err != nil {
			s.logExternalBusyFail(email, err)
			return
		}
		add(busy)
	})
	return dedupeSpans(out)
}

func dedupeSpans(in map[string][]meeting.Span) map[string][]meeting.Span {
	for email, spans := range in {
		sort.Slice(spans, func(i, j int) bool {
			if spans[i].Start.Equal(spans[j].Start) {
				return spans[i].End.Before(spans[j].End)
			}
			return spans[i].Start.Before(spans[j].Start)
		})
		deduped := spans[:0]
		for _, sp := range spans {
			if n := len(deduped); n > 0 && deduped[n-1].Start.Equal(sp.Start) && deduped[n-1].End.Equal(sp.End) {
				continue
			}
			deduped = append(deduped, sp)
		}
		in[email] = deduped
	}
	return in
}

func (s *Services) logExternalBusyFail(email string, err error) {
	if s.Log == nil {
		return
	}
	s.Log.Warn("external_busy_fetch_failed", zap.String("email_hash", hashEmail(email)), zap.Error(err))
}

func hashEmail(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])[:8]
}
