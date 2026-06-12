package model

import (
	"database/sql"
	"errors"
)

var ErrMeetingNotEditable = errors.New("meeting not found or not editable")

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
