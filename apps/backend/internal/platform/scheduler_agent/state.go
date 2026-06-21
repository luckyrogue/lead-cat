package scheduler_agent

import "github.com/luckyrogue/lead-cat/internal/application"

type PendingBooking struct {
	Dept   string   `json:"dept,omitempty"`
	Type   string   `json:"type"`
	Date   string   `json:"date"`
	Start  string   `json:"start"`
	End    string   `json:"end"`
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
	Edit     bool
}
