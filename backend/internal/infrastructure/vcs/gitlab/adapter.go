package gitlab

import (
	"context"
	"time"

	domainvcs "github.com/Jaryq-Lab/notify-bot/internal/domain/vcs"
)

type Adapter struct {
	*Client
}

func (a *Adapter) ListCommits(ctx context.Context, author string, since, until time.Time, loc *time.Location) ([]domainvcs.Commit, error) {
	return a.Client.ListCommits(ctx, author, since, until, loc)
}
