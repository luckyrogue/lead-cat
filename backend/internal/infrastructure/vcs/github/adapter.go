package github

import (
	"context"
	"time"

	domainvcs "github.com/Jaryq-Lab/notify-bot/internal/domain/vcs"
)

// Adapter wraps Client for domain/vcs.Client.
type Adapter struct {
	*Client
}

func (a *Adapter) ListCommits(ctx context.Context, author string, since, until time.Time, loc *time.Location) ([]domainvcs.Commit, error) {
	items, err := a.Client.ListCommits(ctx, author, since, until, loc)
	if err != nil {
		return nil, err
	}
	out := make([]domainvcs.Commit, len(items))
	for i, c := range items {
		out[i] = domainvcs.Commit{SHA: c.SHA, Repo: c.Repo, Message: c.Message}
	}
	return out, nil
}
