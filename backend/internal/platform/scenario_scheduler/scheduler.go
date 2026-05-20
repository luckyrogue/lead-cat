package scenario_scheduler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Jaryq-Lab/notify-bot/internal/domain/scenario"
	asynqqueue "github.com/Jaryq-Lab/notify-bot/internal/infrastructure/queue/asynq"
	"github.com/Jaryq-Lab/notify-bot/internal/infrastructure/persistence/postgres"
	"github.com/Jaryq-Lab/notify-bot/internal/platform/observability/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const lockKey = "leadcat:scheduler:leader"

type Scheduler struct {
	store  *postgres.Store
	queue  *asynqqueue.Client
	rdb    *redis.Client
	log    *zap.Logger
	sent   map[string]struct{}
}

func New(store *postgres.Store, queue *asynqqueue.Client, rdb *redis.Client, log *zap.Logger) *Scheduler {
	return &Scheduler{store: store, queue: queue, rdb: rdb, log: log, sent: make(map[string]struct{})}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	ok, err := s.rdb.SetNX(ctx, lockKey, "1", 90*time.Second).Result()
	if err != nil || !ok {
		return
	}
	now := time.Now()
	scenarios, err := s.store.ListEnabledScenarios(ctx)
	if err != nil {
		s.log.Warn("list scenarios", zap.Error(err))
		return
	}
	for _, sc := range scenarios {
		def, err := scenario.ParseDefinition(sc.Definition)
		if err != nil {
			continue
		}
		for _, n := range def.TriggerNodes() {
			if n.Type != scenario.NodeTriggerCron {
				continue
			}
			var p struct {
				Hour     int   `json:"hour"`
				Minute   int   `json:"minute"`
				Weekdays []int `json:"weekdays"`
			}
			_ = json.Unmarshal(n.Parameters, &p)
			if !cronDue(now, p.Hour, p.Minute, p.Weekdays) {
				continue
			}
			key := sc.ID.String() + ":" + now.Format("2006-01-02") + ":" + n.ID
			if _, dup := s.sent[key]; dup {
				continue
			}
			s.sent[key] = struct{}{}
			run, err := s.store.CreateRun(ctx, sc.ID, sc.WorkspaceID, "cron")
			if err != nil {
				continue
			}
			if err := s.queue.EnqueueRun(ctx, run.ID, sc.ID, "cron"); err != nil {
				_ = s.store.FinishRun(ctx, run.ID, "failed", err.Error())
				metrics.IncScenarioRun("failed")
				s.log.Warn("enqueue", zap.Error(err), zap.String("run_id", run.ID.String()), zap.String("scenario_id", sc.ID.String()))
			}
		}
	}
}

func cronDue(now time.Time, hour, minute int, weekdays []int) bool {
	if now.Hour() != hour || now.Minute() != minute {
		return false
	}
	if len(weekdays) == 0 {
		return true
	}
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	for _, d := range weekdays {
		if d == wd {
			return true
		}
	}
	return false
}

// EnqueueManual used by API
func EnqueueManual(ctx context.Context, store *postgres.Store, queue *asynqqueue.Client, scenarioID, workspaceID uuid.UUID) (uuid.UUID, error) {
	run, err := store.CreateRun(ctx, scenarioID, workspaceID, "manual")
	if err != nil {
		return uuid.Nil, err
	}
	if err := queue.EnqueueRun(ctx, run.ID, scenarioID, "manual"); err != nil {
		_ = store.FinishRun(ctx, run.ID, "failed", err.Error())
		metrics.IncScenarioRun("failed")
		return uuid.Nil, err
	}
	return run.ID, nil
}
