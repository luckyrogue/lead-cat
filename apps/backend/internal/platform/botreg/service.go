package botreg

import (
	"context"
	"net/mail"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
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
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (postgres.BotUser, error)
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

func (s *Service) Start(ctx context.Context, telegramID int64) string {
	if _, err := s.users.GetBotUserByTelegramID(ctx, telegramID); err == nil {
		return "С возвращением! 🐾 Открой приложение из меню."
	}
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepName})
	return "Привет! Давай зарегистрируемся.\nВведи ФИО (Фамилия Имя Отчество):"
}

func (s *Service) finishRegistration(ctx context.Context, telegramID int64, st State) (string, bool) {
	role := "user"
	if s.admins[telegramID] {
		role = "admin"
	}
	if _, err := s.users.CreateBotUser(ctx, telegramID, st.FullName, st.Email, role); err != nil {
		return "Не удалось завершить регистрацию, попробуй позже.", true
	}
	_ = s.sessions.Del(ctx, telegramID)
	return "Готово, " + st.FullName + "! 🐾", true
}

func (s *Service) OnText(ctx context.Context, telegramID int64, text string) (string, bool) {
	st, err := s.sessions.Get(ctx, telegramID)
	if err != nil || st == nil {
		return "", false
	}
	text = strings.TrimSpace(text)
	switch st.Step {
	case stepName:
		if text == "" {
			return "Введи ФИО:", true
		}
		st.FullName = text
		st.Step = stepEmail
		_ = s.sessions.Set(ctx, telegramID, *st)
		return "Теперь корпоративную почту:", true

	case stepEmail:
		addr, perr := mail.ParseAddress(text)
		if perr != nil {
			return "Не похоже на email. Попробуй ещё раз:", true
		}
		email := strings.ToLower(strings.TrimSpace(addr.Address))
		if _, gerr := s.users.GetBotUserByEmail(ctx, email); gerr == nil {
			return "Эта почта уже привязана к другому аккаунту.", true
		}
		st.Email = email
		return s.finishRegistration(ctx, telegramID, *st)
	}
	return "", false
}
