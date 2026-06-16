package scheduler_agent

import "github.com/luckyrogue/lead-cat/internal/application"

// PendingBooking is a meeting the model has proposed and is awaiting the user's
// confirm tap. One-off only in Phase 2.
type PendingBooking struct {
	Dept   string   `json:"dept,omitempty"`
	Type   string   `json:"type"`
	Date   string   `json:"date"`  // YYYY-MM-DD
	Start  string   `json:"start"` // HH:MM
	End    string   `json:"end"`   // HH:MM
	Emails []string `json:"emails"`
	Desc   string   `json:"desc,omitempty"`
}

type State struct {
	History []application.AgentMessage `json:"history,omitempty"`
	Pending *PendingBooking            `json:"pending,omitempty"`
}

type Button struct {
	Text string
	Data string
}

type Reply struct {
	Text     string
	Keyboard [][]Button
}
