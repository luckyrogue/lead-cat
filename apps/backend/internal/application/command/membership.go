package command

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type membershipStore interface {
	AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error
	DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error
	GetOrganizationBySlug(ctx context.Context, slug string) (model.Organization, error)
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	CreateJoinRequest(ctx context.Context, orgID, userID uuid.UUID) error
	AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
	DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error
}

type JoinResult struct {
	AlreadyMember  bool      `json:"already_member"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

type Membership struct {
	Store membershipStore
}

func (c *Membership) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	return c.Store.AcceptInvite(ctx, inviteID, userID, email)
}

func (c *Membership) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	return c.Store.DeclineInvite(ctx, inviteID, email)
}

func (c *Membership) RequestToJoinBySlug(ctx context.Context, userID uuid.UUID, slug string) (JoinResult, error) {
	org, err := c.Store.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return JoinResult{}, err
	}
	if _, ok, err := c.Store.GetOrgMember(ctx, org.ID, userID); err != nil {
		return JoinResult{}, err
	} else if ok {
		return JoinResult{AlreadyMember: true, OrganizationID: org.ID}, nil
	}
	if err := c.Store.CreateJoinRequest(ctx, org.ID, userID); err != nil {
		return JoinResult{}, err
	}
	return JoinResult{OrganizationID: org.ID}, nil
}

func (c *Membership) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return c.Store.AcceptJoinRequest(ctx, orgID, requestID, deciderID)
}

func (c *Membership) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return c.Store.DeclineJoinRequest(ctx, orgID, requestID, deciderID)
}
