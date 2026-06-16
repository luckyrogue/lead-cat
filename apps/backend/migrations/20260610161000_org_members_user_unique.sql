

ALTER TABLE organization_members
  ADD CONSTRAINT organization_members_org_user_unique UNIQUE (organization_id, user_id);

ALTER TABLE organization_members
  DROP CONSTRAINT IF EXISTS organization_members_org_user_unique;
