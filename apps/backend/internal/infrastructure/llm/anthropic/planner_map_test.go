package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
)

// TestToSDKMessages_ThinkingBlockPreserved verifies that an assistant message
// carrying thinking+signature round-trips through toSDKMessages as the FIRST
// content block (thinking must precede tool_use or the API rejects the turn).
func TestToSDKMessages_ThinkingBlockPreserved(t *testing.T) {
	history := []application.AgentMessage{
		{
			Role:              "assistant",
			Thinking:          "reasoning...",
			ThinkingSignature: "sig123",
			Text:              "ok",
			ToolCalls: []application.AgentToolCall{
				{ID: "call1", Name: "search_people", Input: json.RawMessage(`{"query":"alice"}`)},
			},
		},
		{
			Role: "user",
			ToolResults: []application.AgentToolResult{
				{ID: "call1", Content: "- Alice <alice@example.com>"},
			},
		},
	}

	params := toSDKMessages(history)
	if len(params) != 2 {
		t.Fatalf("expected 2 MessageParams, got %d", len(params))
	}

	// Marshal the assistant MessageParam and inspect the JSON.
	assistantJSON, err := json.Marshal(params[0])
	if err != nil {
		t.Fatalf("json.Marshal assistant param: %v", err)
	}

	// Parse into a generic structure to check ordering.
	var msg struct {
		Role    string `json:"role"`
		Content []struct {
			Type      string `json:"type"`
			Signature string `json:"signature"`
			Thinking  string `json:"thinking"`
		} `json:"content"`
	}
	if err := json.Unmarshal(assistantJSON, &msg); err != nil {
		t.Fatalf("json.Unmarshal assistant param: %v", err)
	}

	if msg.Role != "assistant" {
		t.Fatalf("expected role=assistant, got %q", msg.Role)
	}
	// We expect at least 3 blocks: thinking, text, tool_use.
	if len(msg.Content) < 3 {
		t.Fatalf("expected ≥3 content blocks, got %d: %s", len(msg.Content), assistantJSON)
	}
	first := msg.Content[0]
	if first.Type != "thinking" {
		t.Errorf("first block type: want %q, got %q (full JSON: %s)", "thinking", first.Type, assistantJSON)
	}
	if first.Signature != "sig123" {
		t.Errorf("first block signature: want %q, got %q", "sig123", first.Signature)
	}
	if first.Thinking != "reasoning..." {
		t.Errorf("first block thinking: want %q, got %q", "reasoning...", first.Thinking)
	}
}

// TestToSDKTools_RequiredPropagated verifies that the required array from the
// neutral AgentTool.InputSchema is forwarded to the SDK param and serialised.
func TestToSDKTools_RequiredPropagated(t *testing.T) {
	tools := []application.AgentTool{
		{
			Name:        "x",
			Description: "d",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"a": map[string]any{"type": "string"},
				},
				"required": []string{"a"},
			},
		},
	}

	params := toSDKTools(tools)
	if len(params) != 1 {
		t.Fatalf("expected 1 tool param, got %d", len(params))
	}

	toolJSON, err := json.Marshal(params[0])
	if err != nil {
		t.Fatalf("json.Marshal tool param: %v", err)
	}

	// The SDK nests the schema under "input_schema".
	var raw struct {
		InputSchema struct {
			Required []string `json:"required"`
		} `json:"input_schema"`
	}
	if err := json.Unmarshal(toolJSON, &raw); err != nil {
		t.Fatalf("json.Unmarshal tool param: %v", err)
	}

	req := raw.InputSchema.Required
	if len(req) != 1 || req[0] != "a" {
		t.Errorf("required: want [\"a\"], got %v (full JSON: %s)", req, toolJSON)
	}
}
