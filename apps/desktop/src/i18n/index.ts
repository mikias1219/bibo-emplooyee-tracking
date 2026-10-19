// English-only UI. To add a language later: add locales/<lang>/*.json, register
// them in `resources`, and add the same for @ctracking/admin-ui (adminResources).
import i18n from "i18next";
import { ADMIN_NAMESPACES, adminResources } from "@ctracking/admin-ui";
import { initReactI18next } from "react-i18next";

import enCommon from "./locales/en/common.json";
import enAuth from "./locales/en/auth.json";
import enOnboarding from "./locales/en/onboarding.json";
import enWelcome from "./locales/en/welcome.json";
import enSettings from "./locales/en/settings.json";
import enPermissions from "./locales/en/permissions.json";
import enScreens from "./locales/en/screens.json";
import enMedia from "./locales/en/media.json";

const resources = {
  en: { common: enCommon, auth: enAuth, onboarding: enOnboarding, welcome: enWelcome, settings: enSettings, permissions: enPermissions, screens: enScreens, media: enMedia, ...adminResources.en },
};

i18n
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    lng: "en",
    ns: ["common", "auth", "onboarding", "welcome", "settings", "permissions", "screens", "media", ...ADMIN_NAMESPACES],
    defaultNS: "common",
    interpolation: { escapeValue: false },
  });

/** The active locale, suitable for Intl APIs. */
export function activeLocale(): string {
  return i18n.resolvedLanguage || i18n.language || "en";
}

export default i18n;
