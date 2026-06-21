package application

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func TestResponsesCSV(t *testing.T) {
	q1 := model.SurveyQuestion{ID: uuid.New(), Prompt: "Why?", Type: model.QuestionText}
	q2 := model.SurveyQuestion{ID: uuid.New(), Prompt: "Pick", Type: model.QuestionMulti, Options: []string{"a", "b"}}
	sv := model.Survey{Questions: []model.SurveyQuestion{q1, q2}}
	resp := model.SurveyResponse{
		BookerName: "Bo", BookerEmail: "a@b.c", DeclineReason: "slot_taken", Status: "completed",
		Answers: []model.Answer{
			{QuestionID: q1.ID, Value: "no time"},
			{QuestionID: q2.ID, Value: []any{"a", "b"}},
		},
	}
	out := string(ResponsesCSV(sv, []model.SurveyResponse{resp}))
	if !strings.Contains(out, "Why?") || !strings.Contains(out, "Pick") {
		t.Fatalf("expected question headers, got:\n%s", out)
	}
	if !strings.Contains(out, "no time") || !strings.Contains(out, "a; b") {
		t.Fatalf("expected answer values incl multi join, got:\n%s", out)
	}
	if !strings.Contains(out, "a@b.c") || !strings.Contains(out, "slot_taken") {
		t.Fatalf("expected meta columns, got:\n%s", out)
	}
}
