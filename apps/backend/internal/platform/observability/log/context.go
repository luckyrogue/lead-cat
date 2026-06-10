package log

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ctxKey string

const (
	keyRequestID    ctxKey = "request_id"
	keyOrganization ctxKey = "organization_id"
)

func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, keyRequestID, id)
}

func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(keyRequestID).(string); ok {
		return v
	}
	return ""
}

func WithOrganizationID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, keyOrganization, id.String())
}

func FieldsFromContext(ctx context.Context) []zap.Field {
	var fields []zap.Field
	if id := RequestIDFromContext(ctx); id != "" {
		fields = append(fields, zap.String("request_id", id))
	}
	if v, ok := ctx.Value(keyOrganization).(string); ok && v != "" {
		fields = append(fields, zap.String("organization_id", v))
	}
	return fields
}

func WithContext(ctx context.Context, log *zap.Logger) *zap.Logger {
	return log.With(FieldsFromContext(ctx)...)
}
