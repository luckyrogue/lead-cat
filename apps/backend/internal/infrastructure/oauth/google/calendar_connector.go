package google

import (
	"context"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"

	"github.com/luckyrogue/lead-cat/internal/application"
)

type CalendarConnector struct {
	clientID, clientSecret string
	endpoint               oauth2.Endpoint
}

func NewCalendarConnector(clientID, clientSecret string) *CalendarConnector {
	return &CalendarConnector{
		clientID:     clientID,
		clientSecret: clientSecret,
		endpoint:     googleoauth.Endpoint,
	}
}

func (c *CalendarConnector) Name() string { return "google" }

func (c *CalendarConnector) cfg(redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Endpoint:     c.endpoint,
		RedirectURL:  redirectURL,
		Scopes:       []string{calendar.CalendarEventsScope, calendar.CalendarReadonlyScope},
	}
}

func (c *CalendarConnector) AuthURL(state, challenge, redirectURL string) string {
	return c.cfg(redirectURL).AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))
}

func (c *CalendarConnector) Exchange(ctx context.Context, code, verifier, redirectURL string) (application.CalendarToken, error) {
	tok, err := c.cfg(redirectURL).Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		return application.CalendarToken{}, err
	}
	scopes, _ := tok.Extra("scope").(string)
	return application.CalendarToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
		Scopes:       scopes,
	}, nil
}

var _ application.CalendarConnector = (*CalendarConnector)(nil)
