package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type surveyStore interface {
	GetOrgMember(ctx context.Context, orgID, userID uuid.UUID) (model.Member, bool, error)
	CreateSurvey(ctx context.Context, s model.Survey) (model.Survey, error)
	GetSurvey(ctx context.Context, id uuid.UUID) (model.Survey, error)
	UpdateSurvey(ctx context.Context, s model.Survey) error
	DeleteSurvey(ctx context.Context, id uuid.UUID) error
	CountResponses(ctx context.Context, surveyID uuid.UUID) (int, error)
	GetSurveyResponseByToken(ctx context.Context, token string) (model.SurveyResponse, error)
	CompleteSurveyResponse(ctx context.Context, id uuid.UUID, answers []model.Answer) error
}

type Surveys struct {
	Store surveyStore
}

func (c *Surveys) requireOrgMember(ctx context.Context, orgID, userID uuid.UUID) error {
	if _, ok, err := c.Store.GetOrgMember(ctx, orgID, userID); err != nil {
		return err
	} else if !ok {
		return model.ErrForbidden
	}
	return nil
}

func (c *Surveys) requireSurveyOrg(ctx context.Context, orgID, id uuid.UUID) (model.Survey, error) {
	sv, err := c.Store.GetSurvey(ctx, id)
	if err != nil {
		return model.Survey{}, err
	}
	if sv.OrganizationID != orgID {
		return model.Survey{}, model.ErrForbidden
	}
	return sv, nil
}

func (c *Surveys) CreateSurvey(ctx context.Context, orgID, userID uuid.UUID, in model.Survey) (model.Survey, error) {
	if err := c.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, err
	}
	in.OrganizationID = orgID
	if err := in.Validate(); err != nil {
		return model.Survey{}, err
	}
	return c.Store.CreateSurvey(ctx, in)
}

func (c *Surveys) UpdateSurvey(ctx context.Context, orgID, userID, id uuid.UUID, in model.Survey) (model.Survey, error) {
	if err := c.requireOrgMember(ctx, orgID, userID); err != nil {
		return model.Survey{}, err
	}
	if _, err := c.requireSurveyOrg(ctx, orgID, id); err != nil {
		return model.Survey{}, err
	}
	in.ID = id
	in.OrganizationID = orgID
	if err := in.Validate(); err != nil {
		return model.Survey{}, err
	}
	if err := c.Store.UpdateSurvey(ctx, in); err != nil {
		return model.Survey{}, fmt.Errorf("update survey: %w", err)
	}
	return c.Store.GetSurvey(ctx, id)
}

func (c *Surveys) DeleteSurvey(ctx context.Context, orgID, userID, id uuid.UUID) error {
	if err := c.requireOrgMember(ctx, orgID, userID); err != nil {
		return err
	}
	if _, err := c.requireSurveyOrg(ctx, orgID, id); err != nil {
		return err
	}
	n, err := c.Store.CountResponses(ctx, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return model.ErrSurveyHasResponses
	}
	return c.Store.DeleteSurvey(ctx, id)
}

func (c *Surveys) SubmitSurvey(ctx context.Context, token string, answers []model.Answer) error {
	resp, err := c.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return err
	}
	if resp.Status == "completed" {
		return model.ErrResponseCompleted
	}
	sv, err := c.Store.GetSurvey(ctx, resp.SurveyID)
	if err != nil {
		return err
	}
	if !sv.IsActive {
		return model.ErrSurveyClosed
	}
	normalized, err := model.ValidateAnswers(sv.Questions, answers)
	if err != nil {
		return err
	}
	return c.Store.CompleteSurveyResponse(ctx, resp.ID, normalized)
}
