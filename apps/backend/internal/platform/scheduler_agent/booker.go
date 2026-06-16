package scheduler_agent

import (
	"context"
	"fmt"
	"strings"
)

// Booker creates a meeting on behalf of the authenticated Telegram user. The
// implementation resolves org + organizer from the user's account — the agent
// never supplies identity. Book returns a short user-facing confirmation line.
type Booker interface {
	Book(ctx context.Context, telegramID int64, b PendingBooking) (string, error)
}

// describeBooking renders the confirm-card body for a proposed meeting (Russian,
// cozy tone to match the bot).
func describeBooking(b PendingBooking) string {
	var sb strings.Builder
	title := b.Type
	if b.Dept != "" {
		title = b.Dept + " · " + b.Type
	}
	fmt.Fprintf(&sb, "Создать встречу?\n\n📌 %s\n📅 %s, %s–%s\n👥 %s",
		title, b.Date, b.Start, b.End, strings.Join(b.Emails, ", "))
	if b.Desc != "" {
		fmt.Fprintf(&sb, "\n📝 %s", b.Desc)
	}
	return sb.String()
}
