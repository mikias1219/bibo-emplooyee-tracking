# @ctracking/admin-ui

Admin screens shared by the web admin and the desktop app (Windows, macOS, Linux):
Performance, Productivity rules, Workspace settings, Attendance, Agent health, Audit log.
One implementation; each client only provides the wiring.

```
src/
  index.ts        public API (pages, AdminProvider, types, adminResources)
  context.tsx     AdminProvider: api, workspace, user, navigation (client-supplied)
  api.ts          AdminApi contract + createAdminApi(request) + foundationApi
  types.ts        domain types (mirror the Go backend JSON)
  pages/<area>/   one folder per screen
  components/     pieces shared by pages (PageHead, PeriodPicker, useLoad)
  ui.tsx · format.ts · hooks.ts
  locales/en/     admin, admin-performance, admin-settings namespaces
  styles.css      generated from the web theme, scoped under .wi-admin
```

## Hosting the screens

```tsx
<AdminProvider value={{ api, business, businessLoading, reloadBusiness, user, view, navigate, openEmployee }}>
  <Attendance />
</AdminProvider>
```

- Web: `apps/web-admin/src/features/admin/AdminScope.tsx` (fetch transport, react-router).
- Desktop: `apps/desktop/src/features/admin/AdminScope.tsx` (`admin_api` Tauri command).
- Merge `adminResources.en` into the client's i18next resources and add `ADMIN_NAMESPACES` to `ns`.

## Adding an admin screen

1. Add the backend endpoint (use `access.Require` for role/department scope).
2. Add the call to `AdminApi` in `api.ts` (and `createAdminApi` / `foundationApi`).
3. Create `pages/<area>/<Screen>.tsx`, export it from `index.ts`, add an `AdminView`.
4. Register it in both AdminScopes / navigations. Strings go in `locales/en/admin.json`.
