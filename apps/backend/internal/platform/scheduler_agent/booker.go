package scheduler_agent

import (
	"context"
	"fmt"
	"strings"
)

type Booker interface {
	Book(ctx context.Context, telegramID int64, b PendingBooking) (string, error)
}

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
