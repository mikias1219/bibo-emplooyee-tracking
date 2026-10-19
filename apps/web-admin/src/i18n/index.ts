import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

import enCommon from "./locales/en/common.json";
import enAuth from "./locales/en/auth.json";
import enDashboard from "./locales/en/dashboard.json";
import enSettings from "./locales/en/settings.json";
import enUi from "./locales/en/ui.json";
import enReports from "./locales/en/reports.json";
import enPerformance from "./locales/en/performance.json";

import zhCommon from "./locales/zh/common.json";
import zhAuth from "./locales/zh/auth.json";
import zhDashboard from "./locales/zh/dashboard.json";
import zhSettings from "./locales/zh/settings.json";
import zhUi from "./locales/zh/ui.json";
import zhReports from "./locales/zh/reports.json";
import zhPerformance from "./locales/zh/performance.json";

import jaCommon from "./locales/ja/common.json";
import jaAuth from "./locales/ja/auth.json";
import jaDashboard from "./locales/ja/dashboard.json";
import jaSettings from "./locales/ja/settings.json";
import jaUi from "./locales/ja/ui.json";
import jaReports from "./locales/ja/reports.json";
import jaPerformance from "./locales/ja/performance.json";

import viCommon from "./locales/vi/common.json";
import viAuth from "./locales/vi/auth.json";
import viDashboard from "./locales/vi/dashboard.json";
import viSettings from "./locales/vi/settings.json";
import viUi from "./locales/vi/ui.json";
import viReports from "./locales/vi/reports.json";
import viPerformance from "./locales/vi/performance.json";

import idCommon from "./locales/id/common.json";
import idAuth from "./locales/id/auth.json";
import idDashboard from "./locales/id/dashboard.json";
import idSettings from "./locales/id/settings.json";
import idUi from "./locales/id/ui.json";
import idReports from "./locales/id/reports.json";
import idPerformance from "./locales/id/performance.json";

import frCommon from "./locales/fr/common.json";
import frAuth from "./locales/fr/auth.json";
import frDashboard from "./locales/fr/dashboard.json";
import frSettings from "./locales/fr/settings.json";
import frUi from "./locales/fr/ui.json";
import frReports from "./locales/fr/reports.json";
import frPerformance from "./locales/fr/performance.json";

import esCommon from "./locales/es/common.json";
import esAuth from "./locales/es/auth.json";
import esDashboard from "./locales/es/dashboard.json";
import esSettings from "./locales/es/settings.json";
import esUi from "./locales/es/ui.json";
import esReports from "./locales/es/reports.json";
import esPerformance from "./locales/es/performance.json";

/** Supported locales (en is the source of truth). `zh` = Simplified Chinese. */
export const LOCALES = [
  { code: "en", label: "English", flag: "🇺🇸" },
  { code: "zh", label: "中文", flag: "🇨🇳" },
  { code: "ja", label: "日本語", flag: "🇯🇵" },
  { code: "vi", label: "Tiếng Việt", flag: "🇻🇳" },
  { code: "id", label: "Bahasa Indonesia", flag: "🇮🇩" },
  { code: "fr", label: "Français", flag: "🇫🇷" },
  { code: "es", label: "Español", flag: "🇪🇸" },
] as const;

export type LocaleCode = (typeof LOCALES)[number]["code"];

const resources = {
  en: { common: enCommon, auth: enAuth, dashboard: enDashboard, settings: enSettings, ui: enUi, reports: enReports, performance: enPerformance },
  zh: { common: zhCommon, auth: zhAuth, dashboard: zhDashboard, settings: zhSettings, ui: zhUi, reports: zhReports, performance: zhPerformance },
  ja: { common: jaCommon, auth: jaAuth, dashboard: jaDashboard, settings: jaSettings, ui: jaUi, reports: jaReports, performance: jaPerformance },
  vi: { common: viCommon, auth: viAuth, dashboard: viDashboard, settings: viSettings, ui: viUi, reports: viReports, performance: viPerformance },
  id: { common: idCommon, auth: idAuth, dashboard: idDashboard, settings: idSettings, ui: idUi, reports: idReports, performance: idPerformance },
  fr: { common: frCommon, auth: frAuth, dashboard: frDashboard, settings: frSettings, ui: frUi, reports: frReports, performance: frPerformance },
  es: { common: esCommon, auth: esAuth, dashboard: esDashboard, settings: esSettings, ui: esUi, reports: esReports, performance: esPerformance },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    supportedLngs: LOCALES.map((l) => l.code),
    // Browser sends e.g. "fr-FR" / "zh-CN"; collapse to the base language.
    load: "languageOnly",
    nonExplicitSupportedLngs: true,
    ns: ["common", "auth", "dashboard", "settings", "ui", "reports", "performance"],
    defaultNS: "common",
    interpolation: { escapeValue: false },
    detection: {
      // Saved choice first, then the browser/OS language.
      order: ["localStorage", "navigator"],
      lookupLocalStorage: "locale",
      caches: ["localStorage"],
    },
  });

/** The active locale, suitable for Intl APIs. */
export function activeLocale(): string {
  return i18n.resolvedLanguage || i18n.language || "en";
}

export default i18n;
