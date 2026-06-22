package query

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type surveyStore interface {
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	GetSurvey(ctx context.Context, id uuid.UUID) (model.Survey, error)
	ListSurveys(ctx context.Context, orgID uuid.UUID) ([]model.Survey, error)
	ListSurveyResponses(ctx context.Context, surveyID uuid.UUID, f model.ResponseFilter) ([]model.SurveyResponse, error)
	GetSurveyResponseByToken(ctx context.Context, token string) (model.SurveyResponse, error)
}

type Surveys struct {
	Store surveyStore
}

func NewSurveys(store surveyStore) *Surveys {
	return &Surveys{Store: store}
}

func (q *Surveys) requireOrgMember(ctx context.Context, orgID, userID uuid.UUID) error {
	if _, ok, err := q.Store.GetOrgMember(ctx, orgID, userID); err != nil {
		return err
	} else if !ok {
		return model.ErrForbidden
	}
	return nil
}

func (q *Surveys) requireSurveyOrg(ctx context.Context, orgID, id uuid.UUID) (model.Survey, error) {
	sv, err := q.Store.GetSurvey(ctx, id)
	if err != nil {
		return model.Survey{}, err
	}
	if sv.OrganizationID != orgID {
		return model.Survey{}, model.ErrForbidden
	}
	return sv, nil
}

func (q *Surveys) GetSurvey(ctx context.Context, orgID, userID, id uuid.UUID) (model.Survey, error) {
	if err := q.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, err
	}
	return q.requireSurveyOrg(ctx, orgID, id)
}

func (q *Surveys) ListSurveys(ctx context.Context, orgID, userID uuid.UUID) ([]model.Survey, error) {
	if err := q.requireOrgMember(ctx, orgID, userID); err != nil {
		return nil, err
	}
	return q.Store.ListSurveys(ctx, orgID)
}

func (q *Surveys) ListResponses(ctx context.Context, orgID, userID, surveyID uuid.UUID, f model.ResponseFilter) (model.Survey, []model.SurveyResponse, error) {
	if err := q.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, nil, err
	}
	sv, err := q.requireSurveyOrg(ctx, orgID, surveyID)
	if err != nil {
		return model.Survey{}, nil, err
	}
	rs, err := q.Store.ListSurveyResponses(ctx, surveyID, f)
	if err != nil {
		return model.Survey{}, nil, err
	}
	return sv, rs, nil
}

func (q *Surveys) GetPublicSurvey(ctx context.Context, token string) (model.SurveyResponse, model.Survey, error) {
	resp, err := q.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return model.SurveyResponse{}, model.Survey{}, err
	}
	if resp.Status == "completed" {
		return resp, model.Survey{}, model.ErrResponseCompleted
	}
	sv, err := q.Store.GetSurvey(ctx, resp.SurveyID)
	if err != nil {
		return model.SurveyResponse{}, model.Survey{}, err
	}
	if !sv.IsActive {
		return resp, sv, model.ErrSurveyClosed
	}
	return resp, sv, nil
}
