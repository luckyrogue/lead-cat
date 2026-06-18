package resolver

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type googleReaderFactory interface {
	For(ctx context.Context, conn model.CalendarConnection) (docalendar.BusyReader, bool)
}

type msReaderFactory interface {
	For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool)
}

type BusyResolver struct {
	lister connLister
	google googleReaderFactory
	ms     msReaderFactory
}

func NewBusyResolver(lister connLister, google googleReaderFactory, ms msReaderFactory) *BusyResolver {
	return &BusyResolver{lister: lister, google: google, ms: ms}
}

func (r *BusyResolver) ReaderFor(ctx context.Context, email string) (docalendar.BusyReader, bool) {
	if email == "" || r.lister == nil {
		return nil, false
	}
	conns, err := r.lister.ListCalendarConnections(ctx, email)
	if err != nil {
		return nil, false
	}
	best, ok := mostRecent(conns)
	if !ok {
		return nil, false
	}
	switch best.Provider {
	case "microsoft":
		if r.ms == nil {
			return nil, false
		}
		if svc, built := r.ms.For(ctx, best); built {
			if br, isReader := svc.(docalendar.BusyReader); isReader {
				return br, true
			}
		}
	case "google":
		if r.google == nil {
			return nil, false
		}
		return r.google.For(ctx, best)
	}
	return nil, false
}
