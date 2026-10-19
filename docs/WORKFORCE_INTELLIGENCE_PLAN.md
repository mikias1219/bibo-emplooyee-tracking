# Workforce Intelligence — Base Decision, EmpMonitor Reference and Linux Plan

Date: 2026-10-19 · Companion to `docs/Workforce_Intelligence_BiBo_Fork_Engineering_Reference.docx`
and `docs/LOCAL-SETUP-LINUX.md`.

## 1. Decision

| | BiBoTracking (chosen base) | EmpMonitor (reference only) |
|---|---|---|
| Repo / commit | `ngnclht1102/bibo-emplooyee-tracking` @ `f41cdc4` | `EmpCloud/emp-monitor` @ `994f26d` |
| License | MIT | **AGPL-3.0** |
| Desktop agent | Included (Rust/Tauri, macOS + Windows) | **Not published** (`QT/` holds only a README) |
| Keyboard data | Counts only | Raw typed text (`employee_keystrokes.keystrokes TEXT`) + AI sentiment on it |
| Backend | Go + PostgreSQL | 9 Node.js services + MySQL + MongoDB + Redis |
| Admin UI | React 19 + Vite | React 19 + Vite |

**We build on BiBoTracking.** EmpMonitor is used only as a source of product and data-model ideas.

### Rules for using EmpMonitor

- **Copy no code, SQL, UI or assets from EmpMonitor.** AGPL-3.0 would require publishing our full
  modified source to every user of a hosted service. Re-implement ideas from this document only.
- Do not adopt EmpMonitor's raw keystroke text, keystroke sentiment analysis, email monitoring,
  screen recording, webcam, USB, remote-desktop (RDP) access, uninstall password, or full-URL
  browsing history. Each one conflicts with the privacy rules in the engineering reference.
- Re-clone it only for reading: `git clone https://github.com/EmpCloud/emp-monitor.git`
  (the reviewed schema is `Backend/migrations/emp-monitor.sql`).

## 2. Ideas to borrow from EmpMonitor (re-implemented in BiBo)

| Our milestone | EmpMonitor design observed | How we apply it in BiBo |
|---|---|---|
| M2 Mouse activity | `employee_activities` stores `active_seconds`, `keystrokes_count`, `mouseclicks_count`, `mousemovement_count` per app interval | Add a per-minute `mouse_buckets` table (`client_uuid, ts_bucket, movement_count, click_count, active_seconds, updated_at`) next to BiBo's keystroke buckets. Counts only, with no coordinates or click targets. |
| M3 Productivity categories | `organization_apps_web`: one row per app/site with `type` (app / website) and `status` (0 neutral, 1 productive, 2 non-productive), scoped per **department** | Add `category_rules (business_id, department_id NULL, match_type app/domain, pattern, category, effective_from, effective_to)`. Add an explicit **unclassified** state (EmpMonitor treats unknown as neutral), and version the rules by effective date. |
| M3 Report speed | `production_stats` keeps a daily rollup per user (`t_sec` total, `w_sec` working, `n_sec` non-working) | Add a nightly or on-sync `daily_rollups` table in Postgres (`user_id, day, active_s, idle_s, productive_s, unproductive_s, neutral_s, unclassified_s, rules_version`). Reports read the rollup instead of raw rows. |
| M4 Departments | `organization_departments` + `organization_department_location_relation` with `department_head_id` | Store department mapping with effective dates (from Selamnew) plus a `department_managers` table. A manager's report access is checked server-side against it. |
| Attendance estimate | `employee_attendance (date, start_time, end_time, is_manual_attendance, updated_by)` | Derive first/last activity per day. Allow manual corrections only with `corrected_by` + reason, kept as an audit trail and never overwriting the derived values. |
| Post-MVP alerts | `notification_rules` → `_conditions` → `_recipients` → `_alerts` | A good shape for later alerting (e.g. "agent offline > 1 day"). Out of MVP scope. |

## 3. Linux support

### 3.1 Already working on Linux

- Go backend, PostgreSQL (Docker) and the React web admin. Verified on Ubuntu 26.04 with
  `/healthz` and `/admin/`. Steps are in `docs/LOCAL-SETUP-LINUX.md`.
- The browser extension runs on Chrome/Chromium for Linux unchanged (it posts to the local agent on
  `127.0.0.1`).

### 3.2 Not working yet: the desktop agent

`apps/desktop/src-tauri/src/platform/` has only `macos.rs` and `windows.rs`, so the trackers have
no Linux backend. The shared pieces (Tauri, SQLite, sync worker, local ingest server) are already
cross-platform.

### 3.3 Work needed for a Linux agent

