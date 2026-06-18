-- +goose Up
CREATE TABLE organization_join_requests (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id            UUID NOT NULL REFERENCES platform_users(id) ON DELETE CASCADE,
    status             TEXT NOT NULL DEFAULT 'pending',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at         TIMESTAMPTZ,
    decided_by_user_id UUID REFERENCES platform_users(id)
);
CREATE UNIQUE INDEX organization_join_requests_pending_idx
    ON organization_join_requests (organization_id, user_id) WHERE status = 'pending';

-- +goose Down
DROP TABLE organization_join_requests;
