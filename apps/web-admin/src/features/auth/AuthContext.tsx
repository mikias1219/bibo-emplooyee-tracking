import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { subscribeLogout } from "@/shared/api/client";
import { getMe } from "@/features/auth/api";
import { tokenStore } from "@/shared/api/tokenStore";
import { isDemo } from "@/shared/api/demo";
import type { User } from "@/shared/api/types";
import { Sentry } from "@/shared/lib/sentry";

// Dev demo mode: a signed-in "owner" so the protected pages render without a
// backend. id matches the roster owner in demo.ts so their row shows "You".
const DEMO_USER: User = {
  id: "demo-owner",
  email: "brian@home.app",
  display_name: "Brian Nguyen",
  account_type: "parent",
};

interface AuthState {
  user: User | null;
  isAuthed: boolean;
  setSession: (user: User) => void;
  logout: () => void;
}

const AuthCtx = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(
    () => (isDemo() ? tokenStore.getUser() ?? DEMO_USER : tokenStore.getUser()),
  );
  const [isAuthed, setIsAuthed] = useState<boolean>(() => isDemo() || tokenStore.isAuthed());

  useEffect(() => {
    // The API client emits a logout when a refresh fails.
    return subscribeLogout(() => {
      setUser(null);
      setIsAuthed(false);
    });
  }, []);

  // Refresh the stored profile once per load (e.g. to pick up is_super_admin for a
  // session that predates it). Failures keep the stored copy.
  useEffect(() => {
    if (isDemo() || !tokenStore.isAuthed()) return;
    getMe()
      .then((u) => {
        tokenStore.setUser(u);
        setUser(u);
      })
      .catch(() => {});
  }, []);

  // Attach (or clear) the signed-in user on every Sentry event. Covers login,
  // logout, refresh-failure, and a session restored from storage on load.
  useEffect(() => {
    if (user) {
      Sentry.setUser({ id: user.id, email: user.email, username: user.username });
    } else {
      Sentry.setUser(null);
    }
  }, [user]);

  const value: AuthState = {
    user,
    isAuthed,
    setSession: (u) => {
      tokenStore.setUser(u);
      setUser(u);
      setIsAuthed(true);
    },
    logout: () => {
      tokenStore.clear();
      setUser(null);
      setIsAuthed(false);
    },
  };

  return <AuthCtx.Provider value={value}>{children}</AuthCtx.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthCtx);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
