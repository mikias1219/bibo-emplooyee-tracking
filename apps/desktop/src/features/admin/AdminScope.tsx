import { useCallback, useEffect, useState, type ReactNode } from "react";
import { AdminProvider, type AdminView, type Business } from "@ctracking/admin-ui";
import { desktopAdminApi } from "./api";

/** Hosts a shared admin screen (@ctracking/admin-ui) in the desktop app: Tauri
 *  transport, the full workspace record, the signed-in user and screen navigation. */
export function AdminScope({
  businessId,
  view,
  user,
  onNavigate,
  onOpenEmployee,
  children,
}: {
  businessId: string;
  view: AdminView;
  user: { email: string; display_name: string } | null;
  onNavigate: (view: AdminView) => void;
  onOpenEmployee: (employeeId: string) => void;
  children: ReactNode;
}) {
  const [business, setBusiness] = useState<Business | null>(null);
  const [loading, setLoading] = useState(true);

  // The picker only knows id/name/kind; the shared screens need the full policy.
  const reload = useCallback(async () => {
    setLoading(true);
    try {
      const { businesses } = await desktopAdminApi.listMyBusinesses();
      setBusiness(businesses.find((b) => b.id === businessId) ?? null);
    } catch {
      setBusiness(null);
    } finally {
      setLoading(false);
    }
  }, [businessId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return (
    <AdminProvider
      value={{
        api: desktopAdminApi,
        business,
        businessLoading: loading,
        reloadBusiness: reload,
        user,
        view,
        navigate: onNavigate,
        openEmployee: onOpenEmployee,
      }}
    >
      {children}
    </AdminProvider>
  );
}
