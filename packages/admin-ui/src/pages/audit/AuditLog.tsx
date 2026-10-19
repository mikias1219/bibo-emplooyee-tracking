import { useTranslation } from "react-i18next";
import { useAdmin } from "../../context";
import { useWorkspace } from "../../hooks";
import { Empty, Notice, Spinner } from "../../ui";
import { PageHead } from "../../components/PageHead";
import { useLoad } from "../../components/useLoad";

/**
 * Roadmap M9 — Audit log (foundation, admins only): who viewed which report or
 * screenshot and who changed settings or policies, newest first.
 */
export function AuditLog() {
  const { t } = useTranslation("admin");
  const { api } = useAdmin();
  const { selectedId } = useWorkspace();
  const { data, loading, error } = useLoad(() => api.listAudit(selectedId!, 200, 0), [api, selectedId], !!selectedId);
  const entries = data?.entries ?? [];

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <PageHead title={t("audit.title")} subtitle={t("audit.subtitle")} />
      {error && <Notice kind="danger">{error}</Notice>}
      {loading && <Spinner />}
      {!loading && !error && entries.length === 0 && <Empty>{t("audit.empty")}</Empty>}
      {entries.length > 0 && (
        <div className="bibo-card bibo-card--default admin-tablecard">
          <table className="admin-table">
            <thead>
              <tr>
                <th>{t("audit.when")}</th>
                <th>{t("audit.who")}</th>
                <th>{t("audit.action")}</th>
                <th>{t("audit.target")}</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id}>
                  <td className="num">{new Date(e.created_at).toLocaleString()}</td>
                  <td>{e.actor_name || "—"}</td>
                  <td>{t(`audit.actions.${e.action}`, { defaultValue: e.action })}</td>
                  <td>{e.target_type ? `${e.target_type} · ${e.target_id}` : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
