package command

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var ErrLastOwner = errors.New("cannot remove or demote the last owner")

type orgMemberStore interface {
	DeleteInvite(ctx context.Context, orgID, inviteID uuid.UUID) error
	ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]model.Member, error)
	RemoveMember(ctx context.Context, orgID, targetUserID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, orgID, targetUserID uuid.UUID, newRole string) error
	AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (model.Member, error)
	DeleteMember(ctx context.Context, memberID uuid.UUID) error
}

type OrgMembers struct {
	Store orgMemberStore
}

type orgMemberView struct {
	Role string
}

func memberViews(members []model.Member, target uuid.UUID) ([]orgMemberView, int) {
	views := make([]orgMemberView, len(members))
	idx := -1
	for i, m := range members {
		views[i] = orgMemberView{Role: m.Role}
		if m.UserID != nil && *m.UserID == target {
			idx = i
		}
	}
	return views, idx
}

func canDemoteOrRemove(members []orgMemberView, idx int) error {
	if members[idx].Role != "owner" {
		return nil
	}
	for i, m := range members {
		if i != idx && m.Role == "owner" {
			return nil
		}
	}
	return ErrLastOwner
}

func (c *OrgMembers) DeleteOrgInvite(ctx context.Context, orgID, inviteID uuid.UUID) error {
	return c.Store.DeleteInvite(ctx, orgID, inviteID)
}

func (c *OrgMembers) RemoveOrgMember(ctx context.Context, orgID, targetUserID uuid.UUID) error {
	members, err := c.Store.ListOrgMembers(ctx, orgID)
	if err != nil {
		return err
	}
	views, idx := memberViews(members, targetUserID)
	if idx < 0 {
		return nil
	}
	if err := canDemoteOrRemove(views, idx); err != nil {
		return err
	}
	return c.Store.RemoveMember(ctx, orgID, targetUserID)
}

func (c *OrgMembers) SetOrgMemberRole(ctx context.Context, orgID, targetUserID uuid.UUID, newRole string) error {
	members, err := c.Store.ListOrgMembers(ctx, orgID)
	if err != nil {
		return err
	}
	views, idx := memberViews(members, targetUserID)
	if idx < 0 {
		return nil
	}
	if newRole != "owner" {
		if err := canDemoteOrRemove(views, idx); err != nil {
			return err
		}
	}
	return c.Store.UpdateMemberRole(ctx, orgID, targetUserID, newRole)
}

func (c *OrgMembers) AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (model.Member, error) {
	return c.Store.AddMember(ctx, organizationID, username, role)
}

func (c *OrgMembers) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	return c.Store.DeleteMember(ctx, memberID)
}
