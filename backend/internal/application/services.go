package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/query"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
	asynqqueue "github.com/luckyrogue/lead-cat/internal/infrastructure/queue/asynq"
)

// ChatSyncer is a function that syncs chat administrators into workspace_members.
// It is injected at startup to avoid an import cycle between application and
// the telegram infrastructure package.
type ChatSyncer func(ctx context.Context, workspaceID uuid.UUID) (int, error)

type Services struct {
	Store    *postgres.Store
	Cipher   *crypto.TokenCipher
	Queue    *asynqqueue.Client
	Calendar CalendarProvider
	Log      *zap.Logger
	Bot      *bot.Bot
	Queries *query.Meetings
	syncChat ChatSyncer
}

func (s *Services) WireCQRS() {
	if s.Queries == nil {
		s.Queries = query.NewMeetings(s)
	}
}

func (s *Services) Ping(ctx context.Context) error {
	return s.Store.Ping(ctx)
}

func (s *Services) GetBotUserByTelegramID(ctx context.Context, telegramID int64) (postgres.BotUser, error) {
	return s.Store.GetBotUserByTelegramID(ctx, telegramID)
}

func (s *Services) PlatformUserIDForTelegram(ctx context.Context, telegramID int64) (uuid.UUID, bool, error) {
	return s.Store.GetPlatformUserIDByTelegramID(ctx, telegramID)
}

func (s *Services) ListAudit(ctx context.Context, f postgres.AuditFilter) ([]postgres.AuditEntry, error) {
	return s.Store.ListAuditEntries(ctx, f)
}

func (s *Services) ListWorkspacesWithGoogle(ctx context.Context) ([]uuid.UUID, error) {
	return s.Store.ListWorkspacesWithGoogle(ctx)
}

func (s *Services) MiniAppMeetingDTO(ctx context.Context, m postgres.Meeting, loc *time.Location) query.MiniAppMeeting {
	return query.MeetingDTO(ctx, s.Store, m, loc)
}

var ErrTelegramLinkedToOtherAccount = errors.New("telegram already linked to another account")

func (s *Services) linkTelegram(ctx context.Context, userID uuid.UUID, telegramID int64, username string) error {
	if existing, ok, err := s.Store.GetUserTelegramID(ctx, userID); err != nil {
		return err
	} else if ok && existing == telegramID {
		return s.Store.LinkMemberUserIDsByTelegram(ctx, userID, username)
	}
	if ownerID, ok, err := s.Store.GetPlatformUserIDByTelegramID(ctx, telegramID); err != nil {
		return err
	} else if ok && ownerID != userID {
		return ErrTelegramLinkedToOtherAccount
	}
	if err := s.Store.LinkTelegram(ctx, userID, telegramID); err != nil {
		return err
	}
	return s.Store.LinkMemberUserIDsByTelegram(ctx, userID, username)
}

func (s *Services) GetWorkspace(ctx context.Context, id uuid.UUID) (postgres.Workspace, error) {
	return s.Store.GetWorkspace(ctx, id)
}

func (s *Services) LinkChat(ctx context.Context, workspaceID uuid.UUID, chatID int64) error {
	return s.Store.LinkChat(ctx, workspaceID, chatID)
}

// SetChatSyncer injects the ChatSyncer function after bot initialisation.
// Called from main after bot.New to avoid an import cycle.
func (s *Services) SetChatSyncer(fn ChatSyncer) {
	s.syncChat = fn
}

// SyncChatMembers imports current Telegram chat administrators into the
// workspace_members table. Thin wrapper around the telegram helper so HTTP
// handlers don't reach into infrastructure.
func (s *Services) SyncChatMembers(ctx context.Context, workspaceID uuid.UUID) (int, error) {
	if s.syncChat == nil {
		return 0, fmt.Errorf("bot not configured")
	}
	return s.syncChat(ctx, workspaceID)
}

type IntegrationsView struct {
	MeetLink         string `json:"meet_link"`
	TZ               string `json:"tz"`
	HasGoogle        bool   `json:"has_google"`
	GoogleSubject    string `json:"google_subject"`
	GoogleCalendarID string `json:"google_calendar_id"`
}

func (s *Services) GetIntegrations(ctx context.Context, workspaceID uuid.UUID) (IntegrationsView, error) {
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return IntegrationsView{}, err
	}
	encG, subjectG, calIDG, _ := s.Store.GetGoogleConfig(ctx, workspaceID)
	return IntegrationsView{
		MeetLink:         w.MeetLink,
		TZ:               w.TZ,
		HasGoogle:        len(encG) > 0 && subjectG != "",
		GoogleSubject:    subjectG,
		GoogleCalendarID: calIDG,
	}, nil
}

func (s *Services) PatchIntegrations(ctx context.Context, workspaceID uuid.UUID, meetLink, tz string) error {
	if meetLink == "" && tz == "" {
		return nil
	}
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if meetLink == "" {
		meetLink = w.MeetLink
	}
	if tz == "" {
		tz = w.TZ
	}
	return s.Store.UpdateWorkspace(ctx, workspaceID, meetLink, tz)
}

// SetGoogleConfig encrypts and stores per-workspace Google credentials. An empty
// saJSON keeps the existing key (so subject/calendar can be updated alone).
func (s *Services) SetGoogleConfig(ctx context.Context, workspaceID uuid.UUID, saJSON, subject, calendarID string) error {
	var enc []byte
	if saJSON != "" {
		e, err := s.Cipher.Encrypt(saJSON)
		if err != nil {
			return err
		}
		enc = e
	} else {
		e, _, _, err := s.Store.GetGoogleConfig(ctx, workspaceID)
		if err != nil {
			return err
		}
		enc = e
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	return s.Store.SetGoogleConfig(ctx, workspaceID, enc, subject, calendarID)
}

func (s *Services) VerifyIntegrations(ctx context.Context, workspaceID uuid.UUID) error {
	_, err := s.VerifyGoogleIntegration(ctx, workspaceID)
	return err
}

func (s *Services) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]postgres.Member, error) {
	return s.Store.ListMembers(ctx, workspaceID)
}

func (s *Services) AddMember(ctx context.Context, workspaceID uuid.UUID, username, role string) (postgres.Member, error) {
	return s.Store.AddMember(ctx, workspaceID, username, role)
}

func (s *Services) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	return s.Store.DeleteMember(ctx, memberID)
}
