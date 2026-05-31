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

// EmployeeSeed is one directory row to sync. Email is pre-normalized to
// lower-case by the caller. Defined here (not in employeedir) so the repo stays
// free of the platform package and there is no import cycle. §9.4
type EmployeeSeed struct {
	FullName string
	Email    string
	Dept     string
}

// ListWorkspacesWithGoogle returns IDs of workspaces that have Google
// service-account credentials configured (google_sa_json_enc IS NOT NULL).
func (s *Store) ListWorkspacesWithGoogle(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM workspaces WHERE google_sa_json_enc IS NOT NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// SyncEmployees makes workspaceID's employees rows mirror seeds in one
// transaction: rows whose email is not in seeds are deleted; present rows are
// upserted (full_name, dept). has_telegram is left untouched on existing rows.
// Returns per-op counts. Empty seeds is a no-op (caller guards against an empty
// CSV; this is belt-and-suspenders so a stray empty call never wipes the table).
// §9.4
func (s *Store) SyncEmployees(ctx context.Context, workspaceID uuid.UUID, seeds []EmployeeSeed) (added, updated, deleted int, err error) {
	if len(seeds) == 0 {
		return 0, 0, 0, nil
	}
	emails := make([]string, 0, len(seeds))
	for _, sd := range seeds {
		emails = append(emails, sd.Email)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ct, err := tx.Exec(ctx, `DELETE FROM employees WHERE workspace_id = $1 AND email <> ALL($2)`, workspaceID, emails)
	if err != nil {
		return 0, 0, 0, err
	}
	deleted = int(ct.RowsAffected())

	for _, sd := range seeds {
		var inserted bool
		if err = tx.QueryRow(ctx, `
			INSERT INTO employees (workspace_id, full_name, email, dept)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (workspace_id, email) DO UPDATE
				SET full_name = EXCLUDED.full_name, dept = EXCLUDED.dept
			RETURNING (xmax = 0)`,
			workspaceID, sd.FullName, sd.Email, sd.Dept).Scan(&inserted); err != nil {
			return 0, 0, 0, err
		}
		if inserted {
			added++
		} else {
			updated++
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, 0, 0, err
	}
	return added, updated, deleted, nil
}
