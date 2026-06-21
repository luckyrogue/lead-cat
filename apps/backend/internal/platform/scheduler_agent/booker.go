package scheduler_agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/luckyrogue/lead-cat/internal/platform/boti18n"
)

type Booker interface {
	Book(ctx context.Context, telegramID int64, b PendingBooking, lang string) (string, error)
}

func describeBooking(b PendingBooking, lang string) string {
	var sb strings.Builder
	title := b.Type
	if b.Dept != "" {
		title = b.Dept + " · " + b.Type
	}
	fmt.Fprintf(&sb, "%s\n\n📌 %s\n📅 %s, %s–%s\n👥 %s",
		boti18n.T(lang, "agent.card_q"), title, b.Date, b.Start, b.End, strings.Join(b.Emails, ", "))
	if b.Desc != "" {
		fmt.Fprintf(&sb, "\n📝 %s", b.Desc)
	}
	return sb.String()
}
