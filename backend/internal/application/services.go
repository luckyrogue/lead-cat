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
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

// ChatSyncer is a function that syncs chat administrators into organization_members.
// It is injected at startup to avoid an import cycle between application and
// the telegram infrastructure package.
type ChatSyncer func(ctx context.Context, organizationID uuid.UUID) (int, error)

type Services struct {
	Store    *postgres.Store
	Cipher   *crypto.TokenCipher
	Queue    *asynqqueue.Client
	Calendar CalendarProvider
	Log      *zap.Logger
	Bot      *bot.Bot
	Queries  *query.Meetings
	syncChat ChatSyncer

	sso        map[string]SSOProvider
	email      EmailSender
	magic      *magicLinkService
	sessions   *webSessionService
	appBaseURL string
}

// ConfigureWebAuth constructs the web-auth services. Call once at startup after Store is set.
func (s *Services) ConfigureWebAuth(sso map[string]SSOProvider, email EmailSender, appBaseURL string, sessionTTL, magicTTL time.Duration) {
	s.sso = sso
	s.email = email
	s.appBaseURL = appBaseURL
	s.magic = newMagicLinkService(s.Store, email, appBaseURL, magicTTL, time.Now)
	s.sessions = newWebSessionService(s.Store, sessionTTL, time.Now)
}

// AppBaseURL returns the configured public base URL (for building redirects/links).
func (s *Services) AppBaseURL() string { return s.appBaseURL }

// ResolveWebUser resolves a session cookie to the web account (for WebAuth middleware).
func (s *Services) ResolveWebUser(ctx context.Context, rawToken string) (postgres.PlatformUser, bool, error) {
	ws, ok, err := s.sessions.ResolveSession(ctx, rawToken)
	if err != nil || !ok {
		return postgres.PlatformUser{}, false, err
	}
	return s.Store.GetPlatformUserByID(ctx, ws.UserID)
}

func (s *Services) CreateWebSession(ctx context.Context, userID uuid.UUID, ua, ip string) (string, error) {
	return s.sessions.CreateSession(ctx, userID, ua, ip)
}

func (s *Services) RevokeWebSession(ctx context.Context, rawToken string) error {
	return s.sessions.RevokeSession(ctx, rawToken)
}

func (s *Services) RequestMagicLink(ctx context.Context, email string) error {
	return s.magic.RequestMagicLink(ctx, email)
}

func (s *Services) VerifyMagicLink(ctx context.Context, rawToken string) (string, error) {
	return s.magic.VerifyMagicLink(ctx, rawToken)
}

func (s *Services) UpsertWebIdentity(ctx context.Context, email, name, avatarURL, authMethod string) (postgres.PlatformUser, error) {
	return s.Store.UpsertWebIdentity(ctx, email, name, avatarURL, authMethod)
}

func (s *Services) AcceptInvitesForEmail(ctx context.Context, email string, userID uuid.UUID) (int, error) {
	return s.Store.AcceptInvitesForEmail(ctx, email, userID)
}

func (s *Services) SSOProviderByName(name string) (SSOProvider, bool) {
	p, ok := s.sso[name]
	return p, ok
}

func (s *Services) ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]postgres.Organization, error) {
	return s.Store.ListOrganizationsForUser(ctx, userID)
}

// CreateOrganizationForOwner derives a unique slug from name and creates the org with the user as owner.
func (s *Services) CreateOrganizationForOwner(ctx context.Context, name string, ownerUserID uuid.UUID) (postgres.Organization, error) {
	base := slugify(name)
	if base == "" {
		base = "org"
	}
	suffix, err := authweb.NewState(nil) // url-safe random
	if err != nil {
		return postgres.Organization{}, err
	}
	slug := base + "-" + suffix[:6]
	return s.Store.CreateOrganization(ctx, name, slug, ownerUserID)
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

func (s *Services) ListOrganizationsWithGoogle(ctx context.Context) ([]uuid.UUID, error) {
	return s.Store.ListOrganizationsWithGoogle(ctx)
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

func (s *Services) GetOrganization(ctx context.Context, id uuid.UUID) (postgres.Organization, error) {
	return s.Store.GetOrganization(ctx, id)
}

func (s *Services) LinkChat(ctx context.Context, organizationID uuid.UUID, chatID int64) error {
	return s.Store.LinkChat(ctx, organizationID, chatID)
}

// SetChatSyncer injects the ChatSyncer function after bot initialisation.
// Called from main after bot.New to avoid an import cycle.
func (s *Services) SetChatSyncer(fn ChatSyncer) {
	s.syncChat = fn
}

// SyncChatMembers imports current Telegram chat administrators into the
// organization_members table. Thin wrapper around the telegram helper so HTTP
// handlers don't reach into infrastructure.
func (s *Services) SyncChatMembers(ctx context.Context, organizationID uuid.UUID) (int, error) {
	if s.syncChat == nil {
		return 0, fmt.Errorf("bot not configured")
	}
	return s.syncChat(ctx, organizationID)
}

type IntegrationsView struct {
	MeetLink         string `json:"meet_link"`
	TZ               string `json:"tz"`
	HasGoogle        bool   `json:"has_google"`
	GoogleSubject    string `json:"google_subject"`
	GoogleCalendarID string `json:"google_calendar_id"`
}

func (s *Services) GetIntegrations(ctx context.Context, organizationID uuid.UUID) (IntegrationsView, error) {
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return IntegrationsView{}, err
	}
	encG, subjectG, calIDG, _ := s.Store.GetGoogleConfig(ctx, organizationID)
	return IntegrationsView{
		MeetLink:         w.MeetLink,
		TZ:               w.TZ,
		HasGoogle:        len(encG) > 0 && subjectG != "",
		GoogleSubject:    subjectG,
		GoogleCalendarID: calIDG,
	}, nil
}

func (s *Services) PatchIntegrations(ctx context.Context, organizationID uuid.UUID, meetLink, tz string) error {
	if meetLink == "" && tz == "" {
		return nil
	}
	w, err := s.Store.GetOrganization(ctx, organizationID)
	if err != nil {
		return err
	}
	if meetLink == "" {
		meetLink = w.MeetLink
	}
	if tz == "" {
		tz = w.TZ
	}
	return s.Store.UpdateOrganization(ctx, organizationID, meetLink, tz)
}

// SetGoogleConfig encrypts and stores per-organization Google credentials. An empty
// saJSON keeps the existing key (so subject/calendar can be updated alone).
func (s *Services) SetGoogleConfig(ctx context.Context, organizationID uuid.UUID, saJSON, subject, calendarID string) error {
	var enc []byte
	if saJSON != "" {
		e, err := s.Cipher.Encrypt(saJSON)
		if err != nil {
			return err
		}
		enc = e
	} else {
		e, _, _, err := s.Store.GetGoogleConfig(ctx, organizationID)
		if err != nil {
			return err
		}
		enc = e
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	return s.Store.SetGoogleConfig(ctx, organizationID, enc, subject, calendarID)
}

func (s *Services) VerifyIntegrations(ctx context.Context, organizationID uuid.UUID) error {
	_, err := s.VerifyGoogleIntegration(ctx, organizationID)
	return err
}

func (s *Services) ListMembers(ctx context.Context, organizationID uuid.UUID) ([]postgres.Member, error) {
	return s.Store.ListMembers(ctx, organizationID)
}

func (s *Services) AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (postgres.Member, error) {
	return s.Store.AddMember(ctx, organizationID, username, role)
}

func (s *Services) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	return s.Store.DeleteMember(ctx, memberID)
}
