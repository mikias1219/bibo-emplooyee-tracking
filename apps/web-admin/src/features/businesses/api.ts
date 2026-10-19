import { request } from "@/shared/api/client";
import { demoBusinesses, isDemo } from "@/shared/api/demo";
import type { Business } from "@/shared/api/types";

// Workspaces the signed-in owner manages (one per Selamnew organization).

export function listMyBusinesses() {
  if (isDemo()) return Promise.resolve({ businesses: demoBusinesses });
  return request<{ businesses: Business[] }>("/v1/businesses/mine");
}
