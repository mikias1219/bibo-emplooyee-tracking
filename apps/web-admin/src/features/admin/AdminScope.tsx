import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { AdminProvider, type AdminView } from "@ctracking/admin-ui";
import { useAuth } from "@/features/auth/AuthContext";
import { useBusinesses } from "@/features/businesses/useBusinesses";
import { webAdminApi } from "./api";

const ROUTES: Record<AdminView, string> = {
  performance: "/performance",
  rules: "/performance/rules",
  settings: "/settings",
  attendance: "/attendance",
  agents: "/agents",
  audit: "/audit",
};

/** Hosts a shared admin screen (@ctracking/admin-ui) inside the web admin:
 *  fetch transport, the selected workspace, the signed-in user and routing. */
export function AdminScope({ view, children }: { view: AdminView; children: ReactNode }) {
  const nav = useNavigate();
  const { user } = useAuth();
  const { selected, selectedId, loading, reload } = useBusinesses();
  return (
    <AdminProvider
      value={{
        api: webAdminApi,
        business: selected,
        businessLoading: loading,
        reloadBusiness: reload,
        user: user ? { email: user.email || user.username || "", display_name: user.display_name } : null,
        view,
        navigate: (v) => nav(ROUTES[v]),
        openEmployee: (id) => nav(`/employees/${id}?business=${selectedId}`),
      }}
    >
      {children}
    </AdminProvider>
  );
}
