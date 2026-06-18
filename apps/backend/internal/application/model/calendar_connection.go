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

type CalendarOAuthState struct {
	State     string
	Email     string
	Provider  string
	Verifier  string
	ExpiresAt time.Time
}
