package commitsreport

import (
	"context"
	"testing"
	"time"

	domainvcs "github.com/Jaryq-Lab/notify-bot/internal/domain/vcs"
)

type stubVCS struct{}

func (stubVCS) Enabled() bool { return true }

func (stubVCS) ListCommits(_ context.Context, _ string, _, _ time.Time, _ *time.Location) ([]domainvcs.Commit, error) {
	return []domainvcs.Commit{{SHA: "abc1234", Repo: "org/proj", Message: "fix"}}, nil
}

func TestBuilderDaily(t *testing.T) {
	b := &Builder{
		VCS:      stubVCS{},
		Mappings: []DevMapping{{Telegram: "dev", VCSLogin: "octocat"}},
	}
	loc := time.UTC
	text, err := b.Daily(context.Background(), time.Now(), loc)
	if err != nil {
		t.Fatal(err)
	}
	if text == "" {
		t.Fatal("empty report")
	}
}
