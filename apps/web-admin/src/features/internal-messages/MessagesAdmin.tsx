import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { adminListMessages } from "@/features/internal-messages/api";
import type { AdminMessageRow, EventCounts } from "@/shared/api/types";
import { Empty, Notice, Spinner } from "@/shared/ui/index";
import { messageStatus, pct, setActive, uniq } from "./messageUtils";

// Super-admin list of in-app messages with their headline numbers (unique users).
export function MessagesAdmin() {
  const { t } = useTranslation("dashboard");
  const navigate = useNavigate();
  const [rows, setRows] = useState<AdminMessageRow[] | null>(null);
  const [whatsNew, setWhatsNew] = useState<{ id: string; stats: EventCounts }[]>([]);
  const [error, setError] = useState<string | null>(null);

  const load = () =>
    adminListMessages()
      .then((r) => {
        setRows(r.messages);
        setWhatsNew([...r.whatsnew].sort((a, b) => b.id.localeCompare(a.id, undefined, { numeric: true })));
      })
      .catch((e) => setError(String(e.message ?? e)));
  useEffect(() => {
    load();
  }, []);

  async function toggle(m: AdminMessageRow) {
    try {
      await setActive(m, !m.active);
      await load();
    } catch (e) {
      setError(String((e as Error).message ?? e));
    }
  }

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <div className="admin-pagehead">
        <div className="admin-pagehead__main">
          <h1 className="admin-h1">{t("internal.title")}</h1>
          <p className="admin-sub">{t("internal.sub")}</p>
        </div>
        <button type="button" className="bibo-btn bibo-btn--primary" onClick={() => navigate("/internal/messages/new")}>
          {t("internal.new")}
        </button>
      </div>

      {error && <Notice kind="danger">{error}</Notice>}
      {!rows && !error && <Spinner />}
      {rows && rows.length === 0 && <Empty>{t("internal.empty")}</Empty>}

      {rows && rows.length > 0 && (
        <div className="bibo-card bibo-card--default admin-tablecard">
          <table className="admin-table">
            <thead>
              <tr>
                <th>{t("internal.col.message")}</th>
                <th>{t("internal.col.type")}</th>
                <th>{t("internal.col.status")}</th>
                <th className="r">{t("internal.stat.delivered")}</th>
                <th className="r">{t("internal.stat.shown")}</th>
                <th className="r">{t("internal.stat.cta")}</th>
                <th className="r">{t("internal.stat.submitted")}</th>
                <th className="r">{t("internal.stat.dismissed")}</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((m) => {
                const shown = uniq(m.stats, "shown");
                const status = messageStatus(m);
                return (
                  <tr key={m.id} className="im-row" onClick={() => navigate(`/internal/messages/${m.id}`)}>
                    <td>
                      <div className="im-title">{m.content.en?.title ?? m.id}</div>
                      <div className="im-id">{m.id}</div>
                    </td>
                    <td>{t(`internal.kind.${m.kind}`)}</td>
                    <td>
                      <span className={`im-status im-status--${status}`}>{t(`internal.status.${status}`)}</span>
                    </td>
                    <td className="r num">{uniq(m.stats, "delivered")}</td>
                    <td className="r num">{shown}</td>
                    <td className="r num">
                      {m.kind === "survey" ? "—" : <>{uniq(m.stats, "cta")} <span className="im-pct">{pct(uniq(m.stats, "cta"), shown)}</span></>}
                    </td>
                    <td className="r num">
                      {m.kind === "survey" ? <>{uniq(m.stats, "submitted")} <span className="im-pct">{pct(uniq(m.stats, "submitted"), shown)}</span></> : "—"}
                    </td>
                    <td className="r num">{uniq(m.stats, "dismissed")}</td>
                    <td className="r" onClick={(e) => e.stopPropagation()}>
                      <button type="button" className={`im-toggle${m.active ? "" : " im-toggle--on"}`} onClick={() => toggle(m)}>
                        {m.active ? t("internal.pause") : t("internal.activate")}
                      </button>
                      <Link className="admin-viewlink" to={`/internal/messages/${m.id}/edit`}>{t("internal.edit")}</Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {whatsNew.length > 0 && (
        <>
          <h2 className="im-h2">{t("internal.whatsNew")}</h2>
          <div className="bibo-card bibo-card--default admin-tablecard">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>{t("internal.col.version")}</th>
                  <th className="r">{t("internal.stat.shown")}</th>
                  <th className="r">{t("internal.stat.dismissed")}</th>
                </tr>
              </thead>
              <tbody>
                {whatsNew.map((w) => (
                  <tr key={w.id} className="im-row" onClick={() => navigate(`/internal/messages/${w.id}`)}>
                    <td>{w.id.replace(/^whatsnew-/, "")}</td>
                    <td className="r num">{uniq(w.stats, "shown")}</td>
                    <td className="r num">{uniq(w.stats, "dismissed")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
