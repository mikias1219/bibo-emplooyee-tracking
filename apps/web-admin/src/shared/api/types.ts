// Admin domain types live in the shared @ctracking/admin-ui package (used by both
// the web admin and the desktop app); re-exported here for the rest of the app.
export type {
  BusinessKind,
  ScreenshotMode,
  PrivacyAppCategory,
  Business,
  BusinessSettingsPatch,
  ProductivityCategory,
  RuleMatchType,
  ProductivityRule,
  UsageItem,
  PerfFigures,
  PerfEmployee,
  PerformanceResponse,
} from "@ctracking/admin-ui";

// Shapes mirror the backend contract in docs/11-backend-and-sync.md.

/** Persona of a self-signup owner. Personal users have no account at all. */
export type AccountType = "manager" | "parent";

export interface User {
  id: string;
  email: string;
  username?: string;
  display_name: string;
  account_type: AccountType;
  /** Listed in the backend's SUPER_ADMINS — unlocks the internal Messages area. */
  is_super_admin?: boolean;
}

export interface Tokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface AuthResponse {
  user: User;
  tokens: Tokens;
}

/** Org&Emp directory fields shared by roster rows (empty for legacy accounts). */
export interface DirectoryFields {
  department_id?: string;
  department_name?: string;
  job_title?: string;
  avatar_url?: string;
}

export interface Employee extends DirectoryFields {
  id: string;
  email: string;
  username?: string;
  display_name: string;
}

export interface Department {
  id: string;
  name: string;
  members: number;
}

export interface ReportEmployee extends DirectoryFields {
  id: string;
  email: string;
  username?: string;
  display_name: string;
  role?: "owner" | "employee";
  last_seen: number | null;
  active_today_s: number;
  active_yesterday_s: number;
  screenshots_today: number;
  screenshots_yesterday: number;
  /** 0–100 share of active time with keyboard input; null when no activity today. */
  focus_pct_today: number | null;
}

export interface ActivitySample {
  ts: number;
  app_name: string;
  window_title: string;
  duration_s: number;
}

export interface AppBreakdown {
  app_name: string;
  duration_s: number;
}

export interface ActivityResponse {
  samples: ActivitySample[];
  breakdown: AppBreakdown[];
}

export interface KeystrokeBucket {
  ts_bucket: number;
  count: number;
}

export interface BrowserVisit {
  ts: number;
  url: string;
  page_title: string;
  browser: string;
  duration_s: number;
}

export interface ScreenshotMeta {
  client_uuid: string;
  ts: number;
  byte_size: number;
  width: number;
  height: number;
  display_id: number;
}

export interface ScreenshotsResponse {
  screenshots: ScreenshotMeta[];
  limit: number;
  offset: number;
}

// Thrown by the client for non-2xx responses so UIs can show inline errors.
export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

// ---------- in-app messages (super admin, ticket 145) ----------
export type MessageKind = "announcement" | "survey" | "promo";
export type MessageFieldType = "text" | "textarea" | "radio" | "checkbox" | "select" | "rating" | "nps";
export type MessageEvent = "delivered" | "shown" | "dismissed" | "later" | "cta" | "submitted";

export interface MessageContent {
  title: string;
  body: string;
  image?: string;
  cta_label?: string;
  cta_url?: string;
}
export interface MessageFieldDef {
  id: string;
  type: MessageFieldType;
  required: boolean;
  label: Record<string, string>;
  options?: { value: string; label: Record<string, string> }[];
}
export interface MessageAudience {
  roles?: string[];
  kinds?: string[];
  platforms?: string[];
  locales?: string[];
  min_version?: string;
  max_version?: string;
}
export interface MessageDef {
  id: string;
  kind: MessageKind;
  active: boolean;
  start?: string | null;
  end?: string | null;
  audience: MessageAudience;
  repeat_days: number;
  anonymous?: boolean;
  content: Record<string, MessageContent>;
  fields?: MessageFieldDef[];
  created_at?: string;
  updated_at?: string;
  /** Survey answers so far — once > 0 the questions can only be reworded. */
  response_count?: number;
}
export type EventCounts = Partial<Record<MessageEvent, { total: number; unique: number }>>;
export interface AdminMessageRow extends MessageDef {
  stats: EventCounts;
}
export interface MessageFieldSummary {
  id: string;
  type: MessageFieldType;
  label: string;
  answered: number;
  options?: { value: string; label: string; count: number }[];
  scale?: Record<string, number>;
  average?: number;
  nps?: number;
  texts?: string[];
}
export interface MessageResponseRow {
  answers: Record<string, string | string[] | number>;
  user_id: string | null;
  user_name: string | null; // named surveys only
  user_login: string | null;
  app_version: string;
  platform: string;
  locale: string;
  created_at: string;
}
export interface MessageStats {
  id: string;
  totals: EventCounts;
  daily: { day: string; event: MessageEvent; unique: number }[];
  breakdown: { dim: "locale" | "platform" | "version" | "role"; value: string; event: MessageEvent; unique: number }[];
  message?: MessageDef;
  responses?: MessageResponseRow[];
  summary?: MessageFieldSummary[];
}

