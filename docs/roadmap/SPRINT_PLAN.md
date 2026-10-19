# Workforce Intelligence — 45-day sprint plan

**Window:** Mon 19 Oct 2026 → Wed 2 Dec 2026 (45 calendar days, 33 working days)
**Scope:** the 10 modules of `Selamnew_Workforce_Intelligence_Roadmap.pptx`, MVP depth,
on **Windows (primary), macOS and Linux**, plus web admin.
**Code map:** [MODULES.md](MODULES.md) · **Verify before every push:** `pnpm check`

## Timeline

| Sprint | Dates | Working days | Theme | Exit criteria |
|---|---|---|---|---|
| **S0 Foundation** | 19 – 23 Oct | 5 | Structure, schema, roles, shared admin UI | ✅ code merged; `pnpm check` green |
| **S1 Agent & privacy** | 26 Oct – 6 Nov | 10 | Agent install/identity/resilience on 3 OSes; privacy-by-design data | Silent install on Windows; no URLs/titles stored |
| **S2 Access & administration** | 9 – 20 Nov | 10 | Manager scoping everywhere, department policies, Selamnew integration | Manager sees only own department in every view |
| **S3 Insight & release** | 23 Nov – 2 Dec | 8 | Attendance, dashboards, export, agent health, QA, pilot | Pilot org live on Windows; QA matrix signed off |

Platform tags: **BE** backend · **ADM** shared admin UI (web + desktop) · **WEB** web admin ·
**AGT** desktop agent, all OSes · **WIN / MAC / LNX** OS-specific · **EXT** browser extension · **QA**

---

## S0 — Foundation (19 – 23 Oct) · done this week

| ID | Module | Platform | Task | Status / acceptance |
|---|---|---|---|---|
| S0-1 | 8 | BE, ADM, AGT | Login with the Selamnew Org&Emp account; directory + department sync | ✅ |
| S0-2 | — | all | Feature-based folders; shared `packages/admin-ui`; English-only i18n | ✅ |
| S0-3 | all | BE | Migration 00014: roles (owner/manager/employee), manager line, department policies, working hours, mouse counts, agent health, audit log, activity retention | ✅ migrates up/down |
| S0-4 | 9 | BE | `internal/access` scope + `internal/audit`; audit on report/screenshot views and settings | ✅ integration-tested |
| S0-5 | 6, 1, 9 | BE, ADM | Starter pages + APIs: Attendance, Agent health, Audit log (web + desktop) | ✅ |
| S0-6 | 1 | AGT | Agent check-in (version, OS, hostname) every 15 min | ✅ |
| S0-7 | — | all | `scripts/check.sh` (`pnpm check`); CI job running it on every PR | check.sh ✅ · **CI: Fri 23 Oct** |
| S0-8 | — | all | Production backend URL + updater endpoint replace bibotracker.com defaults | **Fri 23 Oct** (needs the prod URL) |

## S1 — Agent & privacy (26 Oct – 6 Nov)

Goal: an agent IT can deploy and forget, collecting only what the roadmap allows.

| ID | Module | Platform | Task | Acceptance |
|---|---|---|---|---|
| S1-1 | 3, 9 | BE, AGT, EXT | **Domain-only web activity**: store host only; drop `url`/`page_title` (migration + extension + agent) | DB has no full URLs/titles; existing rows reduced to domain |
| S1-2 | 2, 9 | AGT, BE | **No window titles**: app name only; migrate existing `window_title` to NULL | Activity reports show apps only |
| S1-3 | 2 | WIN, MAC, LNX | Mouse clicks + movement counts per minute bucket into `keystroke_buckets.mouse_*` | Counts appear in employee detail; no coordinates stored |
| S1-4 | 1, 10 | AGT | Apply **working hours** from policy (org → department): capture screenshots only inside hours; tag activity in/out of hours | Screenshot outside hours = none |
| S1-5 | 1 | WIN | **MSI** with silent install (`msiexec /qn`), per-machine, autostart at login | IT installs on 3 PCs via script without prompts |
| S1-6 | 1 | MAC, LNX | Autostart (LaunchAgent / XDG autostart) + signed pkg / .deb | Agent running after reboot |
| S1-7 | 1, 8 | AGT | **Employee identification**: bind device to the Selamnew user on first login; remember across restarts; show "signed in as" | Re-login not needed after reboot |
| S1-8 | 1 | WIN, MAC, LNX | **Resilience**: auto-restart on crash, survive sleep/resume; hide "Quit" for managed (org) installs | Killing the process → back within 1 min |
| S1-9 | 3 | EXT | Edge + Firefox builds of the extension (domain-only) | Time per domain recorded on all 3 browsers |
| S1-10 | 1 | AGT, QA | Resource budget test: CPU < 2 %, RAM < 150 MB, < 5 MB/day upload | Measured on Windows 10/11 and Ubuntu 24.04 |

