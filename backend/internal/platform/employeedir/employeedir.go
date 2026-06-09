package employeedir

import (
	"context"
	_ "embed"

	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

//go:embed employees.csv
var csvData []byte

// Seed parses the embedded directory CSV and full-syncs it into every
// Google-configured workspace. Best-effort: all failures are logged, never
// fatal — a directory glitch must not take the server down. If the CSV parses
// to zero records the sync is skipped entirely (guard against an empty/truncated
// CSV wiping the directory). §9.4
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
	wsIDs, err := store.ListWorkspacesWithGoogle(ctx)
	if err != nil {
		log.Error("employee_seed_failed", zap.Error(err))
		return
	}
	if len(wsIDs) == 0 {
		// No workspace has Google configured yet — the directory has nowhere to
		// land. This is the expected steady state until Google is set up, so it's
		// Info (a distinct message, not the misleading employee_seed_done{0}).
		log.Info("employee_seed_no_google_workspaces")
		return
	}
	for _, wsID := range wsIDs {
		added, updated, deleted, serr := store.SyncEmployees(ctx, wsID, seeds)
		if serr != nil {
			log.Error("employee_sync_failed", zap.String("workspace_id", wsID.String()), zap.Error(serr))
			continue
		}
		log.Info("employees_synced",
			zap.String("workspace_id", wsID.String()),
			zap.Int("added", added), zap.Int("updated", updated), zap.Int("deleted", deleted))
	}
	log.Info("employee_seed_done", zap.Int("workspaces", len(wsIDs)))
}
