package google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/luckyrogue/lead-cat/internal/application/model"
	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

type configStore interface {
	GetGoogleConfig(ctx context.Context, id uuid.UUID) (encJSON []byte, subject, calendarID string, err error)
}

type connectionStore interface {
	GetCalendarConnection(ctx context.Context, email, provider string) (model.CalendarConnection, error)
	UpsertCalendarConnection(ctx context.Context, conn model.CalendarConnection) error
}

type calendarConnector interface {
	OAuthConfig(redirectURL string) *oauth2.Config
}

var (
	_ configStore     = (*postgres.Store)(nil)
	_ connectionStore = (*postgres.Store)(nil)
)

type Provider struct {
	conns     connectionStore
	store     configStore
	cipher    *crypto.TokenCipher
	connector calendarConnector
	cache     sync.Map
}

func NewProvider(conns connectionStore, store configStore, cipher *crypto.TokenCipher, connector calendarConnector) *Provider {
	return &Provider{conns: conns, store: store, cipher: cipher, connector: connector}
}

func (p *Provider) For(ctx context.Context, organizationID uuid.UUID, organizerEmail string) (docalendar.Service, error) {
	if organizerEmail != "" && p.conns != nil && p.connector != nil {
		if svc, ok := p.userService(ctx, organizerEmail); ok {
			return svc, nil
		}
	}
	return p.saService(ctx, organizationID)
}

func (p *Provider) userService(ctx context.Context, email string) (docalendar.Service, bool) {
	conn, err := p.conns.GetCalendarConnection(ctx, email, "google")
	if err != nil {
		return nil, false
	}
	cfg := p.connector.OAuthConfig("")
	base := cfg.TokenSource(ctx, &oauth2.Token{
		AccessToken:  conn.AccessToken,
		RefreshToken: conn.RefreshToken,
		Expiry:       conn.Expiry,
	})
	src := &savingSource{base: oauth2.ReuseTokenSource(nil, base), save: func(tok *oauth2.Token) {
		conn.AccessToken, conn.Expiry = tok.AccessToken, tok.Expiry
		if tok.RefreshToken != "" {
			conn.RefreshToken = tok.RefreshToken
		}
		_ = p.conns.UpsertCalendarConnection(ctx, conn)
	}}
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, src)))
	if err != nil {
		return nil, false
	}
	return &adapter{svc: svc, calendarID: "primary"}, true
}

func (p *Provider) saService(ctx context.Context, organizationID uuid.UUID) (docalendar.Service, error) {
	enc, subject, calendarID, err := p.store.GetGoogleConfig(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 || subject == "" {
		return nil, docalendar.ErrNotConfigured
	}
	if calendarID == "" {
		calendarID = "primary"
	}

	sum := sha256.Sum256(enc)
	key := organizationID.String() + "|" + subject + "|" + calendarID + "|" + hex.EncodeToString(sum[:])
	if v, ok := p.cache.Load(key); ok {
		return v.(*adapter), nil
	}

	saJSON, err := p.cipher.Decrypt(enc)
	if err != nil {
		return nil, err
	}
	jwtCfg, err := googleoauth.JWTConfigFromJSON([]byte(saJSON), calendar.CalendarScope)
	if err != nil {
		return nil, err
	}
	jwtCfg.Subject = subject
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(jwtCfg.Client(ctx)))
	if err != nil {
		return nil, err
	}
	a := &adapter{svc: svc, calendarID: calendarID}
	p.cache.Store(key, a)
	return a, nil
}
