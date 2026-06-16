
CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL,
    dept TEXT NOT NULL DEFAULT '',
    has_telegram BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, email)
);

CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    organizer_user_id UUID REFERENCES platform_users(id),
    dept TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    recurrence TEXT NOT NULL DEFAULT 'once',
    recurrence_until DATE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    google_event_id TEXT NOT NULL DEFAULT '',
    meet_link TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'scheduled',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX meetings_workspace_id_idx ON meetings (workspace_id);
CREATE INDEX meetings_organizer_idx ON meetings (organizer_user_id);

CREATE TABLE meeting_participants (
    meeting_id UUID NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    employee_id UUID REFERENCES employees(id) ON DELETE SET NULL,
    email TEXT NOT NULL
);

CREATE INDEX meeting_participants_meeting_idx ON meeting_participants (meeting_id);

DROP TABLE IF EXISTS meeting_participants;
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS employees;
