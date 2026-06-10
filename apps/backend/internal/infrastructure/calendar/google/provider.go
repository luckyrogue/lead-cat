package google

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	"github.com/google/uuid"
	googleoauth "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/crypto"
	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// Provider builds a per-organization Google Calendar client from the organization's
// encrypted service-account credentials. Built adapters are cached, keyed by a
// hash of the encrypted creds + subject + calendar id, so changing any of them
// transparently yields a fresh client (no explicit invalidation needed).
type Provider struct {
	store  *postgres.Store
	cipher *crypto.TokenCipher
	cache  sync.Map // key string -> *adapter
}

func NewProvider(store *postgres.Store, cipher *crypto.TokenCipher) *Provider {
	return &Provider{store: store, cipher: cipher}
}

func (p *Provider) For(ctx context.Context, organizationID uuid.UUID) (docalendar.Service, error) {
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
