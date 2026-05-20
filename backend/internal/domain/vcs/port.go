package vcs

import (
	"context"
	"time"
)

type Commit struct {
	SHA     string
	Repo    string
	Message string
}

type Client interface {
	Enabled() bool
	ListCommits(ctx context.Context, author string, since, until time.Time, loc *time.Location) ([]Commit, error)
}
