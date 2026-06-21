package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Services) CreateSurvey(ctx context.Context, orgID uuid.UUID, in model.Survey) (model.Survey, error) {
	in.OrganizationID = orgID
	if err := in.Validate(); err != nil {
		return model.Survey{}, err
	}
	return s.Store.CreateSurvey(ctx, in)
}

func (s *Services) requireSurveyOrg(ctx context.Context, orgID, id uuid.UUID) (model.Survey, error) {
	sv, err := s.Store.GetSurvey(ctx, id)
	if err != nil {
		return model.Survey{}, err
	}
	if sv.OrganizationID != orgID {
		return model.Survey{}, model.ErrForbidden
	}
	return sv, nil
}

func (s *Services) GetSurvey(ctx context.Context, orgID, id uuid.UUID) (model.Survey, error) {
	return s.requireSurveyOrg(ctx, orgID, id)
}

func (s *Services) ListSurveys(ctx context.Context, orgID uuid.UUID) ([]model.Survey, error) {
	return s.Store.ListSurveys(ctx, orgID)
}

func (s *Services) UpdateSurvey(ctx context.Context, orgID, id uuid.UUID, in model.Survey) (model.Survey, error) {
	if _, err := s.requireSurveyOrg(ctx, orgID, id); err != nil {
		return model.Survey{}, err
	}
	in.ID = id
	in.OrganizationID = orgID
	if err := in.Validate(); err != nil {
		return model.Survey{}, err
	}
	if err := s.Store.UpdateSurvey(ctx, in); err != nil {
		return model.Survey{}, fmt.Errorf("update survey: %w", err)
	}
	return s.Store.GetSurvey(ctx, id)
}

func (s *Services) DeleteSurvey(ctx context.Context, orgID, id uuid.UUID) error {
	if _, err := s.requireSurveyOrg(ctx, orgID, id); err != nil {
		return err
	}
	n, err := s.Store.CountResponses(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return model.ErrSurveyHasResponses
	}
	return s.Store.DeleteSurvey(ctx, id)
}
