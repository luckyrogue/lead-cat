package application

import (
	"context"

	"github.com/google/uuid"
)

const (
	defaultWorkspaceTZ       = "Asia/Almaty"
	defaultWorkspaceMeetLink = ""
)

// workspaceEnsurer is the narrow store interface used by EnsureSingleWorkspace
// — defined here so unit tests can mock it.
type workspaceEnsurer interface {
	EnsureLeadCatWorkspaceID(ctx context.Context, tz, meetLink string, ownerID uuid.UUID) (uuid.UUID, error)
}

// EnsureSingleWorkspace returns the id of the singleton Lead Cat workspace,
// creating it on first call. ownerID may be uuid.Nil — the persistence layer
// translates that to a NULL FK (workspaces.owner_user_id is nullable).
func (s *Services) EnsureSingleWorkspace(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, error) {
	return ensureSingleWorkspace(ctx, s.Store, ownerID)
}

func ensureSingleWorkspace(ctx context.Context, store workspaceEnsurer, ownerID uuid.UUID) (uuid.UUID, error) {
	return store.EnsureLeadCatWorkspaceID(ctx, defaultWorkspaceTZ, defaultWorkspaceMeetLink, ownerID)
}
