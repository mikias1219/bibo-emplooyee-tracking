# Workforce Intelligence — Linux input helper (prototype)

**Status: partially implemented prototype, not deployable employee-monitoring software.**
Standalone Cargo workspace: it is NOT part of the desktop app build, installer or
CI, and nothing here is installed or enabled. It builds and its tests pass
(`cargo test --workspace`, `python3 -m unittest discover -s tests`).

## Integration status in this repo
- **GNOME extension: merged and in use.** The starter's focus bridge was merged
  with the agent's extension at `apps/desktop/linux/gnome-extension/` (D-Bus name
  `com.workforceintelligence.Focus1`, adds `GetFocusedAppInfo` and GNOME 50). The
  desktop agent reads it from `apps/desktop/src-tauri/src/platform/linux.rs`.
- **Idle monitor: implemented** in `platform/linux.rs` (Mutter IdleMonitor, KDE fallback).
- **Input buckets (this workspace): not wired.** `agent-bridge` and `sql/` are the
  agreed shape for keyboard/mouse counts with coverage, to be wired into
  `storage/mod.rs`, `sync/worker.rs` and the Go sync ingest together with a
  working collector.
- Fixes applied to the starter: bounded frame reads + per-connection timeouts and
  errors no longer stop the daemon; socket created 0660; `gap_reason` stored in
  snake_case (was e.g. `permissiondenied`); complete+gap_reason rejected in
  `validate()`; `pending_json` borrow error (did not compile); backend SQL now has
  foreign keys; schema test renamed/added for the UUID uniqueness case.

## Implemented here
- Shared, content-free Rust `InputBucket` contract, coverage and protocol messages, validation/tests.
- SQLite pending queue schema and enqueue/ACK implementation (not yet wired to BiBo).
- PostgreSQL input bucket migration for adding to BiBo's goose migrations.
- GNOME Shell 46+ focused-app identifier extension with opt-in D-Bus bridge; needs GNOME runtime QA.
- Root helper **IPC-only prototype** with peer-UID check and no raw device access. This intentionally cannot start monitoring yet.
- systemd prototype with no permitted input devices.
- SQLite integrity tests, locally executable with Python.

## Not yet implemented (required for actual collection)
1. Complete `workforce-inputd` evdev reader, device-class detection, accurate SYN-report motion counters, device hotplug and dropped-event detection.
2. Administrator enrollment and cryptographically validated user consent; logind active seat/session checks on every start/resume. `WI_ALLOWED_UID` alone is NOT sufficient to authorize monitoring.
3. Rust GNOME D-Bus client, idle monitor, screenshot portal/PipeWire receiver.
4. Integration of SQLite queue helpers into BiBo `storage/mod.rs` migration dispatcher and `sync/worker.rs`.
5. Go `handlers/sync.go` and `store/sync.go` extension, with secure tenant-scoped idempotency and an acknowledged `input_activity` array.
6. Browser extension domain-only enforcement at source, screenshot privacy and Selamnew grouping.
7. Packaging, GNOME validation, privilege-hardening, native Rust compile and end-to-end QA.

## Files to apply to BiBo
- `sql/001_local_input.sql` => desktop SQLite schema v3 migration.
- `sql/002_backend_input.sql` => next free numbered goose migration under `apps/backend/internal/db/migrations/`.
- `crates/agent-bridge` => wire `enqueue`/`pending_json`/`acknowledge` into BiBo's Rust desktop storage/sync.
- `crates/inputd` => separate privileged service; **do not deploy or grant input access** until authorization/session control has been implemented and reviewed.

## Run tests
```
python3 -m unittest discover -s tests -v
# with Rust toolchain and dependency access:
cargo test --workspace
```

## Security design decisions
- No keystroke identities or text, cursor coordinates, raw input event histories.
- Any missing collector/permission => `unavailable` or `partial`, never zero-as-complete.
- No privileged-helper network access or backend tokens.
- Mandatory per-user visible consent, pause, and session lock stop before enabling raw input collection.
- For screenshots: obtain ScreenCast portal permission; never bypass selection/prompt.

See existing `Workforce_Intelligence_GNOME_Wayland_Implementation_Spec.md` for implementation details and acceptance requirements.
