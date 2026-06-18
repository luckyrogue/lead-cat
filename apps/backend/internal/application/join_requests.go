package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type JoinResult struct {
	AlreadyMember  bool      `json:"already_member"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

func (s *Services) RequestToJoinBySlug(ctx context.Context, userID uuid.UUID, slug string) (JoinResult, error) {
	org, err := s.Store.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return JoinResult{}, err
	}
	if _, ok, err := s.Store.GetOrgMember(ctx, org.ID, userID); err != nil {
		return JoinResult{}, err
	} else if ok {
		return JoinResult{AlreadyMember: true, OrganizationID: org.ID}, nil
	}
	if err := s.Store.CreateJoinRequest(ctx, org.ID, userID); err != nil {
		return JoinResult{}, err
	}
	return JoinResult{OrganizationID: org.ID}, nil
}

func (s *Services) ListMyJoinRequests(ctx context.Context, userID uuid.UUID) ([]model.JoinRequestView, error) {
	return s.Store.ListJoinRequestsForUser(ctx, userID)
}

func (s *Services) ListOrgJoinRequests(ctx context.Context, orgID uuid.UUID) ([]model.JoinRequestAdminView, error) {
	return s.Store.ListPendingJoinRequests(ctx, orgID)
}

func (s *Services) AcceptJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return s.Store.AcceptJoinRequest(ctx, orgID, requestID, deciderID)
}

func (s *Services) DeclineJoinRequest(ctx context.Context, orgID, requestID, deciderID uuid.UUID) error {
	return s.Store.DeclineJoinRequest(ctx, orgID, requestID, deciderID)
}
