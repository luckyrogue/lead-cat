package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/command"
	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/application/query"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
	"github.com/luckyrogue/lead-cat/internal/platform/emailtemplates"
)

type ChatSyncer func(ctx context.Context, organizationID uuid.UUID) (int, error)

type Services struct {
	Store        Repository
	Cipher       Cipher
	Queue        JobQueue
	Calendar     CalendarProvider
	Busy         BusyResolver
	GoogleProber GoogleProber
	Log          *zap.Logger
	Bot          *bot.Bot
	Queries      *query.Meetings
	Commands     *command.Meetings
	syncChat     ChatSyncer

	sso        map[string]SSOProvider
	connectors map[string]CalendarConnector
	email      EmailSender
	magic      *magicLinkService
	sessions   *webSessionService
	appBaseURL string
	webappURL  string
}

func (s *Services) ConfigureWebAuth(sso map[string]SSOProvider, email EmailSender, appBaseURL, webappURL string, sessionTTL, magicTTL time.Duration) {
	s.sso = sso
	s.email = email
	s.appBaseURL = appBaseURL
	s.webappURL = webappURL
	s.magic = newMagicLinkService(s.Store, email, appBaseURL, webappURL, magicTTL, time.Now)
	s.sessions = newWebSessionService(s.Store, sessionTTL, time.Now)
}

func (s *Services) AppBaseURL() string { return s.appBaseURL }

func (s *Services) ResolveWebUser(ctx context.Context, rawToken string) (model.PlatformUser, bool, error) {
	ws, ok, err := s.sessions.ResolveSession(ctx, rawToken)
	if err != nil || !ok {
		return model.PlatformUser{}, false, err
	}
	return s.Store.GetPlatformUserByID(ctx, ws.UserID)
}

func (s *Services) CreateWebSession(ctx context.Context, userID uuid.UUID, ua, ip string) (string, error) {
	return s.sessions.CreateSession(ctx, userID, ua, ip)
}

func (s *Services) RevokeWebSession(ctx context.Context, rawToken string) error {
	return s.sessions.RevokeSession(ctx, rawToken)
}

func (s *Services) RequestMagicLink(ctx context.Context, email, language string) error {
	if language == "" {
		if lang, ok, err := s.Store.GetPlatformUserLanguageByEmail(ctx, email); err == nil && ok {
			language = lang
		}
	}
	return s.magic.RequestMagicLink(ctx, email, language)
}

func (s *Services) VerifyMagicLink(ctx context.Context, rawToken string) (string, error) {
	return s.magic.VerifyMagicLink(ctx, rawToken)
}

func (s *Services) UpsertWebIdentity(ctx context.Context, email, name, avatarURL, authMethod string) (model.PlatformUser, error) {
	return s.Store.UpsertWebIdentity(ctx, email, name, avatarURL, authMethod)
}

func (s *Services) AcceptInvitesForEmail(ctx context.Context, email string, userID uuid.UUID) (int, error) {
	return s.Store.AcceptInvitesForEmail(ctx, email, userID)
}

func (s *Services) SSOProviderByName(name string) (SSOProvider, bool) {
	p, ok := s.sso[name]
	return p, ok
}

func (s *Services) ConfigureCalendarConnectors(c map[string]CalendarConnector) { s.connectors = c }

func (s *Services) CalendarConnectorByName(name string) (CalendarConnector, bool) {
	c, ok := s.connectors[name]
	return c, ok
}

func (s *Services) ListOrganizationsForUser(ctx context.Context, userID uuid.UUID) ([]model.Organization, error) {
	return s.Store.ListOrganizationsForUser(ctx, userID)
}

func (s *Services) CreateOrganizationForOwner(ctx context.Context, name string, ownerUserID uuid.UUID) (model.Organization, error) {
	base := slugify(name)
	if base == "" {
		base = "org"
	}
	suffix, err := authweb.NewState(nil)
	if err != nil {
		return model.Organization{}, err
	}
	slug := base + "-" + suffix[:6]
	org, err := s.Store.CreateOrganization(ctx, name, slug, ownerUserID)
	if err != nil {
		return model.Organization{}, err
	}
	s.sendWelcomeEmail(ctx, ownerUserID)
	return org, nil
}

func (s *Services) WireCQRS() {
	if s.Queries == nil {
		s.Queries = query.NewMeetings(s)
	}
	if s.Commands == nil {
		s.Commands = &command.Meetings{
			Store:    s.Store,
			Calendar: s.Calendar,
			Queue:    s.Queue,
			Log:      s.Log,
		}
	}
}

func (s *Services) Ping(ctx context.Context) error {
	return s.Store.Ping(ctx)
}

func (s *Services) GetBotUserByTelegramID(ctx context.Context, telegramID int64) (model.BotUser, error) {
	return s.Store.GetBotUserByTelegramID(ctx, telegramID)
}

func (s *Services) PlatformUserIDForTelegram(ctx context.Context, telegramID int64) (uuid.UUID, bool, error) {
	return s.Store.GetPlatformUserIDByTelegramID(ctx, telegramID)
}

func (s *Services) ListAudit(ctx context.Context, f model.AuditFilter) ([]model.AuditEntry, error) {
	return s.Store.ListAuditEntries(ctx, f)
}

func (s *Services) ListOrganizationsWithGoogle(ctx context.Context) ([]uuid.UUID, error) {
	return s.Store.ListOrganizationsWithGoogle(ctx)
}

func (s *Services) MiniAppMeetingDTO(ctx context.Context, m model.Meeting, loc *time.Location) query.MiniAppMeeting {
	return query.MeetingDTO(ctx, s.Store, m, loc)
}

