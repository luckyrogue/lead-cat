package postgres

import (
	"context"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func (s *Store) CreateSurvey(ctx context.Context, sv model.Survey) (model.Survey, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Survey{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.QueryRow(ctx,
		`INSERT INTO surveys (organization_id, name, is_active) VALUES ($1,$2,$3)
		 RETURNING id, created_at, updated_at`,
		sv.OrganizationID, sv.Name, sv.IsActive).
		Scan(&sv.ID, &sv.CreatedAt, &sv.UpdatedAt); err != nil {
		return model.Survey{}, err
	}
	for i := range sv.Questions {
		q := &sv.Questions[i]
		q.OrderIndex = i
		if err := tx.QueryRow(ctx,
			`INSERT INTO survey_questions (survey_id, order_index, prompt, type, options, rating_max, required)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
			sv.ID, q.OrderIndex, q.Prompt, string(q.Type), q.Options, q.RatingMax, q.Required).
			Scan(&q.ID); err != nil {
			return model.Survey{}, err
		}
		q.SurveyID = sv.ID
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Survey{}, err
	}
	return sv, nil
}

func (s *Store) UpdateSurvey(ctx context.Context, sv model.Survey) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE surveys SET name=$2, is_active=$3, updated_at=now() WHERE id=$1`,
		sv.ID, sv.Name, sv.IsActive); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM survey_questions WHERE survey_id=$1`, sv.ID); err != nil {
		return err
	}
	for i, q := range sv.Questions {
		if _, err := tx.Exec(ctx,
			`INSERT INTO survey_questions (survey_id, order_index, prompt, type, options, rating_max, required)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			sv.ID, i, q.Prompt, string(q.Type), q.Options, q.RatingMax, q.Required); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) GetSurvey(ctx context.Context, id uuid.UUID) (model.Survey, error) {
	var sv model.Survey
	if err := s.pool.QueryRow(ctx,
		`SELECT id, organization_id, name, is_active, created_at, updated_at FROM surveys WHERE id=$1`, id).
		Scan(&sv.ID, &sv.OrganizationID, &sv.Name, &sv.IsActive, &sv.CreatedAt, &sv.UpdatedAt); err != nil {
		return model.Survey{}, err
	}
	qs, err := s.questionsFor(ctx, id)
	if err != nil {
		return model.Survey{}, err
	}
	sv.Questions = qs
	return sv, nil
}

func (s *Store) questionsFor(ctx context.Context, surveyID uuid.UUID) ([]model.SurveyQuestion, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, survey_id, order_index, prompt, type, options, rating_max, required
		 FROM survey_questions WHERE survey_id=$1 ORDER BY order_index`, surveyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SurveyQuestion{}
	for rows.Next() {
		var q model.SurveyQuestion
		var typ string
		if err := rows.Scan(&q.ID, &q.SurveyID, &q.OrderIndex, &q.Prompt, &typ, &q.Options, &q.RatingMax, &q.Required); err != nil {
			return nil, err
		}
		q.Type = model.QuestionType(typ)
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) ListSurveys(ctx context.Context, orgID uuid.UUID) ([]model.Survey, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, organization_id, name, is_active, created_at, updated_at
		 FROM surveys WHERE organization_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Survey{}
	for rows.Next() {
		var sv model.Survey
		if err := rows.Scan(&sv.ID, &sv.OrganizationID, &sv.Name, &sv.IsActive, &sv.CreatedAt, &sv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, sv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		qs, err := s.questionsFor(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Questions = qs
	}
	return out, nil
}

func (s *Store) DeleteSurvey(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM surveys WHERE id=$1`, id)
	return err
}

func (s *Store) CountResponses(ctx context.Context, surveyID uuid.UUID) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM survey_responses WHERE survey_id=$1`, surveyID).Scan(&n)
	return n, err
}
