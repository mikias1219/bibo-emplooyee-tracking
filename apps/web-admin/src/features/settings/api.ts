import { request } from "@/shared/api/client";
import type { BusinessSettingsPatch, PrivacyAppCategory } from "@/shared/api/types";

// Workspace capture policy, privacy skip-list and screenshot retention.

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
