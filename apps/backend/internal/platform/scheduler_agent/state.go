package scheduler_agent

import "github.com/luckyrogue/lead-cat/internal/application"

// State is the per-user conversation transcript, persisted in Redis between turns.
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
