import { useTranslation } from "react-i18next";
import { useAdmin } from "./context";

/** The managed workspace, in the shape the admin screens use. */
export function useWorkspace() {
  const a = useAdmin();
  return {
    businesses: a.business ? [a.business] : [],
    selected: a.business,
    selectedId: a.business?.id ?? null,
    loading: a.businessLoading,
    reload: a.reloadBusiness,
  };
}

/** Member wording. Workspaces come from Selamnew organizations, so always "team". */
export function useMemberTerms() {
  const { t } = useTranslation("admin");
  return {
    one: t("terms.one"),
    many: t("terms.many"),
    lowerOne: t("terms.lowerOne"),
    lowerMany: t("terms.lowerMany"),
  };
}
