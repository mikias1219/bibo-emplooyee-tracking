import type {
  Agent,
  AttendanceDay,
  AuditEntry,
  Business,
  BusinessSettingsPatch,
  PerformanceResponse,
  PrivacyAppCategory,
  ProductivityCategory,
  ProductivityRule,
  RuleMatchType,
  UsageItem,
} from "./types";

/**
 * The backend calls the admin screens need. Each client implements it with its
 * own transport: the web admin over fetch (`/v1/...`), the desktop app through
 * the `admin_api` Tauri command. All calls are owner-only on the server.
 */
export interface AdminApi {
  listMyBusinesses(): Promise<{ businesses: Business[] }>;
  updateBusinessSettings(id: string, patch: BusinessSettingsPatch): Promise<unknown>;
  getPrivacyApps(): Promise<{ categories: PrivacyAppCategory[] }>;
  cleanupScreenshots(id: string, olderThanDays: number): Promise<{ deleted_count: number; bytes_freed: number }>;
  reportPerformance(businessId: string, from: number, to: number, employeeId?: string): Promise<PerformanceResponse>;
  listProductivityRules(businessId: string): Promise<{ rules: ProductivityRule[] }>;
  productivityUsage(businessId: string, from: number, to: number): Promise<{ apps: UsageItem[]; domains: UsageItem[] }>;
  setProductivityRule(
    businessId: string,
    match_type: RuleMatchType,
    pattern: string,
    category: ProductivityCategory,
  ): Promise<{ rule: ProductivityRule | null }>;
  // Roadmap foundation
  listAttendance(businessId: string, from: number, to: number, departmentId?: string): Promise<{ days: AttendanceDay[] }>;
  listAgents(businessId: string, departmentId?: string): Promise<{ agents: Agent[] }>;
  listAudit(businessId: string, limit?: number, offset?: number): Promise<{ entries: AuditEntry[] }>;
}

/** The viewer's IANA time zone, so the server buckets days like the viewer does. */
export function viewerTz(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

/** Minimal transport the default implementation needs: GET/POST/PUT/PATCH/DELETE
 *  against the backend with JSON in/out. */
export type AdminRequest = <T>(
  path: string,
  opts?: { method?: string; query?: Record<string, string | number | undefined>; body?: unknown },
) => Promise<T>;

/** AdminApi over a generic request function (used by the desktop app). */
export function createAdminApi(request: AdminRequest): AdminApi {
  return {
    listMyBusinesses: () => request("/v1/businesses/mine"),
    updateBusinessSettings: (id, patch) => request(`/v1/businesses/${id}/settings`, { method: "PATCH", body: patch }),
    getPrivacyApps: () => request("/v1/public/screenshot-privacy-apps"),
    cleanupScreenshots: (id, olderThanDays) =>
      request(`/v1/businesses/${id}/screenshots/cleanup`, { method: "POST", query: { older_than_days: olderThanDays } }),
    reportPerformance: (businessId, from, to, employeeId) =>
      request(`/v1/businesses/${businessId}/performance`, {
        query: { from, to, tz: viewerTz(), ...(employeeId ? { employee_id: employeeId } : {}) },
      }),
    listProductivityRules: (businessId) => request(`/v1/businesses/${businessId}/productivity/rules`),
    productivityUsage: (businessId, from, to) =>
      request(`/v1/businesses/${businessId}/productivity/usage`, { query: { from, to } }),
    setProductivityRule: (businessId, match_type, pattern, category) =>
      request(`/v1/businesses/${businessId}/productivity/rules`, {
        method: "PUT",
        body: { match_type, pattern, category },
      }),
    ...foundationApi(request),
  };
}

/** The roadmap-foundation calls; shared by every transport (web reuses it too). */
export function foundationApi(
  request: AdminRequest,
): Pick<AdminApi, "listAttendance" | "listAgents" | "listAudit"> {
  return {
    listAttendance: (businessId, from, to, departmentId) =>
      request(`/v1/businesses/${businessId}/attendance`, {
        query: { from, to, ...(departmentId ? { department_id: departmentId } : {}) },
      }),
    listAgents: (businessId, departmentId) =>
      request(`/v1/businesses/${businessId}/agents`, { query: departmentId ? { department_id: departmentId } : {} }),
    listAudit: (businessId, limit = 100, offset = 0) =>
      request(`/v1/businesses/${businessId}/audit`, { query: { limit, offset } }),
  };
}
