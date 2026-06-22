package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

func filterEmployees(all []model.Employee, query string) []model.Employee {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []model.Employee
	for _, e := range all {
		if strings.Contains(strings.ToLower(e.FullName), q) || strings.Contains(strings.ToLower(e.Email), q) {
			out = append(out, e)
		}
	}
	return out
}

func (s *Services) SearchEmployees(ctx context.Context, organizationID uuid.UUID, query string) ([]model.Employee, error) {
	all, err := s.Store.ListEmployees(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return filterEmployees(all, query), nil
}

func (s *Services) SearchEmployeesGlobal(ctx context.Context, query string) ([]model.Employee, error) {
	return s.Store.SearchEmployeesGlobal(ctx, query)
}

func (s *Services) EmployeeSchedule(ctx context.Context, email string, from, to time.Time) ([]model.Meeting, error) {
	return s.Store.ListScheduleForEmail(ctx, email, from, to)
}

func (s *Services) ListParticipants(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingParticipant, error) {
	return s.Store.ListParticipants(ctx, meetingID)
}
