package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Services) GetPublicSurvey(ctx context.Context, token string) (model.SurveyResponse, model.Survey, error) {
	resp, err := s.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return model.SurveyResponse{}, model.Survey{}, err
	}
	if resp.Status == "completed" {
		return resp, model.Survey{}, model.ErrResponseCompleted
	}
	sv, err := s.Store.GetSurvey(ctx, resp.SurveyID)
	if err != nil {
		return model.SurveyResponse{}, model.Survey{}, err
	}
	if !sv.IsActive {
		return resp, sv, model.ErrSurveyClosed
	}
	return resp, sv, nil
}

func (s *Services) SubmitSurvey(ctx context.Context, token string, answers []model.Answer) error {
	resp, err := s.Store.GetSurveyResponseByToken(ctx, token)
	if err != nil {
		return err
	}
	if resp.Status == "completed" {
		return model.ErrResponseCompleted
	}
	sv, err := s.Store.GetSurvey(ctx, resp.SurveyID)
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
	return s.Store.CompleteSurveyResponse(ctx, resp.ID, normalized)
}

func (s *Services) CreatePendingResponse(ctx context.Context, et model.BookingEventType, reason string, req BookingRequest) (string, error) {
	if et.SurveyID == nil {
		return "", nil
	}
	sv, err := s.Store.GetSurvey(ctx, *et.SurveyID)
	if err != nil {
		return "", err
	}
	if !sv.IsActive {
		return "", nil
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	etID := et.ID
	_, err = s.Store.CreateSurveyResponse(ctx, model.SurveyResponse{
		SurveyID:           sv.ID,
		OrganizationID:     et.OrganizationID,
		BookingEventTypeID: &etID,
		Token:              token,
		BookerEmail:        req.Email,
		BookerName:         req.Name,
		DeclineReason:      reason,
	})
	if err != nil {
		return "", err
	}
	return token, nil
}
