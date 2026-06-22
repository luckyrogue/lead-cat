package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Services) CreateSurvey(ctx context.Context, orgID, userID uuid.UUID, in model.Survey) (model.Survey, error) {
	return s.SurveyCommands.CreateSurvey(ctx, orgID, userID, in)
}

func (s *Services) UpdateSurvey(ctx context.Context, orgID, userID, id uuid.UUID, in model.Survey) (model.Survey, error) {
	return s.SurveyCommands.UpdateSurvey(ctx, orgID, userID, id, in)
}

func (s *Services) DeleteSurvey(ctx context.Context, orgID, userID, id uuid.UUID) error {
	return s.SurveyCommands.DeleteSurvey(ctx, orgID, userID, id)
}

func (s *Services) SubmitSurvey(ctx context.Context, token string, answers []model.Answer) error {
	return s.SurveyCommands.SubmitSurvey(ctx, token, answers)
}

func (s *Services) GetSurvey(ctx context.Context, orgID, userID, id uuid.UUID) (model.Survey, error) {
	return s.SurveyQueries.GetSurvey(ctx, orgID, userID, id)
}

func (s *Services) ListSurveys(ctx context.Context, orgID, userID uuid.UUID) ([]model.Survey, error) {
	return s.SurveyQueries.ListSurveys(ctx, orgID, userID)
}

func (s *Services) ListResponses(ctx context.Context, orgID, userID, surveyID uuid.UUID, f model.ResponseFilter) (model.Survey, []model.SurveyResponse, error) {
	return s.SurveyQueries.ListResponses(ctx, orgID, userID, surveyID, f)
}

func (s *Services) GetPublicSurvey(ctx context.Context, token string) (model.SurveyResponse, model.Survey, error) {
	return s.SurveyQueries.GetPublicSurvey(ctx, token)
}
