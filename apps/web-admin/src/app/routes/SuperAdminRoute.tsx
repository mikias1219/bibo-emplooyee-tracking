import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "@/features/auth/AuthContext";

/** Gate for the internal super-admin area (the backend enforces it too). */
export function SuperAdminRoute() {
  const { user } = useAuth();
  return user?.is_super_admin ? <Outlet /> : <Navigate to="/" replace />;
}
