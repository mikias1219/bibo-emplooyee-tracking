// @ctracking/admin-ui — owner/admin screens shared by the web admin and the
// desktop app (Linux, Windows, macOS). Clients render these inside <AdminProvider>
// with their own transport (AdminApi), workspace and navigation.
import "./styles.css";

export { AdminProvider, useAdmin, type AdminContextValue, type AdminView } from "./context";
export { createAdminApi, foundationApi, viewerTz, type AdminApi, type AdminRequest } from "./api";
export { adminResources, ADMIN_NAMESPACES } from "./i18n";
export { Performance } from "./pages/performance/Performance";
export { ProductivityRules } from "./pages/performance/ProductivityRules";
export { WorkspaceSettings } from "./pages/settings/WorkspaceSettings";
export { Attendance } from "./pages/attendance/Attendance";
export { AgentHealth } from "./pages/agents/AgentHealth";
export { AuditLog } from "./pages/audit/AuditLog";
export { Empty, Modal, Notice, Spinner } from "./ui";
export { dayRangeToUnix, daysAgoIso, fmtBytes, fmtDuration, isoDate } from "./format";
export * from "./types";
