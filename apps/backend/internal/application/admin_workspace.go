package application

import (
	"context"

	"github.com/google/uuid"
)

const (
	defaultOrganizationTZ       = "Asia/Almaty"
	defaultOrganizationMeetLink = ""
)

// organizationEnsurer is the narrow store interface used by EnsureDefaultOrganization
// — defined here so unit tests can mock it.
type organizationEnsurer interface {
	EnsureDefaultOrganizationID(ctx context.Context, tz, meetLink string, ownerID uuid.UUID) (uuid.UUID, error)
}

// EnsureDefaultOrganization returns the id of the default Lead Cat organization,
// creating it on first call. ownerID may be uuid.Nil — the persistence layer
// translates that to a NULL FK (organizations.owner_user_id is nullable).
func (s *Services) EnsureDefaultOrganization(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, error) {
	return ensureDefaultOrganization(ctx, s.Store, ownerID)
}

func ensureDefaultOrganization(ctx context.Context, store organizationEnsurer, ownerID uuid.UUID) (uuid.UUID, error) {
	return store.EnsureDefaultOrganizationID(ctx, defaultOrganizationTZ, defaultOrganizationMeetLink, ownerID)
}
