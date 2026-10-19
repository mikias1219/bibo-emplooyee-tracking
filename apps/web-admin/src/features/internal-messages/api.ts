import { request } from "@/shared/api/client";
import type { AdminMessageRow, EventCounts, MessageDef, MessageStats } from "@/shared/api/types";

// In-app messages (super-admin only).

export function adminListMessages() {
  return request<{ messages: AdminMessageRow[]; whatsnew: { id: string; stats: EventCounts }[] }>(
    "/v1/admin/messages",
  );
}

export function adminGetMessage(id: string) {
  return request<MessageDef>(`/v1/admin/messages/${encodeURIComponent(id)}`);
}

export function adminCreateMessage(m: MessageDef) {
  return request<{ id: string }>("/v1/admin/messages", { method: "POST", body: m });
}

export function adminUpdateMessage(m: MessageDef) {
  return request<{ id: string }>(`/v1/admin/messages/${encodeURIComponent(m.id)}`, { method: "PUT", body: m });
}

export function adminDeleteMessage(id: string) {
  return request<{ ok: boolean }>(`/v1/admin/messages/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function adminMessageStats(id: string) {
  return request<MessageStats>(`/v1/admin/messages/${encodeURIComponent(id)}/stats`);
}
