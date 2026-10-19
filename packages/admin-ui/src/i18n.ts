// Translations for the shared admin screens, merged into each client's i18next
// resources. Namespaces: "admin", "admin-performance", "admin-settings".

import enAdmin from "./locales/en/admin.json";
import enPerformance from "./locales/en/performance.json";
import enSettings from "./locales/en/settings.json";

export const ADMIN_NAMESPACES = ["admin", "admin-performance", "admin-settings"] as const;

export const adminResources: Record<string, Record<(typeof ADMIN_NAMESPACES)[number], object>> = {
  en: { admin: enAdmin, "admin-performance": enPerformance, "admin-settings": enSettings },
};
