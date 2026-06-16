package application

import (
	"context"
	"encoding/json"
)

// AgentTool is one tool the planner may call, described in provider-neutral form.
// InputSchema is a JSON Schema object (map form).
type AgentTool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// AgentToolCall is a model request to run a tool. Input is the raw JSON arguments.
type AgentToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// AgentToolResult is the outcome of running a tool, fed back to the model.
type AgentToolResult struct {
	ID      string
	Content string
	IsError bool
}

// AgentMessage is one entry in the running transcript. An assistant message may
// carry ToolCalls; the matching user message carries ToolResults.
type AgentMessage struct {
	Role        string            `json:"role"` // "user" | "assistant"
	Text        string            `json:"text,omitempty"`
	ToolCalls   []AgentToolCall   `json:"tool_calls,omitempty"`
	ToolResults []AgentToolResult `json:"tool_results,omitempty"`
}

// AgentTurn is one assistant turn returned by the planner. When ToolCalls is
// non-empty the caller must run them and call Plan again; otherwise Text is the
// final answer.
type AgentTurn struct {
	Text      string
	ToolCalls []AgentToolCall
}

// Planner performs one stateless model round-trip: given the system prompt, the
// full transcript, and the tool list, it returns the next assistant turn.
type Planner interface {
	Plan(ctx context.Context, system string, history []AgentMessage, tools []AgentTool) (AgentTurn, error)
}
