package postgres

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s *Store) ListEmployees(ctx context.Context, organizationID uuid.UUID) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, full_name, email, dept, has_telegram
		FROM employees WHERE organization_id = $1 ORDER BY full_name`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) SearchEmployeesGlobal(ctx context.Context, query string) ([]Employee, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, full_name, email, dept, has_telegram
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
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateEmployee(ctx context.Context, organizationID uuid.UUID, fullName, email, dept string, hasTelegram bool) (Employee, error) {
	var e Employee
	err := s.pool.QueryRow(ctx, `
		INSERT INTO employees (organization_id, full_name, email, dept, has_telegram)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, organization_id, full_name, email, dept, has_telegram`,
		organizationID, fullName, email, dept, hasTelegram).
		Scan(&e.ID, &e.OrganizationID, &e.FullName, &e.Email, &e.Dept, &e.HasTelegram)
	return e, err
}

type EmployeeSeed struct {
	FullName string
	Email    string
	Dept     string
}

func (s *Store) ListOrganizationsWithGoogle(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM organizations WHERE google_sa_json_enc IS NOT NULL ORDER BY id`)
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

func (s *Store) SyncEmployees(ctx context.Context, organizationID uuid.UUID, seeds []EmployeeSeed) (added, updated, deleted int, err error) {
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

	ct, err := tx.Exec(ctx, `DELETE FROM employees WHERE organization_id = $1 AND email <> ALL($2)`, organizationID, emails)
	if err != nil {
		return 0, 0, 0, err
	}
	deleted = int(ct.RowsAffected())

	for _, sd := range seeds {
		var inserted bool
		if err = tx.QueryRow(ctx, `
			INSERT INTO employees (organization_id, full_name, email, dept)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (organization_id, email) DO UPDATE
				SET full_name = EXCLUDED.full_name, dept = EXCLUDED.dept
			RETURNING (xmax = 0)`,
			organizationID, sd.FullName, sd.Email, sd.Dept).Scan(&inserted); err != nil {
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

func (s *Store) FilterKnownEmployeeEmails(ctx context.Context, emails []string) ([]string, error) {
	if len(emails) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool, len(emails))
	unique := make([]string, 0, len(emails))
	for _, e := range emails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		unique = append(unique, e)
	}
	if len(unique) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT email FROM employees WHERE lower(email) = ANY($1)`, unique)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	knownLower := make(map[string]string, len(unique))
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		knownLower[strings.ToLower(email)] = email
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(unique))
	for _, e := range unique {
		if canon, ok := knownLower[e]; ok {
			out = append(out, canon)
		}
	}
	return out, nil
}
