import { createAdminApi, type AdminApi } from "@ctracking/admin-ui";
import { call } from "@/shared/api/tauri";

/**
 * The desktop AdminApi: the shared admin screens (@ctracking/admin-ui) reach the
 * backend through the path-restricted `admin_api` Tauri command, which carries the
 * signed-in session (and refreshes it). Same on Linux, Windows and macOS.
 */
export const desktopAdminApi: AdminApi = createAdminApi((path, opts = {}) =>
  call("admin_api", {
    method: opts.method ?? "GET",
    path,
    query: opts.query ?? null,
    body: opts.body ?? null,
  }),
);
