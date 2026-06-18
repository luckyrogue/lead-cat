package application

import (
	"context"
	"errors"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	"github.com/luckyrogue/lead-cat/internal/platform/authweb"
)

var ErrUnknownConnector = errors.New("unknown_connector")

type CalendarConnectionView struct {
	Provider  string `json:"provider"`
	Connected bool   `json:"connected"`
	Email     string `json:"email"`
	Scopes    string `json:"scopes"`
}

func (s *Services) StartCalendarConnect(ctx context.Context, email, provider, redirectURL string) (string, error) {
	conn, ok := s.CalendarConnectorByName(provider)
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
	if err := s.Store.CreateCalendarOAuthState(ctx, model.CalendarOAuthState{
		State: state, Email: email, Provider: provider, Verifier: verifier,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}); err != nil {
		return "", err
	}
	return conn.AuthURL(state, challenge, redirectURL), nil
}

func (s *Services) FinishCalendarConnect(ctx context.Context, state, code, redirectURL string) error {
	pending, err := s.Store.ConsumeCalendarOAuthState(ctx, state)
	if err != nil {
		return err
	}
	conn, ok := s.CalendarConnectorByName(pending.Provider)
	if !ok {
		return ErrUnknownConnector
	}
	tok, err := conn.Exchange(ctx, code, pending.Verifier, redirectURL)
	if err != nil {
		return err
	}
	return s.Store.UpsertCalendarConnection(ctx, model.CalendarConnection{
		Email: pending.Email, Provider: pending.Provider,
		AccessToken: tok.AccessToken, RefreshToken: tok.RefreshToken,
		Expiry: tok.Expiry, Scopes: tok.Scopes,
	})
}

func (s *Services) ListCalendarConnections(ctx context.Context, email string) ([]CalendarConnectionView, error) {
	rows, err := s.Store.ListCalendarConnections(ctx, email)
	if err != nil {
		return nil, err
	}
	out := []CalendarConnectionView{}
	for _, r := range rows {
		out = append(out, CalendarConnectionView{Provider: r.Provider, Connected: true, Email: r.Email, Scopes: r.Scopes})
	}
	return out, nil
}

func (s *Services) DisconnectCalendar(ctx context.Context, email, provider string) error {
	return s.Store.DeleteCalendarConnection(ctx, email, provider)
}
