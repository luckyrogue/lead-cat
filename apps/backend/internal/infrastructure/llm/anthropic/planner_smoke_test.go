package anthropic

import (
	"context"
	"os"
	"testing"

	"github.com/luckyrogue/lead-cat/internal/application"
)

// TestPlanner_Smoke hits the live API; skipped unless ANTHROPIC_API_KEY is set.
func TestPlanner_Smoke(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set; skipping live smoke test")
	}
	p := New("")
	turn, err := p.Plan(context.Background(), "You are a helpful assistant. Reply in one short word.",
		[]application.AgentMessage{{Role: "user", Text: "Say hi."}}, nil)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if turn.Text == "" && len(turn.ToolCalls) == 0 {
		t.Fatal("empty turn from live API")
	}
}
