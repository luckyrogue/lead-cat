package microsoft

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application"
)

var msEndpoint = oauth2.Endpoint{
	AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
	TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
}

type CalendarConnector struct {
	clientID, clientSecret string
	endpoint               oauth2.Endpoint
}

func NewCalendarConnector(clientID, clientSecret string) *CalendarConnector {
	return &CalendarConnector{clientID: clientID, clientSecret: clientSecret, endpoint: msEndpoint}
}

func (c *CalendarConnector) Name() string { return "microsoft" }

func (c *CalendarConnector) OAuthConfig(redirectURL string) *oauth2.Config { return c.cfg(redirectURL) }

func (c *CalendarConnector) cfg(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     c.endpoint,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://graph.microsoft.com/Calendars.ReadWrite",
			"https://graph.microsoft.com/OnlineMeetings.ReadWrite",
			"offline_access", "openid", "email", "profile",
		},
	}
}

func (c *CalendarConnector) AuthURL(state, challenge, redirectURL string) string {
	return c.cfg(redirectURL).AuthCodeURL(state,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

func (c *CalendarConnector) Exchange(ctx context.Context, code, verifier, redirectURL string) (application.CalendarToken, error) {
	tok, err := c.cfg(redirectURL).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return application.CalendarToken{}, err
	}
	scopes, _ := tok.Extra("scope").(string)
	return application.CalendarToken{AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken, Expiry: tok.Expiry, Scopes: scopes}, nil
}

var _ application.CalendarConnector = (*CalendarConnector)(nil)
