package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type JoinResult = command.JoinResult

func (s *Services) ListMyInvites(ctx context.Context, email string) ([]model.InviteView, error) {
	return s.MembershipQueries.ListMyInvites(ctx, email)
}

func (s *Services) AcceptInvite(ctx context.Context, inviteID, userID uuid.UUID, email string) error {
	return s.MembershipCommands.AcceptInvite(ctx, inviteID, userID, email)
}

func (s *Services) DeclineInvite(ctx context.Context, inviteID uuid.UUID, email string) error {
	return s.MembershipCommands.DeclineInvite(ctx, inviteID, email)
}

func (s *Services) RequestToJoinBySlug(ctx context.Context, userID uuid.UUID, slug string) (JoinResult, error) {
	return s.MembershipCommands.RequestToJoinBySlug(ctx, userID, slug)
}

func (s *Services) ListMyJoinRequests(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	return s.MembershipQueries.ListMyJoinRequests(ctx, userID)
}

func (s *Services) ListOrgJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	return s.MembershipQueries.ListOrgJoinRequests(ctx, orgID)
}

func (s *Services) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return s.MembershipCommands.AcceptJoinRequest(ctx, orgID, requestID, deciderID)
}

func (s *Services) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return s.MembershipCommands.DeclineJoinRequest(ctx, orgID, requestID, deciderID)
}
