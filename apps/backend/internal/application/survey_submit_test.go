package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type submitSurveyFakeStore struct {
	Repository
	resp        model.SurveyResponse
	respErr     error
	survey      model.Survey
	completedID uuid.UUID
	created     model.SurveyResponse
}

func (f *submitSurveyFakeStore) GetSurveyResponseByToken(_ context.Context, _ string) (model.SurveyResponse, error) {
	return f.resp, f.respErr
}
func (f *submitSurveyFakeStore) GetSurvey(_ context.Context, _ uuid.UUID) (model.Survey, error) {
	return f.survey, nil
}
func (f *submitSurveyFakeStore) CompleteSurveyResponse(_ context.Context, id uuid.UUID, _ []model.Answer) error {
	f.completedID = id
	return nil
}
func (f *submitSurveyFakeStore) CreateSurveyResponse(_ context.Context, r model.SurveyResponse) (model.SurveyResponse, error) {
	r.ID = uuid.New()
	f.created = r
	return r, nil
}

func TestSubmitSurveyRejectsCompleted(t *testing.T) {
	store := &submitSurveyFakeStore{resp: model.SurveyResponse{ID: uuid.New(), Status: "completed"}}
	svc := newSurveySvc(store)
	err := svc.SubmitSurvey(context.Background(), "tok", nil)
	if !errors.Is(err, model.ErrResponseCompleted) {
		t.Fatalf("expected ErrResponseCompleted, got %v", err)
	}
}

func TestSubmitSurveyValidatesAndCompletes(t *testing.T) {
	q := model.SurveyQuestion{ID: uuid.New(), Prompt: "Why?", Type: model.QuestionText, Required: true}
	respID := uuid.New()
	store := &submitSurveyFakeStore{
		resp:   model.SurveyResponse{ID: respID, Status: "sent", SurveyID: uuid.New()},
		survey: model.Survey{IsActive: true, Questions: []model.SurveyQuestion{q}},
	}
	svc := newSurveySvc(store)
	if err := svc.SubmitSurvey(context.Background(), "tok", []model.Answer{{QuestionID: q.ID, Value: "x"}}); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if store.completedID != respID {
		t.Fatalf("expected completion of %v, got %v", respID, store.completedID)
	}
}

func TestCreatePendingResponseNoSurvey(t *testing.T) {
	svc := newSurveySvc(&submitSurveyFakeStore{})
	tok, err := svc.CreatePendingResponse(context.Background(), model.BookingEventType{}, "slot_taken", BookingRequest{})
	if err != nil || tok != "" {
		t.Fatalf("expected empty token + no error, got %q %v", tok, err)
	}
}

func TestCreatePendingResponseActiveSurvey(t *testing.T) {
	sid := uuid.New()
	store := &submitSurveyFakeStore{survey: model.Survey{ID: sid, IsActive: true}}
	svc := newSurveySvc(store)
	et := model.BookingEventType{OrganizationID: uuid.New(), SurveyID: &sid}
	tok, err := svc.CreatePendingResponse(context.Background(), et, "slot_taken", BookingRequest{Email: "a@b.c", Name: "Bo"})
	if err != nil || tok == "" {
		t.Fatalf("expected a token, got %q %v", tok, err)
	}
	if store.created.BookerEmail != "a@b.c" || store.created.DeclineReason != "slot_taken" {
		t.Fatalf("unexpected created response: %+v", store.created)
	}
}
