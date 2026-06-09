package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/scenario"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	asynqqueue "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/queue/asynq"
	ghvcs "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/vcs/github"
	glvcs "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/vcs/gitlab"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/scenario_scheduler"
)

// ChatSyncer is a function that syncs chat administrators into workspace_members.
// It is injected at startup to avoid an import cycle between application and
// the telegram infrastructure package.
type ChatSyncer func(ctx context.Context, workspaceID uuid.UUID) (int, error)

type Services struct {
	Store      *postgres.Store
	Cipher     *crypto.TokenCipher
	Queue      *asynqqueue.Client
	Calendar   CalendarProvider
	Log        *zap.Logger
	Bot        *bot.Bot
	syncChat   ChatSyncer
}

var ErrTelegramLinkedToOtherAccount = errors.New("telegram already linked to another account")

func (s *Services) GetMe(ctx context.Context, authSub string) (postgres.User, error) {
	return s.Store.GetUserBySub(ctx, authSub)
}

func (s *Services) LinkTelegram(ctx context.Context, userID uuid.UUID, telegramID int64, username string) error {
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

func (s *Services) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]postgres.Workspace, error) {
	return s.Store.ListWorkspacesForUser(ctx, userID)
}

func (s *Services) CreateWorkspace(ctx context.Context, slug, name string, ownerID uuid.UUID) (postgres.Workspace, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return postgres.Workspace{}, fmt.Errorf("slug required")
	}
	w, err := s.Store.CreateWorkspace(ctx, slug, name, ownerID)
	if err != nil {
		return w, err
	}
	_, _ = s.Store.AddMember(ctx, w.ID, "owner", "owner")
	return w, err
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
	VCSProvider  string `json:"vcs_provider"`
	VCSNamespace string `json:"vcs_namespace"`
	VCSBaseURL   string `json:"vcs_base_url,omitempty"`
	HasVCSToken  bool   `json:"has_vcs_token"`
	MeetLink     string `json:"meet_link"`
	TZ           string `json:"tz"`

	HasGoogle        bool   `json:"has_google"`
	GoogleSubject    string `json:"google_subject"`
	GoogleCalendarID string `json:"google_calendar_id"`
}

func (s *Services) GetIntegrations(ctx context.Context, workspaceID uuid.UUID) (IntegrationsView, error) {
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return IntegrationsView{}, err
	}
	base := ""
	if w.VCSBaseURL != nil {
		base = *w.VCSBaseURL
	}
	encG, subjectG, calIDG, _ := s.Store.GetGoogleConfig(ctx, workspaceID)
	return IntegrationsView{
		VCSProvider:      w.VCSProvider,
		VCSNamespace:     w.VCSNamespace,
		VCSBaseURL:       base,
		HasVCSToken:      w.HasVCSToken,
		MeetLink:         w.MeetLink,
		TZ:               w.TZ,
		HasGoogle:        len(encG) > 0 && subjectG != "",
		GoogleSubject:    subjectG,
		GoogleCalendarID: calIDG,
	}, nil
}

