package microsoft

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application"
)

// Provider implements application.SSOProvider for Microsoft OIDC with PKCE.
//
// The /common endpoint multiplexes multiple per-tenant issuers, so issuer
// verification must be disabled (SkipIssuerCheck: true).
type Provider struct {
	clientID, clientSecret string
	verifier               *oidc.IDTokenVerifier
	endpoint               oauth2.Endpoint
}

// New initialises the Microsoft OIDC provider via discovery. It performs a live
// HTTP call to the /common/v2.0 well-known endpoint and is intended to be
// called once at application startup.
func New(ctx context.Context, clientID, clientSecret string) (*Provider, error) {
	p, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/common/v2.0")
	if err != nil {
		return nil, err
	}
	return &Provider{
		clientID:     clientID,
		clientSecret: clientSecret,
		verifier:     p.Verifier(&oidc.Config{ClientID: clientID, SkipIssuerCheck: true}),
		endpoint:     p.Endpoint(),
	}, nil
}

// Name satisfies application.SSOProvider.
func (p *Provider) Name() string { return "microsoft" }

func (p *Provider) oauth(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Endpoint:     p.endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

// AuthURL builds the Microsoft authorization URL with PKCE S256 challenge.
func (p *Provider) AuthURL(state, challenge, redirectURL string) string {
	return p.oauth(redirectURL).AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

// Exchange swaps the authorization code for a verified application.SSOProfile.
// Microsoft v2 id_tokens may have an empty "email" claim; preferred_username
// is used as a fallback.
func (p *Provider) Exchange(ctx context.Context, code, verifier, redirectURL string) (application.SSOProfile, error) {
	tok, err := p.oauth(redirectURL).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return application.SSOProfile{}, err
	}
	raw, _ := tok.Extra("id_token").(string)
	idt, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return application.SSOProfile{}, err
	}
	var c struct {
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
		Sub               string `json:"sub"`
	}
	if err := idt.Claims(&c); err != nil {
		return application.SSOProfile{}, err
	}
	email := c.Email
	if email == "" {
		email = c.PreferredUsername
	}
	return application.SSOProfile{
		Email:     email,
		Name:      c.Name,
		AvatarURL: "", // Microsoft id_token does not include a picture claim
		Subject:   c.Sub,
		Provider:  "microsoft",
	}, nil
}
