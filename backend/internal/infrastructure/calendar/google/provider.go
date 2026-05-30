package google

import (
	"context"

	"github.com/google/uuid"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/crypto"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
)

// Provider builds a per-workspace Google Calendar client from the workspace's
// encrypted service-account credentials.
type Provider struct {
	store  *postgres.Store
	cipher *crypto.TokenCipher
}

func NewProvider(store *postgres.Store, cipher *crypto.TokenCipher) *Provider {
	return &Provider{store: store, cipher: cipher}
}

func (p *Provider) For(ctx context.Context, workspaceID uuid.UUID) (application.CalendarService, error) {
	enc, subject, calendarID, err := p.store.GetGoogleConfig(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if len(enc) == 0 || subject == "" {
		return nil, application.ErrGoogleNotConfigured
	}
	saJSON, err := p.cipher.Decrypt(enc)
	if err != nil {
		return nil, err
	}
	jwtCfg, err := googleoauth.JWTConfigFromJSON([]byte(saJSON), calendar.CalendarScope)
	if err != nil {
		return nil, err
	}
	jwtCfg.Subject = subject // domain-wide delegation
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(jwtCfg.Client(ctx)))
	if err != nil {
		return nil, err
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	return &adapter{svc: svc, calendarID: calendarID}, nil
}
