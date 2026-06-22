package command

import (
	"context"
	"errors"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

var ErrUnknownConnector = errors.New("unknown_connector")

// CalendarConnector is the per-provider OAuth port. The concrete connectors
// live in infrastructure and are looked up by name at request time.
type CalendarConnector interface {
	AuthURL(state, pkceChallenge, redirectURL string) string
	Exchange(ctx context.Context, code, pkceVerifier, redirectURL string) (model.CalendarToken, error)
}

type calendarStore interface {
	CreateCalendarOAuthState(ctx context.Context, st model.CalendarOAuthState) error
	ConsumeCalendarOAuthState(ctx context.Context, state string) (model.CalendarOAuthState, error)
	UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error
	DeleteCalendarConnection(ctx context.Context, email, provider string) error
}

type Calendar struct {
	Store     calendarStore
	Connector func(provider string) (CalendarConnector, bool)
}

func (c *Calendar) StartCalendarConnect(ctx context.Context, email, provider, redirectURL string) (string, error) {
	conn, ok := c.Connector(provider)
	if !ok {
		return "", ErrUnknownConnector
	}
	state, err := authweb.NewState(nil)
	if err != nil {
		return "", err
	}
	verifier, challenge, err := authweb.NewPKCE(nil)
	if err != nil {
		return "", err
	}
	if err := c.Store.CreateCalendarOAuthState(ctx, model.CalendarOAuthState{
		State: state, Email: email, Provider: provider, Verifier: verifier,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}); err != nil {
		return "", err
	}
	return conn.AuthURL(state, challenge, redirectURL), nil
}

func (c *Calendar) FinishCalendarConnect(ctx context.Context, state, code, redirectURL string) error {
	pending, err := c.Store.ConsumeCalendarOAuthState(ctx, state)
	if err != nil {
		return err
	}
	conn, ok := c.Connector(pending.Provider)
	if !ok {
		return ErrUnknownConnector
	}
	tok, err := conn.Exchange(ctx, code, pending.Verifier, redirectURL)
	if err != nil {
		return err
	}
	return c.Store.UpsertCalendarConnection(ctx, model.CalendarConnection{
		Email: pending.Email, Provider: pending.Provider,
		AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken,
		Expiry: tok.Expiry, Scopes: tok.Scopes,
	})
}

func (c *Calendar) DisconnectCalendar(ctx context.Context, email, provider string) error {
	return c.Store.DeleteCalendarConnection(ctx, email, provider)
}
