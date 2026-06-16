package scheduler_agent

import (
	"testing"
)

func TestToolSpecs_Shape(t *testing.T) {
	specs := ToolSpecs()
	for _, want := range []string{"search_people", "find_free_slots", "check_conflicts", "propose_meeting"} {
		found := false
		for _, spec := range specs {
			if spec.Name == want {
				found = true
				if spec.Description == "" {
					t.Errorf("tool %q has empty description", want)
				}
				if spec.InputSchema == nil {
					t.Errorf("tool %q has nil InputSchema", want)
				}
				break
			}
		}
		if !found {
			t.Errorf("tool %q not found in ToolSpecs()", want)
		}
	}
}
