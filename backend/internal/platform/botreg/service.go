// Package botreg drives the Telegram /start registration FSM: ФИО -> email ->
// OTP -> create bot_users. It depends on small interfaces so the flow is
// testable without Redis, Postgres, or Telegram.
package botreg

import (
	"context"
	"net/mail"
	"strings"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// State is the per-user FSM state (stored in Redis between messages).
type State struct {
	Step     string `json:"step"` // awaiting_name | awaiting_email | awaiting_otp
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

const (
	stepName  = "awaiting_name"
	stepEmail = "awaiting_email"
	stepOTP   = "awaiting_otp"
)

type userStore interface {
	GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error)
	GetBotUserByEmail(ctx context.Context, email string) (postgres.BotUser, error)
	CreateBotUser(ctx context.Context, telegramID int64, fullName, email, role string) (postgres.BotUser, error)
}

type otpSender interface {
	Send(ctx context.Context, channel, dest string) (string, error)
	Verify(ctx context.Context, channel, dest, code string) (bool, error)
}

type sessions interface {
	Get(ctx context.Context, telegramID int64) (*State, error)
	Set(ctx context.Context, telegramID int64, s State) error
	Del(ctx context.Context, telegramID int64) error
}

type Service struct {
	users    userStore
	otp      otpSender
	sessions sessions
	admins   map[int64]bool
}

func New(users userStore, otp otpSender, sess sessions, adminIDs []int64) *Service {
	admins := make(map[int64]bool, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = true
	}
	return &Service{users: users, otp: otp, sessions: sess, admins: admins}
}

// Start handles /start: returns a welcome for registered users, otherwise opens
// the registration flow and prompts for the full name.
func (s *Service) Start(ctx context.Context, telegramID int64) string {
	if _, err := s.users.GetBotUserByTelegramID(ctx, telegramID); err == nil {
		return "С возвращением! 🐾 Открой приложение из меню."
	}
	_ = s.sessions.Set(ctx, telegramID, State{Step: stepName})
	return "Привет! Давай зарегистрируемся.\nВведи ФИО (Фамилия Имя Отчество):"
}

// OnText feeds a free-text message into an active registration session. The
// bool is false (and reply empty) when there is no active session.
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
		// Normalize so "Ivan@Corp.kz" and "ivan@corp.kz" can't become two
		// accounts (the bot_users.email UNIQUE constraint is case-sensitive),
		// and strip any display-name form.
		email := strings.ToLower(strings.TrimSpace(addr.Address))
		if _, gerr := s.users.GetBotUserByEmail(ctx, email); gerr == nil {
			return "Эта почта уже привязана к другому аккаунту.", true
		}
		if _, serr := s.otp.Send(ctx, "email", email); serr != nil {
			return "Не смог отправить код, попробуй позже.", true
		}
		st.Email = email
		st.Step = stepOTP
		_ = s.sessions.Set(ctx, telegramID, *st)
		return "Отправил код на почту. Введи его:", true

	case stepOTP:
		ok, _ := s.otp.Verify(ctx, "email", st.Email, text)
		if !ok {
			return "Неверный код. Попробуй ещё раз:", true
		}
		role := "user"
		if s.admins[telegramID] {
			role = "admin"
		}
		if _, cerr := s.users.CreateBotUser(ctx, telegramID, st.FullName, st.Email, role); cerr != nil {
			return "Не удалось завершить регистрацию, попробуй позже.", true
		}
		_ = s.sessions.Del(ctx, telegramID)
		return "Готово, " + st.FullName + "! 🐾", true
	}
	return "", false
}
