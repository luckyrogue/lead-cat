package application

import (
	"context"

	"github.com/google/uuid"

	platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
)

// EnsureTMAOrganizer find-or-creates the platform_users row backing a TMA user
// (by email, via the email:<email> auth_sub convention) and links the telegram id,
// returning the platform_users UUID used as a meeting organizer. Idempotent; it
// unifies with native email-OTP login (same auth_sub), so a meeting organized on
// the web is editable from the Mini App and vice-versa.
//
// May return ErrTelegramLinkedToOtherAccount if the telegram id is already bound
// to a different platform_users row; callers should map that to 409 Conflict.
func (s *Services) EnsureTMAOrganizer(ctx context.Context, email string, telegramID int64) (uuid.UUID, error) {
	u, err := s.Store.UpsertUserIdentity(ctx, platformauth.SubEmail(email), email, "")
	if err != nil {
		return uuid.Nil, err
	}
	if err := s.LinkTelegram(ctx, u.ID, telegramID, ""); err != nil {
		return uuid.Nil, err
	}
	return u.ID, nil
}
