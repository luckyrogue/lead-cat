package postgres

import "github.com/luckyrogue/lead-cat/internal/application/model"

var ErrMeetingNotEditable = model.ErrMeetingNotEditable

func IsNotFound(err error) bool {
	return model.IsNotFound(err)
}
