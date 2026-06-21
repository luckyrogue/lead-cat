package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type QuestionType string

const (
	QuestionSingle QuestionType = "single"
	QuestionMulti  QuestionType = "multi"
	QuestionRating QuestionType = "rating"
	QuestionText   QuestionType = "text"
)

var (
	ErrInvalidSurvey      = errors.New("invalid survey")
	ErrSurveyHasResponses = errors.New("survey has responses")
	ErrSurveyClosed       = errors.New("survey closed")
	ErrResponseCompleted  = errors.New("response already completed")
)

type Survey struct {
	ID             uuid.UUID        `json:"id"`
	OrganizationID uuid.UUID        `json:"organization_id"`
	Name           string           `json:"name"`
	IsActive       bool             `json:"is_active"`
	Questions      []SurveyQuestion `json:"questions"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type SurveyQuestion struct {
	ID         uuid.UUID    `json:"id"`
	SurveyID   uuid.UUID    `json:"survey_id"`
	OrderIndex int          `json:"order_index"`
	Prompt     string       `json:"prompt"`
	Type       QuestionType `json:"type"`
	Options    []string     `json:"options"`
	RatingMax  int          `json:"rating_max"`
	Required   bool         `json:"required"`
}

type Answer struct {
	QuestionID uuid.UUID    `json:"question_id"`
	Prompt     string       `json:"prompt"`
	Type       QuestionType `json:"type"`
	Value      any          `json:"value"`
}

type SurveyResponse struct {
	ID                 uuid.UUID  `json:"id"`
	SurveyID           uuid.UUID  `json:"survey_id"`
	OrganizationID     uuid.UUID  `json:"organization_id"`
	BookingEventTypeID *uuid.UUID `json:"booking_event_type_id"`
	Token              string     `json:"-"`
	BookerEmail        string     `json:"booker_email"`
	BookerName         string     `json:"booker_name"`
	DeclineReason      string     `json:"decline_reason"`
	Status             string     `json:"status"`
	Answers            []Answer   `json:"answers"`
	CreatedAt          time.Time  `json:"created_at"`
	CompletedAt        *time.Time `json:"completed_at"`
}

type ResponseFilter struct {
	Status string
	Reason string
	From   *time.Time
	To     *time.Time
}

func (q SurveyQuestion) validate() error {
	if q.Prompt == "" {
		return ErrInvalidSurvey
	}
	switch q.Type {
	case QuestionSingle, QuestionMulti:
		if len(q.Options) < 2 {
			return ErrInvalidSurvey
		}
	case QuestionRating:
		if q.RatingMax < 2 || q.RatingMax > 10 {
			return ErrInvalidSurvey
		}
	case QuestionText:
		// no extra fields required
	default:
		return ErrInvalidSurvey
	}
	return nil
}

func (s Survey) Validate() error {
	if s.Name == "" || len(s.Questions) == 0 {
		return ErrInvalidSurvey
	}
	for _, q := range s.Questions {
		if err := q.validate(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateAnswers(questions []SurveyQuestion, answers []Answer) ([]Answer, error) {
	given := map[uuid.UUID]Answer{}
	for _, a := range answers {
		given[a.QuestionID] = a
	}
	out := make([]Answer, 0, len(answers))
	for _, q := range questions {
		a, ok := given[q.ID]
		if !ok || isEmptyAnswer(a.Value) {
			if q.Required {
				return nil, ErrInvalidSurvey
			}
			continue
		}
		if err := validateAnswerValue(q, a.Value); err != nil {
			return nil, err
		}
		out = append(out, Answer{QuestionID: q.ID, Prompt: q.Prompt, Type: q.Type, Value: a.Value})
	}
	return out, nil
}

func isEmptyAnswer(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return t == ""
	case []string:
		return len(t) == 0
	case []any:
		return len(t) == 0
	}
	return false
}

func validateAnswerValue(q SurveyQuestion, v any) error {
	switch q.Type {
	case QuestionText:
		if _, ok := v.(string); !ok {
			return ErrInvalidSurvey
		}
	case QuestionSingle:
		s, ok := v.(string)
		if !ok || !contains(q.Options, s) {
			return ErrInvalidSurvey
		}
	case QuestionMulti:
		vals, err := toStringSlice(v)
		if err != nil {
			return err
		}
		for _, s := range vals {
			if !contains(q.Options, s) {
				return ErrInvalidSurvey
			}
		}
	case QuestionRating:
		n, ok := toInt(v)
		if !ok || n < 1 || n > q.RatingMax {
			return ErrInvalidSurvey
		}
	default:
		return ErrInvalidSurvey
	}
	return nil
}

func contains(opts []string, s string) bool {
	for _, o := range opts {
		if o == s {
			return true
		}
	}
	return false
}

func toStringSlice(v any) ([]string, error) {
	switch t := v.(type) {
	case []string:
		return t, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, ErrInvalidSurvey
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, ErrInvalidSurvey
}

func toInt(v any) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64: // JSON numbers decode to float64
		return int(t), true
	}
	return 0, false
}
