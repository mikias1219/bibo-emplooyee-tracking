-- +goose Up
-- Selamnew Org&Emp integration. Identity, organizations and departments are owned
-- by the shared Org&Emp (core) service; this database keeps a local mirror keyed by
-- the Org&Emp ids so every activity table keeps its existing user/business FKs.
--
--   tenant (Org&Emp)        → businesses.tenant_id   (one business per tenant)
--   user   (Org&Emp)        → users.org_user_id / users.firebase_uid
--   active job department   → memberships.department_id / department_name
--   Core role (owner/admin) → memberships.role = 'owner', everyone else 'employee'

-- Org users never have a local password (Firebase owns the credential).
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ADD COLUMN org_user_id  uuid UNIQUE;
ALTER TABLE users ADD COLUMN firebase_uid text UNIQUE;
ALTER TABLE users ADD COLUMN avatar_url   text;

ALTER TABLE businesses ADD COLUMN tenant_id text UNIQUE;

ALTER TABLE memberships ADD COLUMN department_id   text;
ALTER TABLE memberships ADD COLUMN department_name text;
ALTER TABLE memberships ADD COLUMN job_title       text;
ALTER TABLE memberships ADD COLUMN is_department_lead boolean NOT NULL DEFAULT false;
-- Set when the user disappears from (or is soft-deleted in) the Org&Emp directory;
-- deactivated members can't sign in and are hidden from rosters. History is kept.
ALTER TABLE memberships ADD COLUMN deactivated_at timestamptz;
ALTER TABLE memberships ADD COLUMN synced_at      timestamptz;
CREATE INDEX idx_memberships_business_dept ON memberships(business_id, department_id);

-- +goose Down
DROP INDEX idx_memberships_business_dept;
ALTER TABLE memberships DROP COLUMN synced_at;
ALTER TABLE memberships DROP COLUMN deactivated_at;
ALTER TABLE memberships DROP COLUMN is_department_lead;
ALTER TABLE memberships DROP COLUMN job_title;
ALTER TABLE memberships DROP COLUMN department_name;
ALTER TABLE memberships DROP COLUMN department_id;
ALTER TABLE businesses DROP COLUMN tenant_id;
ALTER TABLE users DROP COLUMN avatar_url;
ALTER TABLE users DROP COLUMN firebase_uid;
ALTER TABLE users DROP COLUMN org_user_id;
-- Down is only safe while no org users (NULL password_hash) remain.
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
