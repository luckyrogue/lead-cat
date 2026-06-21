package model

import (
	"time"

	"github.com/google/uuid"
)

type BookingEventType struct {
	ID               uuid.UUID  `json:"id"`
	HostUserID       uuid.UUID  `json:"host_user_id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
	Slug             string     `json:"slug"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	DurationMins     int        `json:"duration_mins"`
	Active           bool       `json:"active"`
	Timezone         string     `json:"timezone"`
	AvailWeekdays    []int      `json:"avail_weekdays"`
	AvailStartMinute int        `json:"avail_start_minute"`
	AvailEndMinute   int        `json:"avail_end_minute"`
	SurveyID         *uuid.UUID `json:"survey_id"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
