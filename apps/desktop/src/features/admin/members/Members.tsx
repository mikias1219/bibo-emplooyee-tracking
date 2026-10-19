import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { call as invoke } from "@/shared/api/tauri";
import { AVATAR_PALETTE, initials, isSelf as isSelfEmail, localDayStart, memberStatus } from "../lib";
import type { RosterEntry } from "../types";
import { EmployeeDetail } from "../employee-detail/EmployeeDetail";

const svgp = { viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 2, strokeLinecap: "round", strokeLinejoin: "round" } as const;
const IconArrow = () => (<svg {...svgp} width="15" height="15" aria-hidden><path d="M5 12h14" /><path d="m12 5 7 7-7 7" /></svg>);
const IconRefresh = () => (<svg {...svgp} width="16" height="16" aria-hidden><path d="M3 12a9 9 0 0 1 15-6.7L21 8" /><path d="M21 3v5h-5" /><path d="M21 12a9 9 0 0 1-15 6.7L3 16" /><path d="M8 16H3v5" /></svg>);

// The Members admin screen: the workspace roster (drill into a member's native
// detail). People, departments and roles come from the Selamnew Org&Emp
// directory, so there is no add/remove here — owners can filter by department and
// pull a fresh copy of the directory.
export function Members({
  businessId,
  businessName,
  selfEmail,
  detail,
  onView,
}: {
  businessId: string;
  businessName: string;
  selfEmail: string;
  detail: RosterEntry | null;
  onView: (e: RosterEntry | null) => void;
}) {
  const { t } = useTranslation();
  const [rows, setRows] = useState<RosterEntry[] | null>(null);
  const [listError, setListError] = useState<string | null>(null);
  const [department, setDepartment] = useState("");
  const [syncing, setSyncing] = useState(false);
  const [syncMsg, setSyncMsg] = useState<{ ok: boolean; text: string } | null>(null);

  const load = useCallback(() => {
    setRows(null);
    setListError(null);
    invoke<RosterEntry[]>("admin_roster", { businessId, dayStart: localDayStart() })
      .then(setRows)
      .catch((e) => setListError(String(e)));
  }, [businessId]);

  useEffect(() => {
    setDepartment("");
    load();
  }, [load]);

  async function sync() {
    if (syncing) return;
    setSyncing(true);
    setSyncMsg(null);
    try {
      const count = await invoke<number>("admin_sync_directory", { businessId });
      setSyncMsg({ ok: true, text: t("screens:admin.synced", { count }) });
      load();
    } catch (err) {
      setSyncMsg({ ok: false, text: String(err) });
    } finally {
      setSyncing(false);
    }
  }

  if (detail) {
    return <EmployeeDetail employee={detail} selfEmail={selfEmail} onBack={() => onView(null)} />;
  }

  const all = rows ?? [];
  // Departments present in the roster (id → name), for the filter.
  const departments = Array.from(
    new Map(all.filter((e) => e.department_id).map((e) => [e.department_id!, e.department_name || e.department_id!])),
  ).sort((a, b) => a[1].localeCompare(b[1]));
  const list = department ? all.filter((e) => e.department_id === department) : all;

  return (
    <div className="bb-members">
      <div className="bb-members__head">
        <div>
          <h1 className="bb-adminboard__title">{t("nav.Members")}</h1>
          <p className="bb-adminboard__who">
            {businessName} · {list.length} {t("screens:admin.employees").toLowerCase()}
          </p>
        </div>
        <div className="bb-members__actions">
          {departments.length > 0 && (
            <span className="bibo-input">
              <select
                aria-label={t("screens:admin.department")}
                value={department}
                onChange={(e) => setDepartment(e.target.value)}
              >
                <option value="">{t("screens:admin.allDepartments")}</option>
                {departments.map(([id, name]) => (
                  <option key={id} value={id}>{name}</option>
                ))}
              </select>
            </span>
          )}
          <button type="button" className="bibo-btn bibo-btn--secondary bb-members__add" onClick={sync} disabled={syncing}>
            <IconRefresh /> {syncing ? t("screens:admin.syncing") : t("screens:admin.syncDirectory")}
          </button>
        </div>
      </div>

      <p className="muted" style={{ margin: "0 0 12px" }}>{t("screens:admin.directoryNote")}</p>

      {syncMsg && (
        <div className={syncMsg.ok ? "muted" : "auth-err"} role={syncMsg.ok ? "status" : "alert"} style={{ marginBottom: 12 }}>
          {syncMsg.text}
        </div>
      )}
      {listError && <div className="auth-err" role="alert">{listError}</div>}

      <div className="card bb-adminboard__tablecard">
        {rows === null ? (
          <div className="muted bb-adminboard__msg">{t("loading")}</div>
        ) : list.length === 0 ? (
          <div className="muted bb-adminboard__msg">{t("screens:admin.empty")}</div>
        ) : (
          <table className="bb-adminboard__table">
            <thead>
              <tr>
                <th>{t("screens:admin.employees")}</th>
                <th>{t("screens:admin.department")}</th>
                <th>{t("screens:admin.login")}</th>
                <th>{t("screens:admin.role")}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {list.map((e, i) => {
                const pal = AVATAR_PALETTE[i % AVATAR_PALETTE.length];
                const status = memberStatus(e.last_seen);
                const isSelf = isSelfEmail(e, selfEmail);
                return (
                  <tr key={e.id}>
                    <td>
                      <div className="bb-adminboard__name">
                        <span className="bb-adminboard__avatar" style={{ background: pal.bg, color: pal.fg }}>
                          {initials(e.display_name)}
                          <span className={`bb-adminboard__dot bb-adminboard__dot--${status}`} />
                        </span>
                        <span className="bb-adminboard__nameid">
                          <span className="bb-adminboard__nametxt" title={e.job_title || e.display_name}>{e.display_name}</span>
                          {isSelf && (
                            <span className="bibo-badge bb-adminboard__you" style={{ background: pal.bg, color: pal.fg }}>
                              {t("screens:admin.you")}
                            </span>
                          )}
                        </span>
                      </div>
                    </td>
                    <td>{e.department_name || "—"}</td>
                    <td className="bb-adminboard__login">{e.email || e.username}</td>
                    <td>
                      <span className="bibo-badge" style={{ background: pal.bg, color: pal.fg }}>
                        {e.role === "owner" ? t("screens:admin.roleOwner") : t("screens:admin.roleEmployee")}
                      </span>
                    </td>
                    <td className="r">
                      <button type="button" className="bb-adminboard__view" onClick={() => onView(e)}>
                        {t("screens:admin.view")} <IconArrow />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
