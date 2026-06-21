package postgres

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateSurveyResponse(ctx context.Context, r model.SurveyResponse) (model.SurveyResponse, error) {
	answers, err := json.Marshal([]model.Answer{})
	if err != nil {
		return model.SurveyResponse{}, err
	}
	err = s.pool.QueryRow(ctx,
		`INSERT INTO survey_responses
			(survey_id, organization_id, booking_event_type_id, token, booker_email, booker_name, decline_reason, status, answers)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'sent',$8)
		 RETURNING id, created_at`,
		r.SurveyID, r.OrganizationID, r.BookingEventTypeID, r.Token, r.BookerEmail, r.BookerName, r.DeclineReason, answers).
		Scan(&r.ID, &r.CreatedAt)
	r.Status = "sent"
	return r, err
}

func scanResponse(row rowScanner) (model.SurveyResponse, error) {
	var r model.SurveyResponse
	var raw []byte
	if err := row.Scan(&r.ID, &r.SurveyID, &r.OrganizationID, &r.BookingEventTypeID, &r.Token,
		&r.BookerEmail, &r.BookerName, &r.DeclineReason, &r.Status, &raw, &r.CreatedAt, &r.CompletedAt); err != nil {
		return model.SurveyResponse{}, err
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &r.Answers)
	}
	return r, nil
}

const responseCols = `id, survey_id, organization_id, booking_event_type_id, token,
	booker_email, booker_name, decline_reason, status, answers, created_at, completed_at`

func (s *Store) GetSurveyResponseByToken(ctx context.Context, token string) (model.SurveyResponse, error) {
	return scanResponse(s.pool.QueryRow(ctx, `SELECT `+responseCols+` FROM survey_responses WHERE token=$1`, token))
}

func (s *Store) CompleteSurveyResponse(ctx context.Context, id uuid.UUID, answers []model.Answer) error {
	raw, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx,
		`UPDATE survey_responses SET status='completed', answers=$2, completed_at=now() WHERE id=$1`,
		id, raw)
	return err
}

func (s *Store) ListSurveyResponses(ctx context.Context, surveyID uuid.UUID, f model.ResponseFilter) ([]model.SurveyResponse, error) {
	q := `SELECT ` + responseCols + ` FROM survey_responses WHERE survey_id=$1`
	args := []any{surveyID}
	add := func(cond string, v any) {
		args = append(args, v)
		q += cond + "$" + strconv.Itoa(len(args))
	}
	if f.Status != "" {
		add(" AND status=", f.Status)
	}
	if f.Reason != "" {
		add(" AND decline_reason=", f.Reason)
	}
	if f.From != nil {
		add(" AND created_at>=", *f.From)
	}
	if f.To != nil {
		add(" AND created_at<", *f.To)
	}
	q += " ORDER BY created_at DESC"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SurveyResponse{}
	for rows.Next() {
		r, err := scanResponse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
