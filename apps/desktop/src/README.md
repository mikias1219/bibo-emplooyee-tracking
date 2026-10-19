# Desktop UI (`apps/desktop/src`)

Feature-based layout. Import across folders with the `@/` alias (`@/shared/ui`);
inside one feature use relative paths.

```
main.tsx              entry: i18n, Sentry, mounts <App/>
app/                  shell: App.tsx (navigation, session, admin gating), tray menu
features/<name>/      one folder per screen/capability — owns its components + logic
  auth/               Login (Selamnew organization account), Welcome
  onboarding/  permissions/
  dashboard/  activity/  screenshots/  browser/   personal tracking views
  settings/  messages/  updater/
  admin/              owner-only (Org&Emp owners/admins)
    types.ts          shapes of the admin Tauri commands (mirror src-tauri sync/client.rs)
    lib.ts            shared helpers: memberStatus, isSelf, initials, localDayStart
    components/       pieces used by several admin screens (WorkspacePicker)
    team-overview/  members/  employee-detail/ (+ panels/)
shared/               used by 2+ features; never imports from features/
  api/tauri.ts        `call()` — the only wrapper around Tauri `invoke`
  ui/                 design-system primitives (StatCard, Segmented, …)
  components/         app-wide widgets (LanguageSwitcher, AuthTitleBar)
  lib/                log, analytics, sentry, dragWindow
i18n/                 i18next setup + locales/<lang>/<namespace>.json
styles/theme.css      design tokens + component styles
```

## Adding a feature
1. Create `features/<name>/` with its screen component(s).
2. If it needs the backend, add a Rust command in `src-tauri/src/commands/mod.rs`
   (+ a `BackendClient` method in `sync/client.rs`), register it in `lib.rs`, and call
   it with `call("<command>")` from `@/shared/api/tauri`.
3. Add strings to `i18n/locales/en/*.json` (the UI is English-only).
4. Wire the screen into navigation in `app/App.tsx`.

Admin screens shared with the web admin (Performance, Rules, Workspace settings,
Attendance, Agent health, Audit log) live in `packages/admin-ui` and are hosted by
`features/admin/AdminScope.tsx` over the `admin_api` Tauri command — add new admin
screens there, not here.

Move code into `shared/` only once a second feature needs it.
