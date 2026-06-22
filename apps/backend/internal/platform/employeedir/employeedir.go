package employeedir

import (
	"context"
	_ "embed"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/application/model"
)

//go:embed employees.csv
var csvData []byte

type store interface {
	DefaultOrganizationWithGoogle(ctx context.Context) (uuid.UUID, bool, error)
	SyncEmployees(ctx context.Context, organizationID uuid.UUID, seeds []model.EmployeeSeed) (added, updated, deleted int, err error)
}

func Seed(ctx context.Context, store store, log *zap.Logger) {
	records, err := Parse(csvData)
	if err != nil {
		log.Error("employee_csv_parse_failed", zap.Error(err))
		return
	}
	if len(records) == 0 {
		log.Warn("employee_csv_empty")
		return
	}
	seeds := make([]model.EmployeeSeed, 0, len(records))
	for _, r := range records {
		seeds = append(seeds, model.EmployeeSeed{FullName: r.FullName, Email: r.Email, Dept: r.Dept})
	}
	orgID, ok, err := store.DefaultOrganizationWithGoogle(ctx)
	if err != nil {
		log.Error("employee_seed_failed", zap.Error(err))
		return
	}
	if !ok {
		log.Info("employee_seed_no_google_organizations")
		return
	}
	added, updated, deleted, serr := store.SyncEmployees(ctx, orgID, seeds)
	if serr != nil {
		log.Error("employee_sync_failed", zap.String("organization_id", orgID.String()), zap.Error(serr))
		return
	}
	log.Info("employees_synced",
		zap.String("organization_id", orgID.String()),
		zap.Int("added", added), zap.Int("updated", updated), zap.Int("deleted", deleted))
	log.Info("employee_seed_done", zap.Int("organizations", 1))
}
