import { adminUpdateMessage } from "@/features/internal-messages/api";
import type { EventCounts, MessageDef } from "@/shared/api/types";

export const LOCALES = ["en", "vi", "zh", "ja", "id", "fr", "es"] as const;

export type Status = "active" | "paused" | "scheduled" | "ended";

/** Where a message stands right now, from its switch + schedule. */
export function messageStatus(m: MessageDef, now = Date.now()): Status {
  if (!m.active) return "paused";
  if (m.end && Date.parse(m.end) <= now) return "ended";
  if (m.start && Date.parse(m.start) > now) return "scheduled";
  return "active";
}

export const uniq = (s: EventCounts, e: keyof EventCounts) => s[e]?.unique ?? 0;

/** a ÷ b as a whole percent, or "—" when there's no denominator. */
export function pct(a: number, b: number): string {
  return b > 0 ? `${Math.round((a / b) * 100)}%` : "—";
}

/** Answers → CSV (one column per question; multi-choice joined with "; "). */
export function responsesCsv(
  fieldIds: string[],
  rows: {
    answers: Record<string, unknown>;
    created_at: string;
    app_version: string;
    platform: string;
    locale: string;
    user_name: string | null;
    user_login: string | null;
  }[],
): string {
  const esc = (v: unknown) => {
    const s = Array.isArray(v) ? v.join("; ") : v == null ? "" : String(v);
    return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  const head = ["created_at", "user_name", "user_login", "app_version", "platform", "locale", ...fieldIds];
  const lines = rows.map((r) =>
    [r.created_at, r.user_name ?? "", r.user_login ?? "", r.app_version, r.platform, r.locale, ...fieldIds.map((f) => r.answers[f])]
      .map(esc)
      .join(","),
  );
  return [head.join(","), ...lines].join("\n");
}

/** Flip a message between Active and Paused (saves the whole message unchanged otherwise). */
export function setActive(m: MessageDef, active: boolean) {
  return adminUpdateMessage({ ...m, active });
}