## S2 — Access & administration (9 – 20 Nov)

Goal: the right people see the right data, configured per department.

| ID | Module | Platform | Task | Acceptance |
|---|---|---|---|---|
| S2-1 | 9 | BE | Apply `access.Scope` to **all** existing report endpoints (roster, employee detail, screenshots, performance) | Manager gets 403 outside own department (tests) |
| S2-2 | 9 | ADM, WEB, AGT | Manager login: admin section limited to own department; employees see only own activity | Manual test with 3 roles |
| S2-3 | 8 | BE | Manager hierarchy from Org&Emp `reportingTo` (direct + indirect reports) | Line manager sees reports even outside dept (configurable) |
| S2-4 | 10 | ADM | **Department policies UI**: working hours, idle threshold, screenshot interval, app + domain exclusions | Save → agent applies on next sync |
| S2-5 | 10 | AGT | Agent reads effective policy (org + department override) from `/v1/policy` | Policy change reflected within 15 min |
| S2-6 | 5 | BE, ADM | Department-specific productivity labels (`productivity_rules.department_id`) | Social media productive for Marketing only |
| S2-7 | 4 | AGT, ADM | Screenshot exclusion by **domain** (active browser tab) | No capture while excluded domain focused |
| S2-8 | 8 | BE, ADM | **Mismatch review**: devices/users not matched to an Org&Emp employee | List + "link to employee" action |
| S2-9 | 9 | BE | **Activity retention** sweeper (`activity_retention_days`) for activity, keystrokes, browser | Rows older than N days deleted nightly |
| S2-10 | 9 | ADM | Audit log filters (who / action / date) + screenshot view restricted to roles | Every screenshot open appears in the log |
| S2-11 | 8 | WEB | Selamnew SSO hand-off (open from the Selamnew menu with the existing session via `/v1/auth/firebase`) | No second login from Selamnew |

## S3 — Insight & release (23 Nov – 2 Dec)

Goal: the dashboards the roadmap promises, then a pilot.

| ID | Module | Platform | Task | Acceptance |
|---|---|---|---|---|
| S3-1 | 6 | BE, ADM | Attendance vs schedule: late start, early finish, overtime; **no-activity days** | Flags match the department's working hours |
| S3-2 | 7 | BE, ADM | **Department dashboard** + **department comparison** (active time, productivity, attendance) | Drill-down dept → team → person |
| S3-3 | 5, 7 | ADM | Trends (week/month) for scores and active time | Chart per level |
| S3-4 | 7 | BE, ADM | **Export** Excel + PDF for every report | File opens with the same numbers as screen |
| S3-5 | 1, 10 | BE, ADM | Agent health **alerts** (not reporting > 24 h) + version distribution | Alert listed + optional email via notification service |
| S3-6 | 1, 10 | AGT, BE | **Central updates**: own update channel (signed), staged rollout | Pilot agents update without visiting PCs |
| S3-7 | QA | WIN, MAC, LNX | Cross-platform QA matrix (Win 10/11, macOS 14/15, Ubuntu 24.04 GNOME Wayland) + privacy review | All P1 cases pass |
| S3-8 | — | all | Pilot: one department, deploy MSI, run 3 days, collect feedback | Go/no-go on 2 Dec |

---

## Working agreements

- **One implementation per admin screen** in `packages/admin-ui`; clients only add transport.
- New endpoints use `access.Require` + `scope.DepartmentFilter`; anything viewing personal data writes an audit entry.
- Schema changes are new goose migrations only (never edit a shipped one).
- `pnpm check` must pass before a PR; integration tests: `CHECK_DB_URL=… pnpm check`.
- Privacy guardrail: counts and domains only — never typed text, full URLs, page/window titles, or content.

## Risks

| Risk | Mitigation |
|---|---|
| Windows silent install / code signing certificate not ready | Order the certificate in S0; NSIS fallback for internal pilot |
| Org&Emp lacks a service token for background sync | Request `ORG_SERVICE_AUTH_TOKEN` from the core team in S1; owner-session sync meanwhile |
| Linux Wayland limits (focus, screenshots) | GNOME extension + portal path already in place; X11 documented as best-effort |
| Removing titles/URLs reduces report detail | **Decision needed before S1-1/S1-2** (roadmap says domain/app only); domain + app are enough for scoring |
