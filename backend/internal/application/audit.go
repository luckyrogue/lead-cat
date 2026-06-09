package application

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/luckyrogue/lead-cat/internal/infrastructure/persistence/postgres"
)

// auditWhitelist maps action -> allowed detail keys.
var auditWhitelist = map[string]map[string]struct{}{
	"google_config_updated": {"subject": {}, "calendar_id": {}, "has_new_sa_json": {}},
	"google_verified":       {"ok": {}, "calendar_summary": {}, "time_zone": {}, "error_code": {}},
	"chat_linked":    {"chat_id": {}, "chat_title": {}},
	"members_synced": {"added": {}, "removed": {}, "unchanged": {}},
}

// sanitizeAuditDetails filters details by the action's whitelist. Returns the
// JSON-encoded surviving keys + the list of dropped keys for logging.
func sanitizeAuditDetails(action string, details map[string]any) (json.RawMessage, []string) {
	wl := auditWhitelist[action]
	clean := map[string]any{}
	var dropped []string
	for k, v := range details {
		if _, ok := wl[k]; ok {
			clean[k] = v
		} else {
			dropped = append(dropped, k)
		}
	}
	b, _ := json.Marshal(clean)
	return b, dropped
}

// AuditContext carries the actor identity through the request lifecycle.
type AuditContext struct {
	UserID     uuid.UUID
	TelegramID int64
	Email      string
}

type auditCtxKey struct{}

// WithAuditActor stores the actor in ctx (set by the middleware/handler).
func WithAuditActor(ctx context.Context, a AuditContext) context.Context {
	return context.WithValue(ctx, auditCtxKey{}, a)
}

// auditActor returns (actor, ok). ok=false when the ctx has no audit actor —
// in that case the caller must skip the audit write (with a Warn log).
func auditActor(ctx context.Context) (AuditContext, bool) {
	v, ok := ctx.Value(auditCtxKey{}).(AuditContext)
	return v, ok
}

// Audit records an admin action. Audit-write failures NEVER fail the parent
// operation — they are logged at Warn.
func (s *Services) Audit(ctx context.Context, action, targetKind, targetID string, details map[string]any) {
	actor, ok := auditActor(ctx)
	if !ok {
		s.Log.Warn("audit_actor_missing", zap.String("action", action), zap.String("target_id", targetID))
		return
	}
	clean, dropped := sanitizeAuditDetails(action, details)
	if len(dropped) > 0 {
		s.Log.Warn("audit_unexpected_keys", zap.String("action", action), zap.Strings("dropped", dropped))
	}
	err := s.Store.InsertAuditEntry(ctx, postgres.AuditEntry{
		ActorUserID:     actor.UserID,
		ActorTelegramID: actor.TelegramID,
		ActorEmail:      actor.Email,
		Action:          action,
		TargetKind:      targetKind,
		TargetID:        targetID,
		Details:         clean,
	})
	if err != nil {
		s.Log.Warn("audit_write_failed", zap.String("action", action), zap.Error(err))
	}
}
