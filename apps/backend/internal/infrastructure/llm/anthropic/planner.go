package anthropic

import (
	"context"
	"fmt"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/luckyrogue/lead-cat/internal/application"
)

// Planner implements application.Planner using the Anthropic Messages API.
type Planner struct {
	client sdk.Client
	model  sdk.Model
}

// New builds a Planner. apiKey may be empty to fall back to ANTHROPIC_API_KEY.
func New(apiKey string) *Planner {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &Planner{
		client: sdk.NewClient(opts...),
		model:  sdk.ModelClaudeOpus4_8,
	}
}

var _ application.Planner = (*Planner)(nil)

func (p *Planner) Plan(ctx context.Context, system string, history []application.AgentMessage, tools []application.AgentTool) (application.AgentTurn, error) {
	adaptive := sdk.ThinkingConfigAdaptiveParam{}
	resp, err := p.client.Messages.New(ctx, sdk.MessageNewParams{
		Model:     p.model,
		MaxTokens: 16000,
		System:    []sdk.TextBlockParam{{Text: system}},
		Thinking:  sdk.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		Messages:  toSDKMessages(history),
		Tools:     toSDKTools(tools),
	})
	if err != nil {
		return application.AgentTurn{}, fmt.Errorf("anthropic plan: %w", err)
	}
	if resp.StopReason == sdk.StopReasonRefusal {
		return application.AgentTurn{Text: "Извини, не могу с этим помочь 🐾"}, nil
	}
	return fromSDKResponse(resp), nil
}

func toSDKTools(tools []application.AgentTool) []sdk.ToolUnionParam {
	out := make([]sdk.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		req, _ := t.InputSchema["required"].([]string)
		tool := sdk.ToolParam{
			Name:        t.Name,
			Description: sdk.String(t.Description),
			InputSchema: sdk.ToolInputSchemaParam{
				Properties: t.InputSchema["properties"],
				Required:   req,
			},
		}
		out = append(out, sdk.ToolUnionParam{OfTool: &tool})
	}
	return out
}

func toSDKMessages(history []application.AgentMessage) []sdk.MessageParam {
	out := make([]sdk.MessageParam, 0, len(history))
	for _, m := range history {
		switch m.Role {
		case "assistant":
			var blocks []sdk.ContentBlockParamUnion
			// Thinking block must come first; the API rejects an assistant turn that
			// has a tool_use block without its preceding thinking block.
			if m.ThinkingSignature != "" {
				blocks = append(blocks, sdk.NewThinkingBlock(m.ThinkingSignature, m.Thinking))
			}
			if m.Text != "" {
				blocks = append(blocks, sdk.NewTextBlock(m.Text))
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, sdk.ContentBlockParamUnion{
					OfToolUse: &sdk.ToolUseBlockParam{
						ID:    tc.ID,
						Name:  tc.Name,
						Input: tc.Input,
					},
				})
			}
			out = append(out, sdk.NewAssistantMessage(blocks...))
		default: // "user"
			if len(m.ToolResults) > 0 {
				blocks := make([]sdk.ContentBlockParamUnion, 0, len(m.ToolResults))
				for _, tr := range m.ToolResults {
					blocks = append(blocks, sdk.NewToolResultBlock(tr.ID, tr.Content, tr.IsError))
				}
				out = append(out, sdk.NewUserMessage(blocks...))
				continue
			}
			out = append(out, sdk.NewUserMessage(sdk.NewTextBlock(m.Text)))
		}
	}
	return out
}

func fromSDKResponse(resp *sdk.Message) application.AgentTurn {
	var turn application.AgentTurn
	for _, block := range resp.Content {
		switch block.Type {
		case "thinking":
			tb := block.AsThinking()
			turn.Thinking = tb.Thinking
			turn.ThinkingSignature = tb.Signature
		case "text":
			turn.Text += block.AsText().Text
		case "tool_use":
			tb := block.AsToolUse()
			turn.ToolCalls = append(turn.ToolCalls, application.AgentToolCall{
				ID:    tb.ID,
				Name:  tb.Name,
				Input: tb.Input,
			})
		}
	}
	return turn
}
