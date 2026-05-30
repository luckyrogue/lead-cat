package stub

import (
	"context"

	"github.com/google/uuid"

	"github.com/Jaryq-Lab/notify-bot/internal/application"
)

// Provider always resolves to the stub Service, regardless of workspace.
// Used when CALENDAR_STUB=true (local/CI).
type Provider struct{ svc *Service }

func NewProvider() *Provider { return &Provider{svc: New()} }

func (p *Provider) For(_ context.Context, _ uuid.UUID) (application.CalendarService, error) {
	return p.svc, nil
}
