-- Extend BiBo's SQLite migration dispatcher to apply this once (schema v3).
CREATE TABLE IF NOT EXISTS input_activity_bucket (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 client_uuid TEXT NOT NULL UNIQUE,
 start_ts INTEGER NOT NULL,
 interval_s INTEGER NOT NULL CHECK(interval_s BETWEEN 1 AND 60),
 keyboard_presses INTEGER NOT NULL DEFAULT 0 CHECK(keyboard_presses >= 0),
 mouse_clicks INTEGER NOT NULL DEFAULT 0 CHECK(mouse_clicks >= 0),
 pointer_motion_packets INTEGER NOT NULL DEFAULT 0 CHECK(pointer_motion_packets >= 0),
 covered_ms INTEGER NOT NULL CHECK(covered_ms >= 0 AND covered_ms <= interval_s * 1000),
 coverage TEXT NOT NULL CHECK(coverage IN ('complete','partial','unavailable')),
 gap_reason TEXT,
 source TEXT NOT NULL,
 updated_at INTEGER NOT NULL,
 synced INTEGER NOT NULL DEFAULT 0 CHECK(synced IN (0,1)),
 CHECK(coverage != 'complete' OR covered_ms = interval_s * 1000),
 CHECK(coverage != 'unavailable' OR covered_ms = 0),
 CHECK(coverage = 'complete' OR gap_reason IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_input_bucket_pending ON input_activity_bucket(id) WHERE synced=0;
CREATE INDEX IF NOT EXISTS idx_input_bucket_time ON input_activity_bucket(start_ts);
