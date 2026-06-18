package stub

import (
	"context"

	"github.com/google/uuid"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

type Provider struct{ svc *Service }

func NewProvider() *Provider { return &Provider{svc: New()} }

func (p *Provider) For(_ context.Context, _ uuid.UUID, _ string) (docalendar.Service, error) {
	return p.svc, nil
}
