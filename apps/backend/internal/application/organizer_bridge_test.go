package application

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
)

type identityMergeRepo struct {
	unimplementedRepo
	bySub map[string]model.User
	tg    map[int64]uuid.UUID
}

func (s *identityMergeRepo) UpsertUserIdentity(_ context.Context, authSub, email string) (model.User, error) {
	if u, ok := s.bySub[authSub]; ok {
		return u, nil
	}
	u := model.User{ID: uuid.New(), AuthSub: authSub, Email: email}
	s.bySub[authSub] = u
	return u, nil
}

func (s *identityMergeRepo) UpsertWebIdentity(_ context.Context, email, name, avatarURL, authMethod string) (model.PlatformUser, error) {
	sub := platformauth.SubEmail(email)
	if u, ok := s.bySub[sub]; ok {
		return model.PlatformUser{
			ID: u.ID, AuthSub: sub, Email: email,
			AvatarURL: avatarURL, AuthMethod: authMethod,
		}, nil
	}
	u := model.User{ID: uuid.New(), AuthSub: sub, Email: email}
	s.bySub[sub] = u
	return model.PlatformUser{
		ID: u.ID, AuthSub: sub, Email: email,
		AvatarURL: avatarURL, AuthMethod: authMethod,
	}, nil
}

func (s *identityMergeRepo) GetUserTelegramID(_ context.Context, userID uuid.UUID) (int64, bool, error) {
	for tg, id := range s.tg {
		if id == userID {
			return tg, true, nil
		}
	}
	return 0, false, nil
}

func (s *identityMergeRepo) GetPlatformUserIDByTelegramID(_ context.Context, telegramID int64) (uuid.UUID, bool, error) {
	id, ok := s.tg[telegramID]
	return id, ok, nil
}

func (s *identityMergeRepo) LinkTelegram(_ context.Context, userID uuid.UUID, telegramID int64) error {
	s.tg[telegramID] = userID
	return nil
}

func (s *identityMergeRepo) LinkMemberUserIDsByTelegram(context.Context, uuid.UUID, string) error {
	return nil
}

func TestEnsureMiniAppOrganizerMergesWithWebIdentity(t *testing.T) {
	t.Parallel()
	store := &identityMergeRepo{bySub: map[string]model.User{}, tg: map[int64]uuid.UUID{}}
	svc := &Services{Store: store}

	tmaID, err := svc.EnsureMiniAppOrganizer(context.Background(), "alice@corp.io", 42)
	if err != nil {
		t.Fatal(err)
	}
	web, err := svc.UpsertWebIdentity(context.Background(), "alice@corp.io", "Alice", "https://x/a.png", "google")
	if err != nil {
		t.Fatal(err)
	}
	if tmaID != web.ID {
		t.Fatalf("expected one platform_users row, got tma=%s web=%s", tmaID, web.ID)
	}
	if web.AuthSub != platformauth.SubEmail("alice@corp.io") {
		t.Fatalf("auth_sub %q", web.AuthSub)
	}
}
