package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type membershipStore interface {
	ListPendingInvitesForEmail(ctx context.Context, email string) ([]model.InviteView, error)
	ListJoinRequestsForUser(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error)
	ListPendingJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error)
}

type Membership struct {
	Store membershipStore
}

func NewMembership(store membershipStore) *Membership {
	return &Membership{Store: store}
}

func (q *Membership) ListMyInvites(ctx context.Context, email string) ([]model.InviteView, error) {
	return q.Store.ListPendingInvitesForEmail(ctx, email)
}

func (q *Membership) ListMyJoinRequests(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	return q.Store.ListJoinRequestsForUser(ctx, userID)
}

func (q *Membership) ListOrgJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	return q.Store.ListPendingJoinRequests(ctx, orgID)
}
