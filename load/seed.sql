-- Deterministic load-test fixtures.  Safe to run multiple times (ON CONFLICT DO NOTHING).
-- Organizations required columns (from init.sql workspaces, now renamed):
--   slug TEXT NOT NULL UNIQUE, name TEXT NOT NULL
-- All other columns are nullable or have defaults.

INSERT INTO organizations (id, slug, name)
VALUES ('11111111-1111-1111-1111-111111111111', 'loadtest-org', 'Loadtest Org')
ON CONFLICT (id) DO NOTHING;

INSERT INTO platform_users (id, auth_sub, email)
VALUES ('22222222-2222-2222-2222-222222222222', 'loadtest-host', 'loadhost@e2e.test')
ON CONFLICT (id) DO NOTHING;

INSERT INTO booking_event_types
  (id, host_user_id, organization_id, slug, title, description, duration_mins,
   active, timezone, avail_weekdays, avail_start_minute, avail_end_minute)
VALUES
  ('33333333-3333-3333-3333-333333333333',
   '22222222-2222-2222-2222-222222222222',
   '11111111-1111-1111-1111-111111111111',
   'loadtest-intro', 'Loadtest Intro', '', 30,
   true, 'Asia/Almaty', '{1,2,3,4,5}', 540, 1020)
ON CONFLICT (id) DO NOTHING;
