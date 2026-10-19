import { useTranslation } from "react-i18next";
import { useAdmin } from "../../context";
import { useWorkspace } from "../../hooks";
import { Empty, Notice, Spinner } from "../../ui";
import { PageHead } from "../../components/PageHead";
import { useLoad } from "../../components/useLoad";
import type { Agent } from "../../types";

/** Agents that have not reported for this long are flagged (roadmap: alert when an agent stops reporting). */
const STALE_MS = 24 * 3600 * 1000;
const ONLINE_MS = 10 * 60 * 1000;

function health(a: Agent): "online" | "idle" | "stale" {
  if (!a.last_seen_at) return "stale";
  const age = Date.now() - new Date(a.last_seen_at).getTime();
  return age < ONLINE_MS ? "online" : age < STALE_MS ? "idle" : "stale";
}

/**
 * Roadmap M1/M10 — Agent health (foundation): every installed agent with OS,
 * version and last check-in. Alerts and central rollout build on this later.
 */
export function AgentHealth() {
  const { t } = useTranslation("admin");
  const { api } = useAdmin();
  const { selectedId } = useWorkspace();
  const { data, loading, error } = useLoad(() => api.listAgents(selectedId!), [api, selectedId], !!selectedId);
  const agents = data?.agents ?? [];
  const stale = agents.filter((a) => health(a) === "stale").length;

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <PageHead title={t("agents.title")} subtitle={t("agents.subtitle", { count: agents.length })} />
      {stale > 0 && <Notice kind="info">{t("agents.staleWarning", { count: stale })}</Notice>}
      {error && <Notice kind="danger">{error}</Notice>}
      {loading && <Spinner />}
      {!loading && !error && agents.length === 0 && <Empty>{t("agents.empty")}</Empty>}
      {agents.length > 0 && (
        <div className="bibo-card bibo-card--default admin-tablecard">
          <table className="admin-table">
            <thead>
              <tr>
                <th>{t("common.employee")}</th>
                <th>{t("common.department")}</th>
                <th>{t("agents.device")}</th>
                <th>{t("agents.os")}</th>
                <th>{t("agents.version")}</th>
                <th>{t("agents.lastSeen")}</th>
                <th>{t("agents.status")}</th>
              </tr>
            </thead>
            <tbody>
              {agents.map((a) => {
                const h = health(a);
                return (
                  <tr key={a.device_id}>
                    <td>{a.display_name}</td>
                    <td>{a.department_name || "—"}</td>
                    <td>{a.hostname || "—"}</td>
                    <td>{[a.os, a.os_version].filter(Boolean).join(" ") || "—"}</td>
                    <td className="num">{a.agent_version || "—"}</td>
                    <td className="num">{a.last_seen_at ? new Date(a.last_seen_at).toLocaleString() : "—"}</td>
                    <td>
                      <span className={`notice-${h === "online" ? "success" : h === "idle" ? "info" : "danger"}`} style={{ padding: "2px 8px", borderRadius: 999 }}>
                        {t(`agents.health.${h}`)}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
