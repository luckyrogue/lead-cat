package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type orgMemberStore interface {
	ListMembers(ctx context.Context, organizationID uuid.UUID) ([]model.Member, error)
	ListInvites(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationInvite, error)
	ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]model.Member, error)
}

type OrgMembers struct {
	Store orgMemberStore
}

func NewOrgMembers(store orgMemberStore) *OrgMembers {
	return &OrgMembers{Store: store}
}

func (q *OrgMembers) ListMembers(ctx context.Context, organizationID uuid.UUID) ([]model.Member, error) {
	return q.Store.ListMembers(ctx, organizationID)
}

func (q *OrgMembers) ListOrgInvites(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationInvite, error) {
	return q.Store.ListInvites(ctx, orgID)
}

func (q *OrgMembers) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]model.Member, error) {
	return q.Store.ListOrgMembers(ctx, orgID)
}
