
ALTER TABLE bot_users ADD COLUMN timezone TEXT NOT NULL DEFAULT '';
ALTER TABLE bot_users ADD COLUMN language TEXT NOT NULL DEFAULT '';
ALTER TABLE platform_users ADD COLUMN timezone TEXT NOT NULL DEFAULT '';
ALTER TABLE platform_users ADD COLUMN language TEXT NOT NULL DEFAULT '';

ALTER TABLE bot_users DROP COLUMN timezone;
ALTER TABLE bot_users DROP COLUMN language;
ALTER TABLE platform_users DROP COLUMN timezone;
ALTER TABLE platform_users DROP COLUMN language;
