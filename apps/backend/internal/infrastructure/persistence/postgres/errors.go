package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// IsNotFound reports whether err is a "no rows" result from a query.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
