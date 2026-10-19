# Roadmap modules → code

Source of truth for scope: `Selamnew_Workforce_Intelligence_Roadmap.pptx` (10 modules).
This page maps each module to where it lives in the repo and what exists today.
The delivery plan is in [SPRINT_PLAN.md](SPRINT_PLAN.md).

Legend: ✅ done · 🟡 foundation / partial · ❌ not started

## Repository layout

```
apps/
  backend/        Go API (Gin + Postgres). One package per concern under internal/:
                  access (roles/scope) · audit · orgauth + orgemp (Selamnew identity)
                  · identity (Firebase) · store (SQL) · handlers (HTTP) · server (routes)
  web-admin/      React admin (browser). Thin host for packages/admin-ui + roster/detail pages
  desktop/        Tauri agent + UI for Windows, macOS, Linux
    src/          React UI (features/*, shared/*) — hosts packages/admin-ui for admins
    src-tauri/    Rust agent: trackers, platform/{windows,macos,linux}.rs, sync, tray
  extension/      Browser extension (website activity → local agent)
  linux-inputd/   Prototype privileged input helper (not shipped)
packages/
  admin-ui/       Admin screens shared by web + desktop (one implementation, every platform)
docs/roadmap/     This file + the sprint plan
```

Rule of thumb: **admin screens go in `packages/admin-ui`**, never twice. Each client only
supplies an `AdminApi` (web: fetch, desktop: the `admin_api` Tauri command).

## Module map

| # | Module | Status | Backend | Admin UI | Agent / other |
|---|---|---|---|---|---|
| 1 | Desktop Agent | 🟡 | `handlers/sync.go`, `handlers/agents.go` (check-in) | Agent health page | `desktop/src-tauri` (offline SQLite queue, sync worker, policy pull, check-in every 15 min) |
| 2 | Activity Tracking | 🟡 | `store/sync.go`, `store/reports.go`; `keystroke_buckets.mouse_*` columns (M14) | employee detail panels | `src-tauri/src/trackers` (active/idle, app focus, keystroke counts) |
| 3 | Web Activity | 🟡 | `browser_visits` | browser panel | `apps/extension` (Chromium MV3 — Chrome verified, Edge/Firefox in Sprint 1) |
| 4 | Screenshots | 🟡 | `handlers/screenshot.go`, retention sweeper, audit on view | gallery, Workspace settings | interval capture, privacy mode, skip-apps |
| 5 | Productivity Analytics | ✅/🟡 | `handlers/productivity.go`; `productivity_rules.department_id` (M14) | Performance, Productivity rules | — |
| 6 | Observed Attendance | 🟡 | `store/attendance.go`, `GET /businesses/:id/attendance` | Attendance page | — |
| 7 | Reports & Dashboards | 🟡 | `store/reports.go`, roster `department_id` filter | Team overview, Members, employee detail | — |
| 8 | Selamnew Integration | 🟡 | `orgauth`, `orgemp`, directory sync, manager line `memberships.manager_org_user_id` | Members "Sync from directory" | login with Selamnew account on every platform |
| 9 | Access Control & Privacy | 🟡 | `internal/access` (owner/manager/employee scope), `internal/audit`, `audit_log` | Audit log page | counts only, never typed text |
| 10 | Administration | 🟡 | `department_policies` + `handlers/policies.go`, business working hours/time zone (M14) | Workspace settings | policy applied by agent on sync |

M14 = migration `00014_roadmap_foundation.sql`.

## Roles (module 9)

| Role | Source (Org&Emp) | Sees |
|---|---|---|
| `owner` (Admin) | Core role in `OWNER_ROLE_SLUGS` (default `owner,admin`) or `OWNER_EMAILS` | whole organization + settings |
| `manager` | department lead (`departmentLeadOrNot`) | own department only (`access.Scope.DepartmentFilter`) |
| `employee` | everyone else | own activity on their device |

New endpoints must resolve access with `access.Require(c, store, "owner"|"manager")`
and pass `scope.DepartmentFilter(...)` into queries.

## API added by the foundation

| Method | Path | Role | Module |
|---|---|---|---|
| POST | `/v1/agent/checkin` | any signed-in agent | 1, 10 |
| GET | `/v1/businesses/:id/agents` | manager+ | 1, 10 |
| GET | `/v1/businesses/:id/attendance?from&to&department_id` | manager+ | 6 |
| GET/PUT/DELETE | `/v1/businesses/:id/department-policies[/:dept]` | owner (GET one: manager of that dept) | 10 |
| GET | `/v1/businesses/:id/audit` | owner | 9 |

Audit entries are written for every per-employee report view, every screenshot
view, workspace settings changes and department policy changes.
