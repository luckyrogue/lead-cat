package postgres

import (
	"context"

	"github.com/google/uuid"
)

func (s *Store) ListEmployees(ctx context.Context, workspaceID uuid.UUID) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, full_name, email, dept, has_telegram
		FROM employees WHERE workspace_id = $1 ORDER BY full_name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SearchEmployeesGlobal finds directory entries across all workspaces whose name
// or email contains query (case-insensitive), capped at 20.
func (s *Store) SearchEmployeesGlobal(ctx context.Context, query string) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, workspace_id, full_name, email, dept, has_telegram
		FROM employees
		WHERE full_name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%'
		ORDER BY full_name LIMIT 20`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.WorkspaceID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateEmployee(ctx context.Context, workspaceID uuid.UUID, fullName, email, dept string, hasTelegram bool) (Employee, error) {
	var e Employee
	err := s.pool.QueryRow(ctx, `
		INSERT INTO employees (workspace_id, full_name, email, dept, has_telegram)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, workspace_id, full_name, email, dept, has_telegram`,
		workspaceID, fullName, email, dept, hasTelegram).
		Scan(&e.ID, &e.WorkspaceID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram)
	return e, err
}
