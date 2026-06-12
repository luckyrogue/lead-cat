package application

import (
	"context"

	"github.com/google/uuid"
)

type miniAppOrgResolver interface {
	EnsureDefaultOrganizationID(ctx context.Context, defaultTZ, defaultMeetLink string, ownerUserID uuid.UUID) (uuid.UUID, error)
	GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error)
}

func (s *Services) ResolveMiniAppOrganization(ctx context.Context) (uuid.UUID, error) {
	return resolveMiniAppOrganization(ctx, s.Store)
}

func resolveMiniAppOrganization(ctx context.Context, store miniAppOrgResolver) (uuid.UUID, error) {
	id, err := ensureDefaultOrganization(ctx, store, uuid.Nil)
	if err != nil {
		return uuid.Nil, err
	}
	enc, subject, _, err := store.GetGoogleConfig(ctx, id)
	if err != nil {
		return uuid.Nil, err
	}
	if len(enc) == 0 || subject == "" {
		return uuid.Nil, ErrGoogleNotConfigured
	}
	return id, nil
}
