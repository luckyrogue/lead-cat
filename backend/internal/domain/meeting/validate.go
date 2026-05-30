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
	return nil
}
