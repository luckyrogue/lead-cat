package postgres

import "github.com/luckyrogue/lead-cat/internal/application/model"

// ErrMeetingNotEditable and IsNotFound are defined in application/model so the
// application and delivery layers can reference them without importing this
// persistence package. Re-exported here to keep the repository code terse.
var ErrMeetingNotEditable = model.ErrMeetingNotEditable

// IsNotFound reports whether err is a "no rows" result from a query.
func IsNotFound(err error) bool {
	return model.IsNotFound(err)
}
