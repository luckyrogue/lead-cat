package stub

import (
	"context"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

// Provider always resolves to the stub Service, regardless of workspace.
// Used when CALENDAR_STUB=true (local/CI).
type Provider struct{ svc *Service }

func NewProvider() *Provider { return &Provider{svc: New()} }

func (p *Provider) For(_ context.Context, _ uuid.UUID) (docalendar.Service, error) {
	return p.svc, nil
}
