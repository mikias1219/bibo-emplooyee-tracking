import { useEffect, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { listBusinessEmployees, listDepartments, reportEmployees, syncDirectory } from "@/features/employees/api";
import { ApiError, type Department, type Employee } from "@/shared/api/types";
import { Empty, Notice, Spinner } from "@/shared/ui/index";
import { useBusinesses } from "@/features/businesses/useBusinesses";
import { memberTerms } from "@/shared/lib/terms";
import { useAuth } from "@/features/auth/AuthContext";
import { memberStatus } from "@/features/dashboard/Dashboard";

// ── display-only helpers (mirror the Dashboard roster look) ──────────
const svg = (children: ReactNode) => (
  <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
    {children}
  </svg>
);
const IconArrowRight = svg(<><path d="M5 12h14" /><path d="m12 5 7 7-7 7" /></>);
const IconRefresh = svg(<><path d="M3 12a9 9 0 0 1 15-6.7L21 8" /><path d="M21 3v5h-5" /><path d="M21 12a9 9 0 0 1-15 6.7L3 16" /><path d="M8 16H3v5" /></>);

const AVATAR_PALETTE = [
  { bg: "var(--info-soft)", fg: "var(--info)" },
  { bg: "var(--positive-soft)", fg: "var(--positive)" },
  { bg: "color-mix(in srgb, var(--data-rose) 18%, transparent)", fg: "var(--data-rose)" },
  { bg: "color-mix(in srgb, var(--data-amber) 22%, transparent)", fg: "var(--data-amber)" },
];
const initials = (name: string) =>
  name.split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0]?.toUpperCase()).join("") || "?";

/**
 * Directory-backed roster. People, departments and roles are owned by the Selamnew
 * Org&Emp directory; owners can filter by department and trigger a re-sync, but
 * never add or remove people here.
 */
