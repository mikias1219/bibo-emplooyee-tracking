// Pure formatting/date helpers shared by both clients (no i18n dependency).

export function fmtDuration(seconds: number): string {
  if (!seconds || seconds < 0) return "0m";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 0) return m > 0 ? `${h}h ${m}m` : `${h}h`;
  if (m > 0) return `${m}m`;
  return `${seconds}s`;
}

export function fmtBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

// Unix-second bounds for a YYYY-MM-DD date input (local time, inclusive end).
export function dayRangeToUnix(fromDate: string, toDate: string): { from: number; to: number } {
  const from = Math.floor(new Date(`${fromDate}T00:00:00`).getTime() / 1000);
  const to = Math.floor(new Date(`${toDate}T23:59:59`).getTime() / 1000);
  return { from, to };
}

export function isoDate(d: Date): string {
  // LOCAL date, not toISOString() (UTC) — for a UTC+7 viewer the UTC date is
  // still yesterday between local midnight and 7am, which made the default day
  // view (and its screenshot gallery) query an already-ended window.
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

export function daysAgoIso(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return isoDate(d);
}
