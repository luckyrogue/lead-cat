package botreg

import (
	"context"
	"net/mail"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

type State struct {
	Step     string `json:"step"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

const (
	stepName  = "awaiting_name"
	stepEmail = "awaiting_email"
)

type userStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (model.BotUser, error)
	GetBotUserByEmail(ctx context.Context, email string) (model.BotUser, error)
	CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (model.BotUser, error)
}

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	users    userStore
	sessions sessions
	admins   map[int64]bool
}

func New(users userStore, sess sessions, adminIDs []int64) *Service {
	admins := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = true
	}
	return &Service{users: users, sessions: sess, admins: admins}
}

func (s *Service) Start(ctx context.Context, telegramID int64, lang string) string {
	if _, err := s.users.GetBotUserByTelegramID(ctx, telegramID); err == nil {
		return boti18n.T(lang, "botreg.welcome_back")
	}
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepName})
	return boti18n.T(lang, "botreg.start")
}

func (s *Service) finishRegistration(ctx context.Context, telegramID int64, st State, lang string) (string, bool) {
	role := "user"
	if s.admins[telegramID] {
		role = "admin"
	}
	if _, err := s.users.CreateBotUser(ctx, telegramID, st.FullName, st.Email, role); err != nil {
		return boti18n.T(lang, "botreg.failed"), true
	}
	_ = s.sessions.Del(ctx, telegramID)
	return boti18n.T(lang, "botreg.done", st.FullName), true
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text, lang string) (string, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return "", false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepName:
		if text == "" {
			return boti18n.T(lang, "botreg.ask_name"), true
		}
		st.FullName = text
		st.Step = stepEmail
		_ = s.sessions.Set(ctx, telegramID, *st)
		return boti18n.T(lang, "botreg.ask_email"), true

	case stepEmail:
		addr, perr := mail.ParseAddress(text)
		if perr != nil {
			return boti18n.T(lang, "botreg.bad_email"), true
		}
		email := strings.ToLower(strings.TrimSpace(addr.Address))
		if _, gerr := s.users.GetBotUserByEmail(ctx, email); gerr == nil {
			return boti18n.T(lang, "botreg.email_taken"), true
		}
		st.Email = email
		return s.finishRegistration(ctx, telegramID, *st, lang)
	}
	return "", false
}
