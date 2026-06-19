package model

import (
	"database/sql"
	"errors"
)

var ErrMeetingNotEditable = errors.New("meeting not found or not editable")

var ErrInviteEmailMismatch = errors.New("invite email does not match user")

var ErrForbidden = errors.New("forbidden")

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
