package google

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application"
)

type Provider struct {
	clientID, clientSecret string
	verifier               *oidc.IDTokenVerifier
	endpoint               oauth2.Endpoint
}

func New(ctx context.Context, clientID, clientSecret string) (*Provider, error) {
	p, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, err
	}
	return &Provider{
		clientID:     clientID,
		clientSecret: clientSecret,
		verifier:     p.Verifier(&oidc.Config{ClientID: clientID}),
		endpoint:     p.Endpoint(),
	}, nil
}

func (p *Provider) Name() string { return "google" }

func (p *Provider) oauth(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		Endpoint:     p.endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

func (p *Provider) AuthURL(state, challenge, redirectURL string) string {
	return p.oauth(redirectURL).AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

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
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
		Sub     string `json:"sub"`
	}
	if err := idt.Claims(&c); err != nil {
		return application.SSOProfile{}, err
	}
	return application.SSOProfile{
		Email:     c.Email,
		Name:      c.Name,
		AvatarURL: c.Picture,
		Subject:   c.Sub,
		Provider:  "google",
	}, nil
}
