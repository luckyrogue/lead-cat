package meeting

import "time"

type Recurrence string

const (
	Once    Recurrence = "once"
	Daily   Recurrence = "daily"
	Weekly  Recurrence = "weekly"
	Custom  Recurrence = "custom"
	Monthly Recurrence = "monthly"
)

var recurrenceLabels = map[Recurrence]string{
	Once:    "",
	Daily:   "Ежедневно",
	Weekly:  "Еженедельно",
	Custom:  "По выбранным дням",
	Monthly: "Ежемесячно",
}

func (r Recurrence) Valid() bool {
	_, ok := recurrenceLabels[r]
	return ok
}

func (r Recurrence) Label() string { return recurrenceLabels[r] }

type Input struct {
	Dept           string
	Type           string
	Host           string
	StartsAt       time.Time
	EndsAt         time.Time
	Recurrence     Recurrence
	RecurrenceDays []int
	Description    string
}