export function Employees() {
  const { t } = useTranslation("dashboard");
  const { user } = useAuth();
  const { businesses, selected, selectedId, loading: bizLoading } = useBusinesses();

  const [employees, setEmployees] = useState<Employee[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [department, setDepartment] = useState("");
  // Presence comes from the roster report (the members list has no last_seen);
  // members missing from it — or a failed load — show no dot rather than a guess.
  const [lastSeen, setLastSeen] = useState<Record<string, number | null>>({});
  const [loading, setLoading] = useState(false);
  const [listError, setListError] = useState<string | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [syncMsg, setSyncMsg] = useState<{ kind: "success" | "danger"; text: string } | null>(null);

  const terms = memberTerms(selected?.kind);

  function load(id: string) {
    setLoading(true);
    setListError(null);
    listBusinessEmployees(id)
      .then((r) => setEmployees(r.employees))
      .catch(() => setListError(t("employees.errorLoadMembers", { members: terms.lowerMany })))
      .finally(() => setLoading(false));
    listDepartments(id)
      .then((r) => setDepartments(r.departments))
      .catch(() => setDepartments([]));
    reportEmployees(id)
      .then((r) => setLastSeen(Object.fromEntries(r.employees.map((e) => [e.id, e.last_seen]))))
      .catch(() => setLastSeen({}));
  }

  useEffect(() => {
    setDepartment("");
    if (selectedId) load(selectedId);
    else setEmployees([]);
  }, [selectedId]);

  async function sync() {
    if (!selectedId) return;
    setSyncing(true);
    setSyncMsg(null);
    try {
      const res = await syncDirectory(selectedId);
      setSyncMsg({ kind: "success", text: t("employees.synced", { count: res.members }) });
      load(selectedId);
    } catch (err) {
      setSyncMsg({ kind: "danger", text: err instanceof ApiError ? err.message : t("employees.syncFailed") });
    } finally {
      setSyncing(false);
    }
  }

  const hasBusiness = businesses.length > 0;
  const visible = department ? employees.filter((e) => e.department_id === department) : employees;

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <div className="admin-pagehead">
        <div className="admin-pagehead__main">
          <h1 className="admin-h1">{terms.many}</h1>
          {selected && (
            <p className="admin-sub">
              {selected.name} · {visible.length} {terms.many}
            </p>
          )}
        </div>
        {hasBusiness && (
          <div className="admin-pagehead__actions">
            {departments.length > 0 && (
              <span className="bibo-input" style={{ minWidth: 200 }}>
                <select
                  aria-label={t("employees.table.department")}
                  value={department}
                  onChange={(e) => setDepartment(e.target.value)}
                >
                  <option value="">{t("employees.allDepartments")}</option>
                  {departments.map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.name || d.id} ({d.members})
                    </option>
                  ))}
                </select>
              </span>
            )}
            <button className="bibo-btn bibo-btn--secondary" onClick={sync} disabled={syncing || !selectedId}>
              <span style={{ display: "inline-flex", lineHeight: 0 }}>{IconRefresh}</span>
              <span>{syncing ? t("employees.syncing") : t("employees.syncDirectory")}</span>
            </button>
          </div>
        )}
      </div>

      {!hasBusiness && !bizLoading && (
        <div style={{ marginBottom: 16 }}>
          <Notice kind="info">{t("employees.noBusinessNotice")}</Notice>
        </div>
      )}

      {hasBusiness && (
        <p className="admin-muted" style={{ margin: "0 0 16px" }}>
          {t("employees.directoryNote")}
        </p>
      )}

      {syncMsg && (
        <div style={{ marginBottom: 16 }}>
          <Notice kind={syncMsg.kind}>{syncMsg.text}</Notice>
        </div>
      )}

      {bizLoading && <Spinner label={t("employees.loading")} />}

      {listError && <Notice kind="danger">{listError}</Notice>}
      {loading && <Spinner label={t("employees.loadingMembers", { members: terms.lowerMany })} />}

      {!loading && selectedId && visible.length === 0 && !listError && (
        <Empty>{t("employees.noMembersYet", { members: terms.lowerMany })}</Empty>
      )}

      {visible.length > 0 && (
        <div className="bibo-card bibo-card--default admin-tablecard">
          <table className="admin-table admin-table--roster">
            <thead>
              <tr>
                <th>{t("employees.table.name")}</th>
                <th>{t("employees.table.department")}</th>
                <th>{t("employees.table.login")}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {visible.map((e, i) => {
                const pal = AVATAR_PALETTE[i % AVATAR_PALETTE.length];
                const isSelf = e.id === user?.id;
                const status = e.id in lastSeen ? memberStatus(lastSeen[e.id]) : null;
                return (
                  <tr key={e.id}>
                    <td>
                      <div className="admin-name">
                        <span className="bibo-avatar" style={{ ["--_s" as string]: "34px" }}>
                          {e.avatar_url ? (
                            <img className="bibo-avatar__img" src={e.avatar_url} alt={e.display_name} />
                          ) : (
                            <span className="bibo-avatar__img" aria-label={e.display_name} style={{ background: pal.bg, color: pal.fg }}>
                              {initials(e.display_name)}
                            </span>
                          )}
                          {status && <span className={`bibo-avatar__dot bibo-avatar__dot--${status}`} />}
                        </span>
                        <span className="admin-name__txt">
                          {e.display_name}
                          {isSelf && <span className="admin-self">{t("dashboard.selfBadge")}</span>}
                          {e.job_title && (
                            <span className="admin-muted" style={{ display: "block", fontSize: 12 }}>
                              {e.job_title}
                            </span>
                          )}
                        </span>
                      </div>
                    </td>
                    <td>{e.department_name || "—"}</td>
                    <td className="admin-login">{e.email || e.username}</td>
                    <td className="r">
                      <Link className="admin-viewlink" to={`/employees/${e.id}?business=${selectedId}`}>
                        {t("employees.viewReports")}
                        {IconArrowRight}
                      </Link>
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
