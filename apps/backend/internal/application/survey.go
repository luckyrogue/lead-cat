package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Services) requireOrgMember(ctx context.Context, orgID, userID uuid.UUID) error {
	if _, ok, err := s.Store.GetOrgMember(ctx, orgID, userID); err != nil {
		return err
	} else if !ok {
		return model.ErrForbidden
	}
	return nil
}

func (s *Services) CreateSurvey(ctx context.Context, orgID, userID uuid.UUID, in model.Survey) (model.Survey, error) {
	if err := s.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, err
	}
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

func (s *Services) GetSurvey(ctx context.Context, orgID, userID, id uuid.UUID) (model.Survey, error) {
	if err := s.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, err
	}
	return s.requireSurveyOrg(ctx, orgID, id)
}

func (s *Services) ListSurveys(ctx context.Context, orgID, userID uuid.UUID) ([]model.Survey, error) {
	if err := s.requireOrgMember(ctx, orgID, userID); err != nil {
		return nil, err
	}
	return s.Store.ListSurveys(ctx, orgID)
}

func (s *Services) UpdateSurvey(ctx context.Context, orgID, userID, id uuid.UUID, in model.Survey) (model.Survey, error) {
	if err := s.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, err
	}
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

func (s *Services) DeleteSurvey(ctx context.Context, orgID, userID, id uuid.UUID) error {
	if err := s.requireOrgMember(ctx, orgID, userID); err != nil {
		return err
	}
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

func (s *Services) ListResponses(ctx context.Context, orgID, userID, surveyID uuid.UUID, f model.ResponseFilter) (model.Survey, []model.SurveyResponse, error) {
	if err := s.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, nil, err
	}
	sv, err := s.requireSurveyOrg(ctx, orgID, surveyID)
	if err != nil {
		return model.Survey{}, nil, err
	}
	rs, err := s.Store.ListSurveyResponses(ctx, surveyID, f)
	if err != nil {
		return model.Survey{}, nil, err
	}
	return sv, rs, nil
}
