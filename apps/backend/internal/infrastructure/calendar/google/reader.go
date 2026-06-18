package google

import (
	"context"
	"time"

	"golang.org/x/oauth2"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type googleReader struct {
	svc *calendar.Service
}

func newGoogleReader(svc *calendar.Service) *googleReader { return &googleReader{svc: svc} }

func (r *googleReader) BusyTimes(ctx context.Context, emails []string, from, to time.Time) (map[string][]docalendar.Interval, error) {
	items := make([]*calendar.FreeBusyRequestItem, 0, len(emails))
	for _, e := range emails {
		items = append(items, &calendar.FreeBusyRequestItem{Id: e})
	}
	resp, err := r.svc.Freebusy.Query(&calendar.FreeBusyRequest{
		TimeMin: from.UTC().Format(time.RFC3339),
		TimeMax: to.UTC().Format(time.RFC3339),
		Items:   items,
	}).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]docalendar.Interval, len(resp.Calendars))
	for email, cal := range resp.Calendars {
		for _, b := range cal.Busy {
			start, _ := time.Parse(time.RFC3339, b.Start)
			end, _ := time.Parse(time.RFC3339, b.End)
			out[email] = append(out[email], docalendar.Interval{Start: start, End: end})
		}
	}
	return out, nil
}

type ReaderFactory struct {
	conns     connectionStore
	connector calendarConnector
}

func NewReaderFactory(conns connectionStore, connector calendarConnector) *ReaderFactory {
	return &ReaderFactory{conns: conns, connector: connector}
}

func (f *ReaderFactory) For(ctx context.Context, conn model.CalendarConnection) (docalendar.BusyReader, bool) {
	if f.connector == nil {
		return nil, false
	}
	cfg := f.connector.OAuthConfig("")
	base := cfg.TokenSource(ctx, &oauth2.Token{AccessToken: conn.AccessToken, RefreshToken: conn.RefreshToken, Expiry: conn.Expiry})
	src := &savingSource{base: oauth2.ReuseTokenSource(nil, base), save: func(tok *oauth2.Token) {
		conn.AccessToken, conn.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			conn.RefreshToken = tok.RefreshToken
		}
		_ = f.conns.UpsertCalendarConnection(ctx, conn)
	}}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, src)))
	if err != nil {
		return nil, false
	}
	return newGoogleReader(svc), true
}

var _ docalendar.BusyReader = (*googleReader)(nil)
