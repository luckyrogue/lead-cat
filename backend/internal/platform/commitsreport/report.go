package commitsreport

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	domainvcs "github.com/Jaryq-Lab/notify-bot/internal/domain/vcs"
)

const maxCommitsPerDev = 8

type DevMapping struct {
	Telegram string
	VCSLogin string
}

type Builder struct {
	VCS      domainvcs.Client
	Mappings []DevMapping
}

func (b *Builder) Daily(ctx context.Context, day time.Time, loc *time.Location) (string, error) {
	if b.VCS == nil || !b.VCS.Enabled() {
		return "", fmt.Errorf("vcs not configured")
	}

	day = day.In(loc)
	since, until := dayBounds(day, loc)

	var sb strings.Builder
	fmt.Fprintf(&sb, "📊 Коммиты за %s\n\n", day.Format("02.01.2006"))

	for _, m := range b.Mappings {
		section, err := b.section(ctx, m, since, until, loc)
		if err != nil {
			return "", fmt.Errorf("%s: %w", m.Telegram, err)
		}
		sb.WriteString(section)
		sb.WriteByte('\n')
	}

	return strings.TrimSpace(sb.String()), nil
}

func (b *Builder) section(ctx context.Context, m DevMapping, since, until time.Time, loc *time.Location) (string, error) {
	if m.VCSLogin == "" {
		return fmt.Sprintf("@%s — VCS не привязан ⚠️", m.Telegram), nil
	}

	commits, err := b.VCS.ListCommits(ctx, m.VCSLogin, since, until, loc)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return fmt.Sprintf("@%s (%s) — 0 коммитов ⚠️", m.Telegram, m.VCSLogin), nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("@%s (%s) — %d", m.Telegram, m.VCSLogin, len(commits)))
	show := commits
	if len(show) > maxCommitsPerDev {
		show = show[:maxCommitsPerDev]
	}
	for _, c := range show {
		lines = append(lines, fmt.Sprintf("  • %s %s — %s", c.Repo, c.SHA, c.Message))
	}
	if len(commits) > maxCommitsPerDev {
		lines = append(lines, fmt.Sprintf("  … ещё %d", len(commits)-maxCommitsPerDev))
	}
	return strings.Join(lines, "\n"), nil
}

func dayBounds(day time.Time, loc *time.Location) (time.Time, time.Time) {
	y, m, d := day.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, loc)
	end := start.Add(24*time.Hour - time.Nanosecond)
	return start, end
}

func SortMappings(m []DevMapping) []DevMapping {
	out := append([]DevMapping(nil), m...)
	sort.Slice(out, func(i, j int) bool { return out[i].Telegram < out[j].Telegram })
	return out
}
