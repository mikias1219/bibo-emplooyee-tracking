# Web admin (`apps/web-admin/src`)

Feature-based layout. Import across folders with the `@/` alias (`@/shared/ui`);
inside one feature use relative paths.

```
main.tsx              entry: theme, i18n, Sentry, mounts <App/>
app/                  App.tsx (routes), layout/AppShell, routes/ (Protected, SuperAdmin)
features/<name>/      one folder per area — owns its pages, components and api.ts
  auth/               SignIn (Selamnew organization account), AuthContext, api.ts
  businesses/         useBusinesses (selected workspace context), api.ts
  dashboard/          team roster + KPIs
  employees/          directory-backed roster, department filter, Org&Emp sync, api.ts
    detail/           per-employee reports (+ panels/)
  admin/              AdminScope + api.ts: hosts the shared screens from packages/admin-ui
                      (Performance, Rules, Workspace settings, Attendance, Agents, Audit)
  internal-messages/  super-admin in-app messages, api.ts
shared/               used by 2+ features; never imports from features/
  api/                client (fetch + token refresh), tokenStore, types, demo data
  ui/                 primitives (Spinner, Notice, Empty, Modal)
  components/         app-wide widgets (LanguageSwitcher)
  lib/                format, terms, log, sentry
  theme/              theme.css tokens + ThemeProvider
i18n/                 i18next setup + locales/<lang>/<namespace>.json
```

## Adding a feature
1. Create `features/<name>/` with the page component(s).
2. Put its HTTP calls in `features/<name>/api.ts` using `request()` from
   `@/shared/api/client`; add response shapes to `@/shared/api/types`.
3. Add strings to `i18n/locales/en/*.json` (the UI is English-only).
4. Register the route in `app/App.tsx` and, if needed, a nav item in `app/layout/AppShell.tsx`.

Move code into `shared/` only once a second feature needs it.
