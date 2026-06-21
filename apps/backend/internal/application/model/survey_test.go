package model

import (
	"testing"

	"github.com/google/uuid"
)

func textQ() SurveyQuestion {
	return SurveyQuestion{ID: uuid.New(), OrderIndex: 0, Prompt: "Why?", Type: QuestionText, Required: true}
}
func singleQ() SurveyQuestion {
	return SurveyQuestion{ID: uuid.New(), OrderIndex: 1, Prompt: "Pick", Type: QuestionSingle, Options: []string{"a", "b"}, Required: true}
}
func ratingQ() SurveyQuestion {
	return SurveyQuestion{ID: uuid.New(), OrderIndex: 2, Prompt: "Rate", Type: QuestionRating, RatingMax: 5, Required: false}
}

func TestSurveyValidate(t *testing.T) {
	ok := Survey{Name: "S", Questions: []SurveyQuestion{textQ(), singleQ(), ratingQ()}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if err := (Survey{Name: "", Questions: []SurveyQuestion{textQ()}}).Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := (Survey{Name: "S"}).Validate(); err == nil {
		t.Fatal("expected error for zero questions")
	}
	bad := Survey{Name: "S", Questions: []SurveyQuestion{{Prompt: "p", Type: QuestionSingle, Options: []string{"only"}}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected error: single needs >=2 options")
	}
	badRating := Survey{Name: "S", Questions: []SurveyQuestion{{Prompt: "p", Type: QuestionRating, RatingMax: 1}}}
	if err := badRating.Validate(); err == nil {
		t.Fatal("expected error: rating_max must be 2..10")
	}
}

func TestValidateAnswers(t *testing.T) {
	tq, sq, rq := textQ(), singleQ(), ratingQ()
	qs := []SurveyQuestion{tq, sq, rq}

	got, err := ValidateAnswers(qs, []Answer{
		{QuestionID: tq.ID, Value: "because"},
		{QuestionID: sq.ID, Value: "a"},
	})
	if err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	if len(got) != 2 || got[0].Prompt != "Why?" || got[0].Type != QuestionText {
		t.Fatalf("expected snapshotted answers, got %+v", got)
	}

	if _, err := ValidateAnswers(qs, []Answer{{QuestionID: sq.ID, Value: "a"}}); err == nil {
		t.Fatal("expected error: required text unanswered")
	}
	if _, err := ValidateAnswers(qs, []Answer{{QuestionID: tq.ID, Value: "x"}, {QuestionID: sq.ID, Value: "z"}}); err == nil {
		t.Fatal("expected error: option not allowed")
	}
	if _, err := ValidateAnswers(qs, []Answer{{QuestionID: tq.ID, Value: "x"}, {QuestionID: sq.ID, Value: "a"}, {QuestionID: rq.ID, Value: 9}}); err == nil {
		t.Fatal("expected error: rating out of range")
	}
}
