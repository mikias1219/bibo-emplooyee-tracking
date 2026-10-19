-- PROPOSED (not applied): add as the next free goose migration in
-- apps/backend/internal/db/migrations/ once the sync ingest is implemented.
-- Upserts on client_uuid must be scoped to the authenticated user/business.
-- +goose Up
CREATE TABLE IF NOT EXISTS input_activity_buckets (
 client_uuid UUID PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 business_id UUID NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
 device_id UUID NOT NULL,
 start_ts BIGINT NOT NULL,
 interval_s INTEGER NOT NULL CHECK(interval_s BETWEEN 1 AND 60),
 keyboard_presses BIGINT NOT NULL CHECK(keyboard_presses >= 0),
 mouse_clicks BIGINT NOT NULL CHECK(mouse_clicks >= 0),
 pointer_motion_packets BIGINT NOT NULL CHECK(pointer_motion_packets >= 0),
 covered_ms INTEGER NOT NULL CHECK(covered_ms >= 0 AND covered_ms <= interval_s*1000),
 coverage TEXT NOT NULL CHECK(coverage IN ('complete','partial','unavailable')),
 gap_reason TEXT,
 source TEXT NOT NULL,
 client_updated_at BIGINT NOT NULL,
 received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(coverage != 'complete' OR covered_ms = interval_s*1000),
 CHECK(coverage != 'unavailable' OR covered_ms = 0),
 CHECK(coverage = 'complete' OR gap_reason IS NOT NULL)
);
CREATE INDEX idx_input_biz_user_ts ON input_activity_buckets (business_id,user_id,start_ts);
-- +goose Down
DROP TABLE IF EXISTS input_activity_buckets;