1. **`platform/linux.rs`** implementing the same functions as `windows.rs`:
   `capability_rows`, `open_settings`, `permission_status`, `request_screen_recording`,
   `request_input_monitoring`, `request_accessibility`, `run_keyboard_tap`, `idle_seconds`.
2. **`platform/mod.rs`**: add the `#[cfg(target_os = "linux")] mod linux; pub use linux::*;` branch.
3. **`src/lib.rs:43`**: check the existing `not(any(macos, windows))` fallback, and route Linux to
   the new backend.
4. **Consent, a required privacy fix:** `src/settings/mod.rs:179` requires consent only on Windows
   (`!cfg!(target_os = "windows") || s.consented`). On Linux that line skips consent. Change
   it so Linux also requires `s.consented` before any tracking starts.
5. **`Cargo.toml`**: add a `[target.'cfg(target_os = "linux")'.dependencies]` section for the
   D-Bus crate chosen below. Add only what each feature needs.

### 3.4 Per-feature approach (Wayland only)

Current Ubuntu has no X11 session. Ubuntu 26.04 ships GNOME Shell 50 with only a Wayland session
(`/usr/share/xsessions` does not exist), so the Linux agent is designed for **Wayland only**.
Wayland deliberately blocks apps from reading other windows and global input. Show any
unavailable feature as "Unavailable" in the UI; never fake it.

| Feature | Approach on GNOME Wayland | Status |
|---|---|---|
| Idle / active time | Session D-Bus `org.gnome.Mutter.IdleMonitor` → `GetIdletime` (ms). No root, no prompt. KDE fallback: `org.freedesktop.ScreenSaver.GetSessionIdleTime` | Verified on Ubuntu 26.04 |
| Activity level (replaces key/mouse counts) | Per-minute `active_seconds` derived from idle polling. Feeds the M2 bucket's `active_seconds` with `movement_count`/`click_count` left NULL on Linux | Design |
| Active app | Our own small GNOME Shell extension (JavaScript) exposing **only** the focused window's app ID (e.g. `org.gnome.TextEditor`) on the session D-Bus. No window titles. The employee installs and enables it, and it shows in the Extensions app. Retest on every GNOME major release | To build |
| Screenshots | xdg-desktop-portal **ScreenCast** with `persist_mode=2`: one-time user grant plus a restore token, and GNOME shows a screen-sharing indicator while capturing. Opt-in only | Portal present on 26.04; verify `xcap` uses it at the pinned version |
| Key counts | **Not on Linux.** The only route is a root service reading `/dev/input` (or `input` group membership), which gives keylogger-level access. Report "Unavailable" | Decided |
| Mouse counts | **Not on Linux**, for the same reason. Use activity level instead | Decided |

Rejected: the `InputCapture` portal (it diverts input to another machine, it does not observe
activity) and standardising laptops on an X11 desktop such as Xfce (legacy, and not under our
control on employee machines).

### 3.5 Build, packaging and QA

- Build packages (Ubuntu, needs sudo):
  `sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev librsvg2-dev libayatana-appindicator3-dev libssl-dev libxdo-dev pkg-config build-essential`,
  plus whatever `xcap` and the chosen D-Bus crate (e.g. `zbus`) require (check each crate's README at the pinned version).
- Ship the GNOME Shell extension alongside the `.deb`, with its own version and a tested GNOME range in `metadata.json`.
- Package with `pnpm tauri build` as `.deb` and AppImage. Start on login through an XDG autostart
  `.desktop` file that the user can see and disable.
- Use our own bundle ID, updater URL and signing key (never BiBo's), the same as for Windows.
- QA matrix: Ubuntu 26.04 GNOME (Wayland), the previous Ubuntu LTS GNOME (Wayland), one KDE Plasma Wayland desktop. Run the
  acceptance tests from section 12 of the engineering reference on each, especially
  "no typed text anywhere".

### 3.6 Estimate and placement

About 2 weeks for one developer: `linux.rs` with Mutter idle and activity level (3–4 days), the
GNOME Shell extension and its D-Bus link (3–4 days), the ScreenCast portal flow (2–3 days) and
"Unavailable" states, plus about 3 days of QA. Schedule it as a **new milestone after M5**. The MVP
target remains Windows.

## 4. Repository housekeeping

- Workspace contains only `workforce-intelligence/` (the BiBo fork). The earlier `Workforce-agent`
  skeleton and the `emp-monitor` clone were deleted on 2026-10-19.
- Local change against upstream: `scripts/dev-db.sh` accepts `PG_PORT` (Docker Postgres on 5433
  because the system Postgres owns 5432).
- Next: create the organization fork on GitHub, run `git remote set-url origin <fork-url>`, and tag
  the baseline: `git tag wi-baseline-f41cdc4`.
