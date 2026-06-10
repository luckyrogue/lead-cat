package application

import "context"

// Auth method values stored in platform_users.auth_method and set by the
// corresponding sign-in path. Keep in sync with SSOProvider.Name() returns.
const (
	AuthMethodGoogle    = "google"
	AuthMethodMicrosoft = "microsoft"
	AuthMethodMagicLink = "magiclink"
)

// SSOProfile is the normalized identity returned by any SSO provider.
type SSOProfile struct {
	Email     string
	Name      string
	AvatarURL string
	Subject   string // provider's stable subject id
	Provider  string // "google" | "microsoft"
}

// SSOProvider abstracts an OIDC authorization-code + PKCE flow.
type SSOProvider interface {
	Name() string
	// AuthURL builds the provider authorization URL.
	AuthURL(state, pkceChallenge, redirectURL string) string
	// Exchange swaps the callback code for a verified profile.
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (SSOProfile, error)
}

// EmailSender delivers transactional email (magic-link, invites).
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
