import type { RosterEntry } from "./types";

// Display helpers shared by the admin screens (team overview, members, detail).

export type MemberStatus = "active" | "idle" | "offline";

/** Presence from the last device heartbeat: <5 min active, <30 min idle. */
export function memberStatus(lastSeen: number | null): MemberStatus {
  if (!lastSeen) return "offline";
  const age = Date.now() / 1000 - lastSeen;
  if (age < 5 * 60) return "active";
  if (age < 30 * 60) return "idle";
  return "offline";
}

/** True when the row is the signed-in user. An organization can have several
 *  owners, so this matches the email — never the role. */
export function isSelf(member: Pick<RosterEntry, "email">, selfEmail: string): boolean {
  return !!selfEmail && member.email.toLowerCase() === selfEmail.toLowerCase();
}

/** The viewer's local midnight (unix seconds) — sent as the roster's `day_start`
 *  so "today" matches the per-employee detail views (which use local days). */
export function localDayStart(): number {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return Math.floor(d.getTime() / 1000);
}

export const initials = (name: string) =>
  name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]?.toUpperCase()).join("") || "?";

export const AVATAR_PALETTE = [
  { bg: "var(--accent-weak)", fg: "var(--accent)" },
  { bg: "color-mix(in srgb, var(--data-mint) 20%, transparent)", fg: "var(--data-mint)" },
  { bg: "color-mix(in srgb, var(--data-rose) 18%, transparent)", fg: "var(--data-rose)" },
  { bg: "color-mix(in srgb, var(--data-amber) 22%, transparent)", fg: "var(--data-amber)" },
  { bg: "color-mix(in srgb, var(--data-sky) 20%, transparent)", fg: "var(--data-sky)" },
];
