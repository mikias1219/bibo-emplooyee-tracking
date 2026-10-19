import { request } from "@/shared/api/client";
import { tokenStore } from "@/shared/api/tokenStore";
import type { AuthResponse, User } from "@/shared/api/types";

// Session: sign-in with the Selamnew organization account and /me.

export async function login(identifier: string, password: string, business_id?: string) {
  const res = await request<AuthResponse>("/v1/auth/login", {
    method: "POST",
    auth: false,
    body: { identifier, password, business_id },
  });
  tokenStore.setSession(res.tokens, res.user);
  return res;
}

export function getMe() {
  return request<User>("/v1/me");
}
