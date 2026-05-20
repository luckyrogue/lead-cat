package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	platformauth "github.com/Jaryq-Lab/notify-bot/internal/platform/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
)

type Config struct {
	WebappURL          string
	GitHubClientID     string
	GitHubClientSecret string
	GitLabClientID     string
	GitLabClientSecret string
	GitLabBaseURL      string
}

type Service struct {
	cfg    Config
	store  *postgres.Store
	sess   *platformauth.SessionStore
	gh     *oauth2.Config
	gl     *oauth2.Config
	client *http.Client
}

type statePayload struct {
	Provider string `json:"provider"`
}

type githubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

type gitlabUser struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

func NewService(cfg Config, store *postgres.Store, sess *platformauth.SessionStore) *Service {
	redirect := strings.TrimSuffix(cfg.WebappURL, "/") + "/api/auth/oauth/callback"
	s := &Service{cfg: cfg, store: store, sess: sess, client: http.DefaultClient}
	if cfg.GitHubClientID != "" {
		s.gh = &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  redirect,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		}
	}
	if cfg.GitLabClientID != "" {
		base, _ := url.Parse(cfg.GitLabBaseURL)
		if base == nil || base.String() == "" {
			base, _ = url.Parse("https://gitlab.com")
		}
		ep := gitlab.Endpoint
		ep.AuthURL = strings.TrimSuffix(base.String(), "/") + "/oauth/authorize"
		ep.TokenURL = strings.TrimSuffix(base.String(), "/") + "/oauth/token"
		s.gl = &oauth2.Config{
			ClientID:     cfg.GitLabClientID,
			ClientSecret: cfg.GitLabClientSecret,
			RedirectURL:  redirect,
			Scopes:       []string{"read_user"},
			Endpoint:     ep,
		}
	}
	return s
}

func (s *Service) AuthURL(ctx context.Context, provider string) (string, string, error) {
	var cfg *oauth2.Config
	switch provider {
	case "github":
		cfg = s.gh
	case "gitlab":
		cfg = s.gl
	default:
		return "", "", fmt.Errorf("unknown provider")
	}
	if cfg == nil {
		return "", "", fmt.Errorf("%s oauth not configured", provider)
	}
	state := fmt.Sprintf("%s-%d", provider, time.Now().UnixNano())
	if err := s.sess.Set(ctx, "oauth:"+state, statePayload{Provider: provider}, 10*time.Minute); err != nil {
		return "", "", err
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOnline), state, nil
}

func (s *Service) Exchange(ctx context.Context, state, code string) (postgres.User, error) {
	var payload statePayload
	ok, err := s.sess.Get(ctx, "oauth:"+state, &payload)
	if err != nil || !ok {
		return postgres.User{}, fmt.Errorf("oauth state expired")
	}
	_ = s.sess.Delete(ctx, "oauth:"+state)
	var cfg *oauth2.Config
	switch payload.Provider {
	case "github":
		cfg = s.gh
	case "gitlab":
		cfg = s.gl
	default:
		return postgres.User{}, fmt.Errorf("unknown provider")
	}
	if cfg == nil {
		return postgres.User{}, fmt.Errorf("oauth not configured")
	}
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return postgres.User{}, err
	}
	switch payload.Provider {
	case "github":
		return s.githubUser(ctx, tok)
	case "gitlab":
		return s.gitlabUser(ctx, tok)
	default:
		return postgres.User{}, fmt.Errorf("unknown provider")
	}
}

func (s *Service) githubUser(ctx context.Context, tok *oauth2.Token) (postgres.User, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return postgres.User{}, err
	}
	defer resp.Body.Close()
	var gu githubUser
	if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
		return postgres.User{}, err
	}
	email := gu.Email
	if email == "" {
		email = gu.Login + "@users.noreply.github.com"
	}
	return s.store.UpsertUserIdentity(ctx, platformauth.SubGitHub(gu.ID), email, "")
}

func (s *Service) gitlabUser(ctx context.Context, tok *oauth2.Token) (postgres.User, error) {
	base := strings.TrimSuffix(s.cfg.GitLabBaseURL, "/")
	if base == "" {
		base = "https://gitlab.com"
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v4/user", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return postgres.User{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var gu gitlabUser
	if err := json.Unmarshal(body, &gu); err != nil {
		return postgres.User{}, err
	}
	return s.store.UpsertUserIdentity(ctx, platformauth.SubGitLab(gu.ID), gu.Email, "")
}

func (s *Service) GitHubEnabled() bool { return s.gh != nil }
func (s *Service) GitLabEnabled() bool { return s.gl != nil }
