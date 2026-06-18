package resolver

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type connLister interface {
	ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error)
}

type calProvider interface {
	For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error)
}

type msFactory interface {
	For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool)
}

type Resolver struct {
	lister connLister
	google calProvider
	ms     msFactory
}

func New(lister connLister, google calProvider, ms msFactory) *Resolver {
	return &Resolver{lister: lister, google: google, ms: ms}
}

func (r *Resolver) For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error) {
	if organizerEmail != "" && r.lister != nil {
		if conns, err := r.lister.ListCalendarConnections(ctx, organizerEmail); err == nil {
			if best, ok := mostRecent(conns); ok && best.Provider == "microsoft" && r.ms != nil {
				if svc, built := r.ms.For(ctx, best); built {
					return svc, nil
				}
			}
		}
	}
	return r.google.For(ctx, organizationID, organizerEmail)
}

func mostRecent(conns []model.CalendarConnection) (model.CalendarConnection, bool) {
	var best model.CalendarConnection
	found := false
	for _, c := range conns {
		if !found || c.UpdatedAt.After(best.UpdatedAt) {
			best, found = c, true
		}
	}
	return best, found
}
