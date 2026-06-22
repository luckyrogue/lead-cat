package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type bookingStore interface {
	ListBookingEventTypesForUser(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error)
	GetBookingEventTypeBySlug(ctx context.Context, slug string) (model.BookingEventType, error)
}

type Bookings struct {
	Store bookingStore
}

func NewBookings(store bookingStore) *Bookings {
	return &Bookings{Store: store}
}

func (q *Bookings) ListMyEventTypes(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	return q.Store.ListBookingEventTypesForUser(ctx, hostUserID)
}

func (q *Bookings) GetBookingEventTypeBySlugPublic(ctx context.Context, slug string) (model.BookingEventType, error) {
	return q.Store.GetBookingEventTypeBySlug(ctx, slug)
}
