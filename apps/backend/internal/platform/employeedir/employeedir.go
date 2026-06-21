package employeedir

import (
	"context"
	_ "embed"

	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

var csvData []byte

func Seed(ctx context.Context, store *postgres.Store, log *zap.Logger) {
	records, err := Parse(csvData)
	if err != nil {
		log.Error("employee_csv_parse_failed", zap.Error(err))
		return
	}
	if len(records) == 0 {
		log.Warn("employee_csv_empty")
		return
	}
	seeds := make([]postgres.EmployeeSeed, 0, len(records))
	for _, r := range records {
		seeds = append(seeds, postgres.EmployeeSeed{FullName: r.FullName, Email: r.Email, Dept: r.Dept})
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
