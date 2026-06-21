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
	// isMember controls whether GetOrgMember reports membership (default true).
	isMember bool
}

func (f *surveyFakeStore) GetOrgMember(_ context.Context, _, _ uuid.UUID) (model.Member, bool, error) {
	return model.Member{}, f.isMember, nil
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

// memberStore is a surveyFakeStore that reports the caller as a member.
func memberStore(base *surveyFakeStore) *surveyFakeStore {
	base.isMember = true
	return base
}

func TestCreateSurveyValidates(t *testing.T) {
	svc := newSurveySvc(memberStore(&surveyFakeStore{}))
	userID := uuid.New()
	_, err := svc.CreateSurvey(context.Background(), uuid.New(), userID, model.Survey{Name: ""})
	if !errors.Is(err, model.ErrInvalidSurvey) {
		t.Fatalf("expected ErrInvalidSurvey, got %v", err)
	}
}

func TestDeleteSurveyBlockedWhenResponsesExist(t *testing.T) {
	org := uuid.New()
	id := uuid.New()
	store := memberStore(&surveyFakeStore{survey: model.Survey{ID: id, OrganizationID: org}, respCount: 3})
	svc := newSurveySvc(store)
	err := svc.DeleteSurvey(context.Background(), org, uuid.New(), id)
	if !errors.Is(err, model.ErrSurveyHasResponses) {
		t.Fatalf("expected ErrSurveyHasResponses, got %v", err)
	}
	if store.deleted {
		t.Fatal("survey must not be deleted when responses exist")
	}
}

func TestSurveyOrgScoping(t *testing.T) {
	store := memberStore(&surveyFakeStore{survey: model.Survey{ID: uuid.New(), OrganizationID: uuid.New()}})
	svc := newSurveySvc(store)
	_, err := svc.GetSurvey(context.Background(), uuid.New() /* different org */, uuid.New(), store.survey.ID)
	if !errors.Is(err, model.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// TestSurveyRejectsNonMember verifies that all admin survey methods return
// ErrForbidden when the caller is not a member of the org, regardless of
// whether they supply a valid org ID in the header.
func TestSurveyRejectsNonMember(t *testing.T) {
	org := uuid.New()
	surveyID := uuid.New()
	nonMemberStore := &surveyFakeStore{
		isMember: false,
		survey:   model.Survey{ID: surveyID, OrganizationID: org},
	}
	svc := newSurveySvc(nonMemberStore)
	userID := uuid.New() // caller who is NOT a member

	t.Run("CreateSurvey", func(t *testing.T) {
		_, err := svc.CreateSurvey(context.Background(), org, userID, model.Survey{Name: "x"})
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("ListSurveys", func(t *testing.T) {
		_, err := svc.ListSurveys(context.Background(), org, userID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("GetSurvey", func(t *testing.T) {
		_, err := svc.GetSurvey(context.Background(), org, userID, surveyID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("UpdateSurvey", func(t *testing.T) {
		_, err := svc.UpdateSurvey(context.Background(), org, userID, surveyID, model.Survey{Name: "y"})
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("DeleteSurvey", func(t *testing.T) {
		err := svc.DeleteSurvey(context.Background(), org, userID, surveyID)
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})

	t.Run("ListResponses", func(t *testing.T) {
		_, _, err := svc.ListResponses(context.Background(), org, userID, surveyID, model.ResponseFilter{})
		if !errors.Is(err, model.ErrForbidden) {
			t.Fatalf("expected ErrForbidden, got %v", err)
		}
	})
}
