import { request } from "@/shared/api/client";
import { demoActivity, demoBrowser, demoEmployees, demoKeystrokes, demoRoster, demoScreenshots, isDemo } from "@/shared/api/demo";
import type { ActivityResponse, BrowserVisit, Department, Employee, KeystrokeBucket, ReportEmployee, ScreenshotsResponse } from "@/shared/api/types";

// Directory-backed roster (Org&Emp sync, departments) and per-employee reports.

export function listDepartments(businessId: string) {
  if (isDemo()) return Promise.resolve({ departments: [] as Department[] });
  return request<{ departments: Department[] }>(`/v1/businesses/${businessId}/departments`);
}

export function syncDirectory(businessId: string) {
  return request<{ status: string; members: number }>(
    `/v1/businesses/${businessId}/directory/sync`,
    { method: "POST" },
  );
}

export function listBusinessEmployees(businessId: string) {
  if (isDemo()) return Promise.resolve({ employees: demoEmployees() });
  return request<{ employees: Employee[] }>(`/v1/businesses/${businessId}/employees`);
}

export function reportEmployees(businessId: string, departmentId?: string) {
  if (isDemo()) return Promise.resolve({ employees: demoRoster() });
  const midnight = new Date();
  midnight.setHours(0, 0, 0, 0);
  return request<{ employees: ReportEmployee[] }>("/v1/reports/employees", {
    query: {
      business_id: businessId,
      day_start: Math.floor(midnight.getTime() / 1000),
      ...(departmentId ? { department_id: departmentId } : {}),
    },
  });
}

export function reportActivity(employeeId: string, from: number, to: number) {
  if (isDemo()) return Promise.resolve(demoActivity(employeeId));
  return request<ActivityResponse>(`/v1/reports/employees/${employeeId}/activity`, {
    query: { from, to },
  });
}

export function reportKeystrokes(employeeId: string, from: number, to: number) {
  if (isDemo()) return Promise.resolve(demoKeystrokes(employeeId));
  return request<{ buckets: KeystrokeBucket[] }>(
    `/v1/reports/employees/${employeeId}/keystrokes`,
    { query: { from, to } },
  );
}

export function reportBrowser(employeeId: string, from: number, to: number) {
  if (isDemo()) return Promise.resolve(demoBrowser(employeeId));
  return request<{ visits: BrowserVisit[] }>(`/v1/reports/employees/${employeeId}/browser`, {
    query: { from, to },
  });
}

export function reportScreenshots(
  employeeId: string,
  from: number,
  to: number,
  limit = 60,
  offset = 0,
) {
  if (isDemo()) return Promise.resolve(demoScreenshots(employeeId));
  return request<ScreenshotsResponse>(`/v1/reports/employees/${employeeId}/screenshots`, {
    query: { from, to, limit, offset },
  });
}
