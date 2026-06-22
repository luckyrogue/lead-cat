package query

import (
	"context"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

type CalendarConnectionView struct {
	Provider  string `json:"provider"`
	Connected bool   `json:"connected"`
	Email     string `json:"email"`
	Scopes    string `json:"scopes"`
}

type calendarStore interface {
	ListCalendarConnections(ctx context.Context, email string) ([]model.CalendarConnection, error)
}

type Calendar struct {
	Store calendarStore
}

func NewCalendar(store calendarStore) *Calendar {
	return &Calendar{Store: store}
}

func (q *Calendar) ListCalendarConnections(ctx context.Context, email string) ([]CalendarConnectionView, error) {
	rows, err := q.Store.ListCalendarConnections(ctx, email)
	if err != nil {
		return nil, err
	}
	out := []CalendarConnectionView{}
	for _, r := range rows {
		out = append(out, CalendarConnectionView{Provider: r.Provider, Connected: true, Email: r.Email, Scopes: r.Scopes})
	}
	return out, nil
}
