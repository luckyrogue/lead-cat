package tz

import "time"

// Default is the fallback IANA timezone used when none is configured.
const Default = "Asia/Almaty"

// Load returns the location for name, or the default zone when name is empty
// or unknown. It never fails: if even the default cannot be loaded it returns
// a fixed +05:00 zone.
func Load(name string) *time.Location {
	if name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	if loc, err := time.LoadLocation(Default); err == nil {
		return loc
	}
	return time.FixedZone("Almaty", 5*3600)
}
