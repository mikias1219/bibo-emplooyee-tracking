-- +goose Up
-- Productivity / performance tracking. Owners classify the apps and website
-- domains their team uses as productive, unproductive or neutral; anything
-- without a rule is reported as "unclassified" (never silently counted as
-- unproductive). Patterns are stored normalised: app names lowercased, domains
-- as a bare lowercase host ("github.com", which also matches subdomains).
CREATE TABLE productivity_rules (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id uuid NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
    match_type  text NOT NULL CHECK (match_type IN ('app','domain')),
    pattern     text NOT NULL CHECK (length(pattern) BETWEEN 1 AND 255),
    category    text NOT NULL CHECK (category IN ('productive','unproductive','neutral')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (business_id, match_type, pattern)
);

-- Expected productive time per working day (EmpMonitor-style "productive
-- hours"); performance reports show productive time as a share of it.
ALTER TABLE businesses ADD COLUMN productive_target_s integer NOT NULL DEFAULT 28800
    CHECK (productive_target_s BETWEEN 3600 AND 86400);

-- +goose Down
ALTER TABLE businesses DROP COLUMN productive_target_s;
DROP TABLE productivity_rules;
