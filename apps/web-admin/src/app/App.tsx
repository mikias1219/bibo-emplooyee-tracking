import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AppShell } from "./layout/AppShell";
import { ProtectedRoute } from "./routes/ProtectedRoute";
import { AuthProvider } from "@/features/auth/AuthContext";
import { BusinessProvider } from "@/features/businesses/useBusinesses";
import { ThemeProvider } from "@/shared/theme/ThemeProvider";
import { Dashboard } from "@/features/dashboard/Dashboard";
import { EmployeeDetail } from "@/features/employees/detail/EmployeeDetail";
import { Employees } from "@/features/employees/Employees";
import { SignIn } from "@/features/auth/SignIn";
import { AgentHealth, Attendance, AuditLog, Performance, ProductivityRules, WorkspaceSettings } from "@ctracking/admin-ui";
import { AdminScope } from "@/features/admin/AdminScope";
import { SuperAdminRoute } from "./routes/SuperAdminRoute";
import { MessagesAdmin } from "@/features/internal-messages/MessagesAdmin";
import { MessageEditor } from "@/features/internal-messages/MessageEditor";
import { MessageStats } from "@/features/internal-messages/MessageStats";

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter basename={import.meta.env.BASE_URL.replace(/\/+$/, "")}>
          <Routes>
            <Route path="/login" element={<SignIn />} />
            {/* Accounts come from the Selamnew Org&Emp directory — no self-signup. */}
            <Route path="/signup" element={<Navigate to="/login" replace />} />
            <Route element={<ProtectedRoute />}>
              <Route
                element={
                  <BusinessProvider>
                    <AppShell />
                  </BusinessProvider>
                }
              >
                <Route index element={<Dashboard />} />
                <Route path="employees" element={<Employees />} />
                <Route path="employees/:id" element={<EmployeeDetail />} />
                <Route path="performance" element={<AdminScope view="performance"><Performance /></AdminScope>} />
                <Route path="performance/rules" element={<AdminScope view="rules"><ProductivityRules /></AdminScope>} />
                <Route path="settings" element={<AdminScope view="settings"><WorkspaceSettings /></AdminScope>} />
                <Route path="attendance" element={<AdminScope view="attendance"><Attendance /></AdminScope>} />
                <Route path="agents" element={<AdminScope view="agents"><AgentHealth /></AdminScope>} />
                <Route path="audit" element={<AdminScope view="audit"><AuditLog /></AdminScope>} />
                <Route element={<SuperAdminRoute />}>
                  <Route path="internal/messages" element={<MessagesAdmin />} />
                  <Route path="internal/messages/new" element={<MessageEditor />} />
                  <Route path="internal/messages/:id" element={<MessageStats />} />
                  <Route path="internal/messages/:id/edit" element={<MessageEditor />} />
                </Route>
              </Route>
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  );
}
