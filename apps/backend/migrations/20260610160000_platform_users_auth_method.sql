
ALTER TABLE platform_users ADD COLUMN auth_method TEXT NOT NULL DEFAULT '';

ALTER TABLE platform_users DROP COLUMN auth_method;
