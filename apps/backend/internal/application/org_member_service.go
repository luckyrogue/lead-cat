package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

var ErrLastOwner = command.ErrLastOwner

func (s *Services) ListMembers(ctx context.Context, organizationID uuid.UUID) ([]model.Member, error) {
	return s.OrgMemberQueries.ListMembers(ctx, organizationID)
}

func (s *Services) ListOrgInvites(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationInvite, error) {
	return s.OrgMemberQueries.ListOrgInvites(ctx, orgID)
}

func (s *Services) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]model.Member, error) {
	return s.OrgMemberQueries.ListOrgMembers(ctx, orgID)
}

func (s *Services) DeleteOrgInvite(ctx context.Context, orgID, inviteID uuid.UUID) error {
	return s.OrgMemberCommands.DeleteOrgInvite(ctx, orgID, inviteID)
}

func (s *Services) RemoveOrgMember(ctx context.Context, orgID, targetUserID uuid.UUID) error {
	return s.OrgMemberCommands.RemoveOrgMember(ctx, orgID, targetUserID)
}

func (s *Services) SetOrgMemberRole(ctx context.Context, orgID, targetUserID uuid.UUID, newRole string) error {
	return s.OrgMemberCommands.SetOrgMemberRole(ctx, orgID, targetUserID, newRole)
}

func (s *Services) AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (model.Member, error) {
	return s.OrgMemberCommands.AddMember(ctx, organizationID, username, role)
}

func (s *Services) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	return s.OrgMemberCommands.DeleteMember(ctx, memberID)
}
