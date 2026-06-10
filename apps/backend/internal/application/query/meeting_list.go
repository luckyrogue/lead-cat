package query

import (
	"context"
	"time"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

// Meetings groups read-side meeting operations (CQRS entry points).
type Meetings struct {
	App meetingListApp
}

type meetingListApp interface {
	EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error)
}

func NewMeetings(app meetingListApp) *Meetings {
	return &Meetings{App: app}
}

func (m *Meetings) Schedule(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error) {
	return m.App.EmployeeSchedule(ctx, email, from, to)
}
