package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Services) ListMyInvites(ctx context.Context, email string) ([]model.InviteView, error) {
	return s.Store.ListPendingInvitesForEmail(ctx, email)
}

func (s *Services) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	return s.Store.AcceptInvite(ctx, inviteID, userID, email)
}

func (s *Services) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	return s.Store.DeclineInvite(ctx, inviteID, email)
}
