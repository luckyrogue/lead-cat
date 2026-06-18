package model

import "time"

type CalendarConnection struct {
	Email        string
	Provider     string
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
	Scopes       string
	ConnectedAt  time.Time
	UpdatedAt    time.Time
}
