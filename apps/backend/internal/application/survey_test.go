package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type surveyFakeStore struct {
	Repository
	survey    model.Survey
	created   model.Survey
	respCount int
	deleted   bool
}

func (f *surveyFakeStore) CreateSurvey(_ context.Context, s model.Survey) (model.Survey, error) {
	s.ID = uuid.New()
	f.created = s
	return s, nil
}
func (f *surveyFakeStore) GetSurvey(_ context.Context, _ uuid.UUID) (model.Survey, error) {
	return f.survey, nil
}
func (f *surveyFakeStore) UpdateSurvey(_ context.Context, _ model.Survey) error { return nil }
func (f *surveyFakeStore) CountResponses(_ context.Context, _ uuid.UUID) (int, error) {
	return f.respCount, nil
}
func (f *surveyFakeStore) DeleteSurvey(_ context.Context, _ uuid.UUID) error {
	f.deleted = true
	return nil
}

func newSurveySvc(store Repository) *Services {
	return &Services{Store: store, Log: zap.NewNop()}
}

func TestCreateSurveyValidates(t *testing.T) {
	svc := newSurveySvc(&surveyFakeStore{})
	_, err := svc.CreateSurvey(context.Background(), uuid.New(), model.Survey{Name: ""})
	if !errors.Is(err, model.ErrInvalidSurvey) {
		t.Fatalf("expected ErrInvalidSurvey, got %v", err)
	}
}

func TestDeleteSurveyBlockedWhenResponsesExist(t *testing.T) {
	org := uuid.New()
	id := uuid.New()
	store := &surveyFakeStore{survey: model.Survey{ID: id, OrganizationID: org}, respCount: 3}
	svc := newSurveySvc(store)
	err := svc.DeleteSurvey(context.Background(), org, id)
	if !errors.Is(err, model.ErrSurveyHasResponses) {
		t.Fatalf("expected ErrSurveyHasResponses, got %v", err)
	}
	if store.deleted {
		t.Fatal("survey must not be deleted when responses exist")
	}
}

func TestSurveyOrgScoping(t *testing.T) {
	store := &surveyFakeStore{survey: model.Survey{ID: uuid.New(), OrganizationID: uuid.New()}}
	svc := newSurveySvc(store)
	_, err := svc.GetSurvey(context.Background(), uuid.New() /* different org */, store.survey.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
