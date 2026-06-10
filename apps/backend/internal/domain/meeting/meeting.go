// Package meeting holds the meeting domain: types, recurrence rules,
// validation, and the meeting-name standard. No persistence or transport here.
package meeting

import "time"

// Recurrence is how often a meeting repeats.
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

// Valid reports whether r is a known recurrence value.
func (r Recurrence) Valid() bool {
	_, ok := recurrenceLabels[r]
	return ok
}

// Label is the human-readable Russian label used in the meeting name.
func (r Recurrence) Label() string { return recurrenceLabels[r] }

// Input is the validated, parsed payload for creating a meeting.
type Input struct {
	Dept           string
	Type           string
	Host           string
	StartsAt       time.Time
	EndsAt         time.Time
	Recurrence     Recurrence
	RecurrenceDays []int // 1=Mon..7=Sun; required when Recurrence == Custom.
	Description    string
}