func (s *Services) PatchIntegrations(ctx context.Context, workspaceID uuid.UUID, provider, namespace, baseURL, token, meetLink, tz string) error {
	var basePtr *string
	if baseURL != "" {
		basePtr = &baseURL
	}
	if token != "" {
		enc, err := s.Cipher.Encrypt(token)
		if err != nil {
			return err
		}
		if err := s.Store.SetVCSToken(ctx, workspaceID, provider, namespace, basePtr, enc); err != nil {
			return err
		}
	} else if provider != "" || namespace != "" || basePtr != nil {
		enc, _ := s.Store.GetVCSTokenEnc(ctx, workspaceID)
		if err := s.Store.SetVCSToken(ctx, workspaceID, provider, namespace, basePtr, enc); err != nil {
			return err
		}
	}
	if meetLink != "" || tz != "" {
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
	return nil
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
	w, err := s.Store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	enc, err := s.Store.GetVCSTokenEnc(ctx, workspaceID)
	if err != nil || len(enc) == 0 {
		return fmt.Errorf("vcs token not configured")
	}
	token, err := s.Cipher.Decrypt(enc)
	if err != nil || token == "" {
		return fmt.Errorf("vcs token invalid")
	}
	switch w.VCSProvider {
	case "gitlab":
		base := ""
		if w.VCSBaseURL != nil {
			base = *w.VCSBaseURL
		}
		if err := glvcs.New(token, w.VCSNamespace, base).Ping(ctx); err != nil {
			return fmt.Errorf("vcs verify failed: %w", err)
		}
	default:
		gh := ghvcs.NewClient(token, w.VCSNamespace)
		if !gh.Enabled() {
			return fmt.Errorf("vcs not configured")
		}
	}
	return nil
}

func (s *Services) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]postgres.MemberWithVCS, error) {
	return s.Store.ListMembersWithVCS(ctx, workspaceID)
}

func (s *Services) SetMemberVCS(ctx context.Context, workspaceID uuid.UUID, username, githubLogin, gitlabLogin string) error {
	return s.Store.SetMemberVCS(ctx, workspaceID, username, githubLogin, gitlabLogin)
}

func (s *Services) AddMember(ctx context.Context, workspaceID uuid.UUID, username, role string) (postgres.Member, error) {
	return s.Store.AddMember(ctx, workspaceID, username, role)
}

func (s *Services) DeleteMember(ctx context.Context, memberID uuid.UUID) error {
	return s.Store.DeleteMember(ctx, memberID)
}

func (s *Services) ListScenarios(ctx context.Context, workspaceID uuid.UUID) ([]postgres.Scenario, error) {
	return s.Store.ListScenarios(ctx, workspaceID)
}

func (s *Services) GetScenario(ctx context.Context, scenarioID uuid.UUID) (postgres.Scenario, error) {
	return s.Store.GetScenario(ctx, scenarioID)
}

func (s *Services) CreateScenario(ctx context.Context, workspaceID uuid.UUID, name string, def json.RawMessage) (postgres.Scenario, error) {
	if len(def) == 0 {
		def = scenario.PresetCommits()
	}
	if err := scenario.ValidateDefinition(def); err != nil {
		return postgres.Scenario{}, err
	}
	return s.Store.CreateScenario(ctx, workspaceID, name, def)
}

func (s *Services) UpdateScenario(ctx context.Context, scenarioID uuid.UUID, name string, enabled *bool, def json.RawMessage) (postgres.Scenario, error) {
	sc, err := s.Store.GetScenario(ctx, scenarioID)
	if err != nil {
		return sc, err
	}
	if name != "" {
		sc.Name = name
	}
	if enabled != nil {
		sc.Enabled = *enabled
	}
	if len(def) > 0 {
		if err := scenario.ValidateDefinition(def); err != nil {
			return postgres.Scenario{}, err
		}
		sc.Definition = def
	}
	if err := s.Store.UpdateScenario(ctx, scenarioID, sc.Name, sc.Enabled, sc.Definition); err != nil {
		return sc, err
	}
	return s.Store.GetScenario(ctx, scenarioID)
}

func (s *Services) DeleteScenario(ctx context.Context, scenarioID uuid.UUID) error {
	return s.Store.DeleteScenario(ctx, scenarioID)
}

func (s *Services) RunScenario(ctx context.Context, scenarioID, workspaceID uuid.UUID) (uuid.UUID, error) {
	return scenario_scheduler.EnqueueManual(ctx, s.Store, s.Queue, scenarioID, workspaceID)
}

func (s *Services) ListRuns(ctx context.Context, scenarioID uuid.UUID) ([]postgres.ScenarioRun, error) {
	return s.Store.ListRuns(ctx, scenarioID, 30)
}
