// Admin domain types shared by every client (web admin + desktop).
// Mirror the Go backend JSON (apps/backend/internal/store).

/** A business is a team or a family; `kind` drives member wording (employees/kids). */
export type BusinessKind = "team" | "family";

/**
 * How employee screens are captured. "privacy" (the default) captures only the
 * frontmost window; "normal" captures every display in full. The backend may
 * still return pre-rename values ("full_screen"/"active_window") — normalize
 * before comparing.
 */
export type ScreenshotMode = "privacy" | "normal";

/** One category of the backend's curated sensitive-app list. */
export interface PrivacyAppCategory {
  key: string;
  apps: string[];
}

export interface Business {
  id: string;
  name: string;
  kind: BusinessKind;
  owner_user_id: string;
  screenshot_retention_days: number | null;
  screenshot_interval_s: number;
  idle_threshold_s: number;
  allow_employee_override: boolean;
  screenshot_mode: string; // normalize to ScreenshotMode before comparing
  screenshot_skip_apps: string[];
  productive_target_s: number; // expected productive seconds per working day
}

export interface BusinessSettingsPatch {
  screenshot_retention_days?: number | null;
  screenshot_interval_s?: number;
  idle_threshold_s?: number;
  allow_employee_override?: boolean;
  screenshot_mode?: ScreenshotMode;
  screenshot_skip_apps?: string[];
  productive_target_s?: number;
}

// ---------- productivity / performance ----------
export type ProductivityCategory = "productive" | "unproductive" | "neutral" | "unclassified";

export type RuleMatchType = "app" | "domain";

export interface ProductivityRule {
  id: string;
  match_type: RuleMatchType;
  pattern: string;
  category: Exclude<ProductivityCategory, "unclassified">;
}

export interface UsageItem {
  name: string;
  duration_s: number;
  category: ProductivityCategory;
}

/** Time per category plus derived figures; percentages are null when undefined. */
export interface PerfFigures {
  active_s: number;
  productive_s: number;
  unproductive_s: number;
  neutral_s: number;
  unclassified_s: number;
  days_active: number;
  /** productive ÷ classified time (0–100). */
  score_pct: number | null;
  /** classified ÷ active time (0–100). */
  coverage_pct: number | null;
  /** productive ÷ (daily target × days active); can exceed 100. */
  target_pct: number | null;
}

export interface PerfEmployee extends PerfFigures {
  id: string;
  display_name: string;
  email: string;
  username: string;
  role: "owner" | "employee";
}

export interface PerformanceResponse {
  target_per_day_s: number;
  team: PerfFigures;
  employees: PerfEmployee[];
  days: (PerfFigures & { day: string })[];
}

// ── Roadmap foundation (M6 attendance, M1/M10 agents, M9 audit, M10 policies) ──

/** One employee's observed attendance for one local day. */
export interface AttendanceDay {
  user_id: string;
  display_name: string;
  department_name: string;
  day: string; // YYYY-MM-DD (organization time zone)
  first_activity: number; // unix seconds
  last_activity: number;
  active_s: number;
  observed_s: number; // last - first; idle = observed - active
}

/** One installed desktop agent (Agent health). */
export interface Agent {
  device_id: string;
  user_id: string;
  display_name: string;
  email: string;
  department_name: string;
  hostname: string;
  os: string; // windows | macos | linux
  os_version: string;
  agent_version: string;
  last_seen_at: string | null; // ISO time
}

/** One audit-log entry. */
export interface AuditEntry {
  id: number;
  actor_user_id: string;
  actor_name: string;
  action: string; // report.view | screenshot.view | settings.update | policy.update
  target_type: string;
  target_id: string;
  meta?: unknown;
  created_at: string;
}

/** Per-department policy override; null fields inherit the organization policy. */
export interface DepartmentPolicy {
  department_id: string;
  work_start: string | null; // "HH:MM"
  work_end: string | null;
  work_days: number[] | null; // ISO 1=Mon … 7=Sun
  idle_threshold_s: number | null;
  screenshot_interval_s: number | null;
  screenshot_skip_apps: string[] | null;
  screenshot_skip_domains: string[] | null;
}
