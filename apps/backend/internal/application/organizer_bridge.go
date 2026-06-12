package application

import (
	"context"

	"github.com/google/uuid"

	platformauth "github.com/luckyrogue/lead-cat/internal/platform/auth"
)

func (s *Services) EnsureMiniAppOrganizer(ctx context.Context, email string, telegramID int64) (uuid.UUID, error) {
	u, err := s.Store.UpsertUserIdentity(ctx, platformauth.SubEmail(email), email)
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.linkTelegram(ctx, u.ID, telegramID, ""); err != nil {
		return uuid.Nil, err
	}
	return u.ID, nil
}
