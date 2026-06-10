package model

import (
	"database/sql"
	"errors"
)

// ErrMeetingNotEditable means the meeting does not exist in the workspace or is
// not in the 'scheduled' state (e.g. already cancelled).
var ErrMeetingNotEditable = errors.New("meeting not found or not editable")

// IsNotFound reports whether err is a "no rows" result from a query. pgx.ErrNoRows
// proxies database/sql.ErrNoRows, so matching the stdlib sentinel keeps this helper
// free of persistence-driver imports.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
