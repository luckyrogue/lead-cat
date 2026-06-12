package application

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type filterFakeRepo struct {
	unimplementedRepo
	owner     uuid.UUID
	gotFilter model.MeetingFilter
}

func (r *filterFakeRepo) GetOrganization(context.Context, uuid.UUID) (model.Organization, error) {
	owner := r.owner
	return model.Organization{OwnerUserID: &owner}, nil
}

func (r *filterFakeRepo) ListMeetingsFiltered(_ context.Context, _ uuid.UUID, f model.MeetingFilter) ([]model.Meeting, error) {
	r.gotFilter = f
	return nil, nil
}

func TestListMeetingsFiltered_NonOwnerForcedToSelf(t *testing.T) {
	userID := uuid.New()
	requested := uuid.New()
	repo := &filterFakeRepo{owner: uuid.New()} // owner != userID
	s := &Services{Store: repo}

	_, err := s.ListMeetingsFiltered(context.Background(), uuid.New(), userID, model.MeetingFilter{Organizer: &requested})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotFilter.Organizer == nil || *repo.gotFilter.Organizer != userID {
		t.Fatalf("non-owner organizer = %v, want %v", repo.gotFilter.Organizer, userID)
	}
}

func TestListMeetingsFiltered_OwnerKeepsRequestedOrganizer(t *testing.T) {
	ownerID := uuid.New()
	requested := uuid.New()
	repo := &filterFakeRepo{owner: ownerID}
	s := &Services{Store: repo}

	_, err := s.ListMeetingsFiltered(context.Background(), uuid.New(), ownerID, model.MeetingFilter{Organizer: &requested})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.gotFilter.Organizer == nil || *repo.gotFilter.Organizer != requested {
		t.Fatalf("owner organizer = %v, want %v", repo.gotFilter.Organizer, requested)
	}
}
