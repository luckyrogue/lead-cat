package handlers

import "time"

func almatyLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Almaty")
	if err != nil {
		return time.FixedZone("Asia/Almaty", 5*3600)
	}
	return loc
}

// resolveLoc loads tz, falling back to Asia/Almaty when tz is empty or invalid.
func resolveLoc(tz string) *time.Location {
	if tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return almatyLoc()
}
