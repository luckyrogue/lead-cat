package application

import (
	"context"

	"github.com/google/uuid"
)

const (
	defaultOrganizationTZ       = "Asia/Almaty"
	defaultOrganizationMeetLink = ""
)

type organizationEnsurer interface {
	EnsureDefaultOrganizationID(ctx context.Context, tz, meetLink string, ownerID uuid.UUID) (uuid.UUID, error)
}

func (s *Services) EnsureDefaultOrganization(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, error) {
	return ensureDefaultOrganization(ctx, s.Store, ownerID)
}

func ensureDefaultOrganization(ctx context.Context, store organizationEnsurer, ownerID uuid.UUID) (uuid.UUID, error) {
	return store.EnsureDefaultOrganizationID(ctx, defaultOrganizationTZ, defaultOrganizationMeetLink, ownerID)
}
