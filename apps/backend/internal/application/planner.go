package application

import (
	"context"
	"encoding/json"
)

type AgentTool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

type AgentToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

type AgentToolResult struct {
	ID      string
	Content string
	IsError bool
}

type AgentMessage struct {
	Role               string            `json:"role"` 
	Text               string            `json:"text,omitempty"`
	Thinking           string            `json:"thinking,omitempty"`
	ThinkingSignature  string            `json:"thinking_signature,omitempty"`
	ToolCalls          []AgentToolCall   `json:"tool_calls,omitempty"`
	ToolResults        []AgentToolResult `json:"tool_results,omitempty"`
}

type AgentTurn struct {
	Text              string
	Thinking          string
	ThinkingSignature string
	ToolCalls         []AgentToolCall
}

type Planner interface {
	Plan(ctx context.Context, system string, history []AgentMessage, tools []AgentTool) (AgentTurn, error)
}