var ErrTelegramLinkedToOtherAccount = errors.New("telegram already linked to another account")

var ErrUnknownParticipant = errors.New("unknown participant email")

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

func (s *Services) GetOrganization(ctx context.Context, id uuid.UUID) (model.Organization, error) {
	return s.Store.GetOrganization(ctx, id)
}

func (s *Services) LinkChat(ctx context.Context, organizationID uuid.UUID, chatID int64) error {
	return s.Store.LinkChat(ctx, organizationID, chatID)
}

func (s *Services) SetChatSyncer(fn ChatSyncer) {
	s.syncChat = fn
}

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

func (s *Services) ListMembers(ctx context.Context, organizationID uuid.UUID) ([]model.Member, error) {
	return s.Store.ListMembers(ctx, organizationID)
}

func (s *Services) InviteToOrg(ctx context.Context, orgID uuid.UUID, email, role string, inviterUserID uuid.UUID) (model.OrganizationInvite, error) {
	raw, err := authweb.NewState(nil)
	if err != nil {
		return model.OrganizationInvite{}, err
	}
	inv, err := s.Store.CreateInvite(ctx, orgID, email, role, authweb.HashToken(raw), time.Now().Add(14*24*time.Hour), inviterUserID)
	if err != nil {
		return model.OrganizationInvite{}, err
	}
	if s.email != nil {
		org, oerr := s.Store.GetOrganization(ctx, orgID)
		if oerr != nil {
			s.Log.Warn("invite_email_org_lookup_failed", zap.Error(oerr))
		} else {
			lang := "ru"
			if l, ok, lerr := s.Store.GetPlatformUserLanguageByEmail(ctx, email); lerr == nil && ok {
				lang = l
			}
			loginURL := s.appBaseURL + "/login"
			inviterName := ""
			if inviter, ok, ierr := s.Store.GetPlatformUserByID(ctx, inviterUserID); ierr == nil && ok {
				inviterName = emailtemplates.FirstNameFromDisplay("", inviter.Email)
			}
			subject, text, html, rerr := emailtemplates.RenderOrgInvite(emailtemplates.InviteData{
				Language:    lang,
				OrgName:     org.Name,
				RoleLabel:   emailtemplates.RoleLabelLocal(lang, role),
				LoginURL:    loginURL,
				InviterName: inviterName,
			})
			if rerr != nil {
				s.Log.Warn("invite_email_render_failed", zap.Error(rerr))
			} else if err := s.email.SendMultipart(ctx, email, subject, text, html, ""); err != nil {
				s.Log.Warn("invite_email_failed", zap.Error(err))
			}
		}
	}
	return inv, nil
}

func (s *Services) ListOrgInvites(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationInvite, error) {
	return s.Store.ListInvites(ctx, orgID)
}

func (s *Services) DeleteOrgInvite(ctx context.Context, orgID, inviteID uuid.UUID) error {
	return s.Store.DeleteInvite(ctx, orgID, inviteID)
}

func (s *Services) ListOrgMembers(ctx context.Context, orgID uuid.UUID) ([]model.Member, error) {
	return s.Store.ListOrgMembers(ctx, orgID)
}

func (s *Services) RemoveOrgMember(ctx context.Context, orgID, targetUserID uuid.UUID) error {
	members, err := s.Store.ListOrgMembers(ctx, orgID)
	if err != nil {
		return err
	}
	views, idx := memberViews(members, targetUserID)
	if idx < 0 {
		return nil
	}
	if err := canDemoteOrRemove(views, idx); err != nil {
		return err
	}
	return s.Store.RemoveMember(ctx, orgID, targetUserID)
}

func (s *Services) SetOrgMemberRole(ctx context.Context, orgID, targetUserID uuid.UUID, newRole string) error {
	members, err := s.Store.ListOrgMembers(ctx, orgID)
	if err != nil {
		return err
	}
	views, idx := memberViews(members, targetUserID)
	if idx < 0 {
		return nil
	}

	if newRole != "owner" {
		if err := canDemoteOrRemove(views, idx); err != nil {
			return err
		}
	}
	return s.Store.UpdateMemberRole(ctx, orgID, targetUserID, newRole)
}

func (s *Services) AddMember(ctx context.Context, organizationID uuid.UUID, username, role string) (model.Member, error) {
	return s.Store.AddMember(ctx, organizationID, username, role)
}

func (s *Services) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	return s.Store.DeleteMember(ctx, memberID)
}

func (s *Services) sendWelcomeEmail(ctx context.Context, ownerUserID uuid.UUID) {
	if s.email == nil {
		return
	}
	user, ok, err := s.Store.GetPlatformUserByID(ctx, ownerUserID)
	if err != nil || !ok || user.Email == "" {
		return
	}
	lang := user.Language
	if lang == "" {
		lang = "ru"
	}
	unsub := s.webappURL
	if unsub == "" {
		unsub = "#"
	}
	appURL := s.webappURL
	if appURL == "" {
		appURL = "#"
	}
	subject, text, html, rerr := emailtemplates.RenderWelcome(emailtemplates.WelcomeData{
		Language:       lang,
		FirstName:      emailtemplates.FirstNameFromDisplay("", user.Email),
		AppURL:         appURL,
		UnsubscribeURL: unsub,
	})
	if rerr != nil {
		s.Log.Warn("welcome_email_render_failed", zap.Error(rerr))
		return
	}
	if err := s.email.SendMultipart(ctx, user.Email, subject, text, html, unsub); err != nil {
		s.Log.Warn("welcome_email_failed", zap.Error(err))
	}
}
