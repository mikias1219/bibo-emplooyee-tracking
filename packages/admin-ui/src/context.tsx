import { createContext, useContext, type ReactNode } from "react";
import type { AdminApi } from "./api";
import type { Business } from "./types";

/** Screens inside the shared admin area. Each client maps these to its own routing. */
export type AdminView = "performance" | "rules" | "settings" | "attendance" | "agents" | "audit";

export interface AdminContextValue {
  api: AdminApi;
  /** The workspace being managed (one per Selamnew organization). */
  business: Business | null;
  businessLoading: boolean;
  /** Re-fetch the workspace after a settings change. */
  reloadBusiness: () => Promise<void>;
  /** Signed-in owner, shown read-only on the settings page. */
  user: { email: string; display_name: string } | null;
  view: AdminView;
  navigate: (view: AdminView) => void;
  /** Open one member's report (web: route, desktop: Members → detail). */
  openEmployee: (employeeId: string) => void;
}

const AdminContext = createContext<AdminContextValue | null>(null);

/** Provides the client-specific wiring (transport, workspace, navigation) to the shared admin screens. */
export function AdminProvider({ value, children }: { value: AdminContextValue; children: ReactNode }) {
  return (
    <AdminContext.Provider value={value}>
      {/* styles.css is scoped to this wrapper so it never restyles the host app */}
      <div className="wi-admin">{children}</div>
    </AdminContext.Provider>
  );
}

export function useAdmin(): AdminContextValue {
  const ctx = useContext(AdminContext);
  if (!ctx) throw new Error("useAdmin must be used inside <AdminProvider>");
  return ctx;
}
