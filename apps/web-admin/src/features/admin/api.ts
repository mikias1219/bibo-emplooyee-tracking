import { type AdminApi, type AdminRequest, foundationApi, viewerTz } from "@ctracking/admin-ui";
import { request } from "@/shared/api/client";
import { demoPerformance, demoRules, demoUsage, isDemo } from "@/shared/api/demo";
import type {
  BusinessSettingsPatch,
  PerformanceResponse,
  PrivacyAppCategory,
  ProductivityCategory,
  ProductivityRule,
  RuleMatchType,
  UsageItem,
} from "@/shared/api/types";
import { listMyBusinesses } from "@/features/businesses/api";

// The web admin's AdminApi: the shared admin screens (@ctracking/admin-ui) call the
// backend through these fetch-based functions (with dev demo data, see isDemo).

export function updateBusinessSettings(id: string, patch: BusinessSettingsPatch) {
  return request<{ status: string }>(`/v1/businesses/${id}/settings`, {
    method: "PATCH",
    body: patch,
  });
}

export function getPrivacyApps() {
  return request<{ categories: PrivacyAppCategory[] }>("/v1/public/screenshot-privacy-apps");
}

export function cleanupScreenshots(id: string, olderThanDays: number) {
  return request<{ deleted_count: number; bytes_freed: number }>(
    `/v1/businesses/${id}/screenshots/cleanup`,
    { method: "POST", query: { older_than_days: olderThanDays } },
  );
}

export function reportPerformance(businessId: string, from: number, to: number, employeeId?: string) {
  if (isDemo()) return Promise.resolve(demoPerformance(from, to));
  return request<PerformanceResponse>(`/v1/businesses/${businessId}/performance`, {
    query: { from, to, tz: viewerTz(), employee_id: employeeId },
  });
}

export function listProductivityRules(businessId: string) {
  if (isDemo()) return Promise.resolve(demoRules());
  return request<{ rules: ProductivityRule[] }>(`/v1/businesses/${businessId}/productivity/rules`);
}

export function productivityUsage(businessId: string, from: number, to: number) {
  if (isDemo()) return Promise.resolve(demoUsage());
  return request<{ apps: UsageItem[]; domains: UsageItem[] }>(
    `/v1/businesses/${businessId}/productivity/usage`,
    { query: { from, to } },
  );
}

export function setProductivityRule(
  businessId: string,
  match_type: RuleMatchType,
  pattern: string,
  category: ProductivityCategory,
) {
  if (isDemo()) return Promise.resolve({ rule: null });
  return request<{ rule: ProductivityRule | null }>(`/v1/businesses/${businessId}/productivity/rules`, {
    method: "PUT",
    body: { match_type, pattern, category },
  });
}

export const webAdminApi: AdminApi = {
  listMyBusinesses,
  updateBusinessSettings,
  getPrivacyApps,
  cleanupScreenshots,
  reportPerformance,
  listProductivityRules,
  productivityUsage,
  setProductivityRule,
  ...foundationApi(request as AdminRequest),
};
