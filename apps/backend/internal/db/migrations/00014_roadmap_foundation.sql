-- +goose Up
-- Roadmap foundation (Selamnew Workforce Intelligence, modules 1–10).
-- Schema only: every column/table here is additive with safe defaults, so existing
-- agents and reports keep working. Features are built on top in later sprints
-- (see docs/roadmap/SPRINT_PLAN.md).

-- ── M9 Access control: Admin (owner) / Manager / Employee ──────────────────────
ALTER TABLE memberships DROP CONSTRAINT IF EXISTS memberships_role_check;
ALTER TABLE memberships ADD CONSTRAINT memberships_role_check
    CHECK (role IN ('owner', 'manager', 'employee'));
-- M8 Manager hierarchy: the Org&Emp reporting line (manager's org user id).
ALTER TABLE memberships ADD COLUMN manager_org_user_id uuid;
CREATE INDEX idx_memberships_manager ON memberships(business_id, manager_org_user_id);

-- ── M10 Administration / M1 central configuration: working hours + policies ────
ALTER TABLE businesses ADD COLUMN timezone        text     NOT NULL DEFAULT 'Africa/Addis_Ababa';
ALTER TABLE businesses ADD COLUMN work_start      time     NOT NULL DEFAULT '08:30';
ALTER TABLE businesses ADD COLUMN work_end        time     NOT NULL DEFAULT '17:30';
ALTER TABLE businesses ADD COLUMN work_days       smallint[] NOT NULL DEFAULT '{1,2,3,4,5}'; -- ISO 1=Mon … 7=Sun
-- M4 domain exclusions (apps already exist as screenshot_skip_apps).
ALTER TABLE businesses ADD COLUMN screenshot_skip_domains text[] NOT NULL DEFAULT '{}';
-- M9 retention for activity data (screenshots already have screenshot_retention_days). NULL = keep.
ALTER TABLE businesses ADD COLUMN activity_retention_days integer CHECK (activity_retention_days IS NULL OR activity_retention_days > 0);

-- Per-department overrides of the organization policy. NULL column = inherit.
CREATE TABLE department_policies (
    business_id             uuid NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    department_id           text NOT NULL,
    work_start              time,
    work_end                time,
    work_days               smallint[],
    idle_threshold_s        integer CHECK (idle_threshold_s IS NULL OR idle_threshold_s > 0),
    screenshot_interval_s   integer CHECK (screenshot_interval_s IS NULL OR screenshot_interval_s > 0),
    screenshot_skip_apps    text[],
    screenshot_skip_domains text[],
    updated_at              timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (business_id, department_id)
);

-- ── M5 Productivity: department-specific classification (NULL = organization-wide) ──
ALTER TABLE productivity_rules ADD COLUMN department_id text;

-- ── M2 Activity: mouse intensity alongside keystroke counts (counts only) ──────
ALTER TABLE keystroke_buckets ADD COLUMN mouse_clicks integer NOT NULL DEFAULT 0;
ALTER TABLE keystroke_buckets ADD COLUMN mouse_moves  integer NOT NULL DEFAULT 0;

-- ── M1/M10 Agent health: what each installed agent reports at check-in ────────
ALTER TABLE devices ADD COLUMN agent_version   text;
ALTER TABLE devices ADD COLUMN os              text;   -- windows | macos | linux
ALTER TABLE devices ADD COLUMN os_version      text;
ALTER TABLE devices ADD COLUMN hostname        text;
ALTER TABLE devices ADD COLUMN first_seen_at   timestamptz NOT NULL DEFAULT now();
ALTER TABLE devices ADD COLUMN last_checkin_at timestamptz;

-- ── M9 Audit log: who viewed which report/screenshot or changed what ──────────
CREATE TABLE audit_log (
    id            bigserial PRIMARY KEY,
    business_id   uuid REFERENCES businesses(id) ON DELETE CASCADE,
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action        text NOT NULL,          -- e.g. screenshot.view, report.view, settings.update
    target_type   text,                   -- employee | screenshot | business | department
    target_id     text,
    meta          jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_business_time ON audit_log(business_id, created_at DESC);

-- +goose Down
DROP TABLE audit_log;
ALTER TABLE devices DROP COLUMN last_checkin_at, DROP COLUMN first_seen_at, DROP COLUMN hostname,
    DROP COLUMN os_version, DROP COLUMN os, DROP COLUMN agent_version;
ALTER TABLE keystroke_buckets DROP COLUMN mouse_moves, DROP COLUMN mouse_clicks;
ALTER TABLE productivity_rules DROP COLUMN department_id;
DROP TABLE department_policies;
ALTER TABLE businesses DROP COLUMN activity_retention_days, DROP COLUMN screenshot_skip_domains,
    DROP COLUMN work_days, DROP COLUMN work_end, DROP COLUMN work_start, DROP COLUMN timezone;
DROP INDEX idx_memberships_manager;
ALTER TABLE memberships DROP COLUMN manager_org_user_id;
ALTER TABLE memberships DROP CONSTRAINT memberships_role_check;
ALTER TABLE memberships ADD CONSTRAINT memberships_role_check CHECK (role IN ('owner', 'employee'));
