import { request } from "@/shared/api/client";
import { demoPerformance, demoRules, demoUsage, isDemo } from "@/shared/api/demo";
import type { PerformanceResponse, ProductivityCategory, ProductivityRule, RuleMatchType, UsageItem } from "@/shared/api/types";

// Productivity rules and performance reports.

function viewerTz(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
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
