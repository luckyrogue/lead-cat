package microsoft

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

const graphBaseURL = "https://graph.microsoft.com/v1.0"

type connStore interface {
	UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error
}

type oauthConfigProvider interface {
	OAuthConfig(redirectURL string) *oauth2.Config
}

type Factory struct {
	conns     connStore
	connector oauthConfigProvider
	baseURL   string
}

func NewFactory(conns connStore, connector oauthConfigProvider) *Factory {
	return &Factory{conns: conns, connector: connector, baseURL: graphBaseURL}
}

func (f *Factory) For(ctx context.Context, conn model.CalendarConnection) (docalendar.Service, bool) {
	if f.connector == nil {
		return nil, false
	}
	cfg := f.connector.OAuthConfig("")
	base := cfg.TokenSource(ctx, &oauth2.Token{AccessToken: conn.AccessToken, RefreshToken: conn.RefreshToken, Expiry: conn.Expiry})
	src := &savingSource{base: oauth2.ReuseTokenSource(nil, base), save: func(tok *oauth2.Token) {
		conn.AccessToken, conn.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			conn.RefreshToken = tok.RefreshToken
		}
		_ = f.conns.UpsertCalendarConnection(ctx, conn)
	}}
	return newAdapter(oauth2.NewClient(ctx, src), f.baseURL), true
}
