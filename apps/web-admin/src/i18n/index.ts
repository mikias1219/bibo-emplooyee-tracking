// English-only UI. To add a language later: add locales/<lang>/*.json, register
// them in `resources`, and add the same for @ctracking/admin-ui (adminResources).
import i18n from "i18next";
import { ADMIN_NAMESPACES, adminResources } from "@ctracking/admin-ui";
import { initReactI18next } from "react-i18next";

import enCommon from "./locales/en/common.json";
import enAuth from "./locales/en/auth.json";
import enDashboard from "./locales/en/dashboard.json";
import enUi from "./locales/en/ui.json";
import enReports from "./locales/en/reports.json";

const resources = {
  en: { common: enCommon, auth: enAuth, dashboard: enDashboard, ui: enUi, reports: enReports, ...adminResources.en },
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    lng: "en",
    ns: ["common", "auth", "dashboard", "ui", "reports", ...ADMIN_NAMESPACES],
    defaultNS: "common",
    interpolation: { escapeValue: false },
  });

/** The active locale, suitable for Intl APIs. */
export function activeLocale(): string {
  return i18n.resolvedLanguage || i18n.language || "en";
}

export default i18n;
