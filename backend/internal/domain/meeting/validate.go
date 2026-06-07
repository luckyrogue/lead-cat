package meeting

import "fmt"

// Validate checks required fields, time ordering, and recurrence.
func (in Input) Validate() error {
	if in.Dept == "" {
		return fmt.Errorf("dept required")
	}
	if in.Type == "" {
		return fmt.Errorf("type required")
	}
	if in.Host == "" {
		return fmt.Errorf("host required")
	}
	if in.StartsAt.IsZero() || in.EndsAt.IsZero() {
		return fmt.Errorf("start and end required")
	}
	if !in.EndsAt.After(in.StartsAt) {
		return fmt.Errorf("end must be after start")
	}
	if !in.Recurrence.Valid() {
		return fmt.Errorf("unknown recurrence: %q", in.Recurrence)
	}
	if in.Recurrence == Custom && len(in.RecurrenceDays) == 0 {
		return ErrRecurrenceDays
	}
	for _, d := range in.RecurrenceDays {
		if d < 1 || d > 7 {
			return fmt.Errorf("recurrence_days values must be 1..7, got %d", d)
		}
	}
	return nil
}
