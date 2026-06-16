package scheduler_agent

import "github.com/luckyrogue/lead-cat/internal/application"

type State struct {
	History []application.AgentMessage `json:"history,omitempty"`
}

type Button struct {
	Text string
	Data string
}

type Reply struct {
	Text     string
	Keyboard [][]Button
}
