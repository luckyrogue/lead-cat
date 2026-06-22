package handlers

import (
	"time"

	"github.com/luckyrogue/lead-cat/internal/platform/tz"
)

func resolveLoc(name string) *time.Location {
	return tz.Load(name)
}
