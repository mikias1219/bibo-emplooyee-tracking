// Small formatting helpers. Locale-aware via the active i18n language + Intl.
// Tabular figures handle alignment in CSS (.num).
import i18n, { activeLocale } from "@/i18n";

// Pure helpers are shared with the desktop app via @ctracking/admin-ui.
export { dayRangeToUnix, daysAgoIso, fmtBytes, fmtDuration, isoDate } from "@ctracking/admin-ui";

export function fmtRelative(unixSeconds: number | null): string {
  if (!unixSeconds) return i18n.t("time.never");
  const diff = Date.now() / 1000 - unixSeconds;
  if (diff < 60) return i18n.t("time.justNow");
  if (diff < 3600) return i18n.t("time.minutesAgo", { count: Math.floor(diff / 60) });
  if (diff < 86400) return i18n.t("time.hoursAgo", { count: Math.floor(diff / 3600) });
  if (diff < 7 * 86400) return i18n.t("time.daysAgo", { count: Math.floor(diff / 86400) });
  return new Date(unixSeconds * 1000).toLocaleDateString(activeLocale());
}

export function fmtTime(unixSeconds: number): string {
  return new Date(unixSeconds * 1000).toLocaleString(activeLocale(), {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

