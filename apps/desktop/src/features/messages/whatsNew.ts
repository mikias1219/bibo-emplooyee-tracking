// "What's new" notes bundled with the app (ticket 145), keyed by app version then
// locale ("en" is the fallback). Shown once, the first launch after an update to a
// version listed here — works offline and in personal mode. Add an entry when
// cutting a release; versions without an entry show nothing.
export const WHATS_NEW: Record<string, Record<string, string[]>> = {
  "1.6.0": {
    en: [
      "See reports and screenshots for everyone you track right in the desktop app — no web dashboard needed",
      "Fixed only the latest 60 screenshots being viewable — you can now browse them all",
      "Fixed the web dashboard appearing blank or broken in Firefox and with ad blockers on",
      "Fixed some admin screens showing placeholder values instead of real data",
    ],
  },
  "1.5.1": {
    en: [
      "Screenshot privacy mode: capture only the active window, and skip the apps you choose.",
      "Dashboard fixes and a new full-screen screenshot viewer.",
    ],
  },
};

export function whatsNewFor(version: string, locale: string): string[] | null {
  const entry = WHATS_NEW[version];
  if (!entry) return null;
  return entry[locale] ?? entry.en ?? null;
}
