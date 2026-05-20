package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s *Store) ListScenarios(ctx context.Context, workspaceID uuid.UUID) ([]Scenario, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, name, enabled, definition
		FROM scenarios WHERE workspace_id = $1 ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scenario
	for rows.Next() {
		var sc Scenario
		if err := rows.Scan(&sc.ID, &sc.WorkspaceID, &sc.Name, &sc.Enabled, &sc.Definition); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Store) GetScenario(ctx context.Context, id uuid.UUID) (Scenario, error) {
	var sc Scenario
	err := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, name, enabled, definition FROM scenarios WHERE id = $1`, id).
		Scan(&sc.ID, &sc.WorkspaceID, &sc.Name, &sc.Enabled, &sc.Definition)
	return sc, err
}

func (s *Store) CreateScenario(ctx context.Context, workspaceID uuid.UUID, name string, def json.RawMessage) (Scenario, error) {
	var sc Scenario
	err := s.pool.QueryRow(ctx, `
		INSERT INTO scenarios (workspace_id, name, definition) VALUES ($1, $2, $3)
		RETURNING id, workspace_id, name, enabled, definition`,
		workspaceID, name, def).Scan(&sc.ID, &sc.WorkspaceID, &sc.Name, &sc.Enabled, &sc.Definition)
	return sc, err
}

func (s *Store) UpdateScenario(ctx context.Context, id uuid.UUID, name string, enabled bool, def json.RawMessage) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE scenarios SET name = $2, enabled = $3, definition = $4, updated_at = now()
		WHERE id = $1`, id, name, enabled, def)
	return err
}

func (s *Store) DeleteScenario(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM scenarios WHERE id = $1`, id)
	return err
}

func (s *Store) ListEnabledScenarios(ctx context.Context) ([]Scenario, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.workspace_id, s.name, s.enabled, s.definition
		FROM scenarios s
		JOIN workspaces w ON w.id = s.workspace_id
		WHERE s.enabled AND w.notify_chat_id IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scenario
	for rows.Next() {
		var sc Scenario
		if err := rows.Scan(&sc.ID, &sc.WorkspaceID, &sc.Name, &sc.Enabled, &sc.Definition); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Store) CreateRun(ctx context.Context, scenarioID, workspaceID uuid.UUID, trigger string) (ScenarioRun, error) {
	var r ScenarioRun
	err := s.pool.QueryRow(ctx, `
		INSERT INTO scenario_runs (scenario_id, workspace_id, status, trigger)
		VALUES ($1, $2, 'running', $3)
		RETURNING id, scenario_id, workspace_id, status, trigger, error, started_at, finished_at`,
		scenarioID, workspaceID, trigger).Scan(
		&r.ID, &r.ScenarioID, &r.WorkspaceID, &r.Status, &r.Trigger, &r.Error, &r.StartedAt, &r.FinishedAt)
	return r, err
}

func (s *Store) FinishRun(ctx context.Context, runID uuid.UUID, status, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE scenario_runs SET status = $2, error = $3, finished_at = now() WHERE id = $1`,
		runID, status, errMsg)
	return err
}

func (s *Store) AddRunStep(ctx context.Context, runID uuid.UUID, nodeID, nodeType, status string, output json.RawMessage, durMs int, errMsg string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO scenario_run_steps (run_id, node_id, node_type, status, output, duration_ms, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		runID, nodeID, nodeType, status, output, durMs, errMsg)
	return err
}

func (s *Store) ListRuns(ctx context.Context, scenarioID uuid.UUID, limit int) ([]ScenarioRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, scenario_id, workspace_id, status, trigger, error, started_at, finished_at
		FROM scenario_runs WHERE scenario_id = $1 ORDER BY started_at DESC LIMIT $2`, scenarioID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScenarioRun
	for rows.Next() {
		var r ScenarioRun
		if err := rows.Scan(&r.ID, &r.ScenarioID, &r.WorkspaceID, &r.Status, &r.Trigger, &r.Error, &r.StartedAt, &r.FinishedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
