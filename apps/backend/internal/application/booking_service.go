package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type EventTypeInput = command.EventTypeInput

var ErrInvalidEventType = command.ErrInvalidEventType

func (s *Services) CreateEventType(ctx context.Context, hostUserID, orgID uuid.UUID, in EventTypeInput) (model.BookingEventType, error) {
	return s.BookingCommands.CreateEventType(ctx, hostUserID, orgID, in)
}

func (s *Services) UpdateEventType(ctx context.Context, hostUserID, id uuid.UUID, in EventTypeInput) error {
	return s.BookingCommands.UpdateEventType(ctx, hostUserID, id, in)
}

func (s *Services) DeleteEventType(ctx context.Context, hostUserID, id uuid.UUID) error {
	return s.BookingCommands.DeleteEventType(ctx, hostUserID, id)
}

func (s *Services) ListMyEventTypes(ctx context.Context, hostUserID uuid.UUID) ([]model.BookingEventType, error) {
	return s.BookingQueries.ListMyEventTypes(ctx, hostUserID)
}

func (s *Services) GetBookingEventTypeBySlugPublic(ctx context.Context, slug string) (model.BookingEventType, error) {
	return s.BookingQueries.GetBookingEventTypeBySlugPublic(ctx, slug)
}
