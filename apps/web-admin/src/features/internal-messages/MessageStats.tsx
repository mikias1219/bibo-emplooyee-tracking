import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { adminMessageStats } from "@/features/internal-messages/api";
import type {
  MessageDef,
  MessageEvent,
  MessageFieldSummary,
  MessageResponseRow,
  MessageStats as Stats,
} from "@/shared/api/types";
import { Empty, Notice, Spinner } from "@/shared/ui/index";
import { messageStatus, pct, responsesCsv, setActive, uniq } from "./messageUtils";

const DIMS = ["locale", "platform", "version", "role"] as const;

// Per-message analytics: headline numbers, last 60 days, breakdowns, and for
// surveys a per-question summary with CSV export. Counts are unique users.
export function MessageStats() {
  const { t } = useTranslation("dashboard");
  const { id = "" } = useParams();
  const [s, setS] = useState<Stats | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = () => adminMessageStats(id).then(setS).catch((e) => setError(String(e.message ?? e)));
  useEffect(() => {
    load();
  }, [id]);

  async function toggleActive(active: boolean) {
    try {
      await setActive(s!.message!, active);
      await load();
    } catch (e) {
      setError(String((e as Error).message ?? e));
    }
  }

  const m = s?.message;
  const isWhatsNew = id.startsWith("whatsnew-");
  const isSurvey = m?.kind === "survey";
  // Events worth showing for this kind of message.
  const events: MessageEvent[] = isWhatsNew
    ? ["shown", "dismissed"]
    : isSurvey
      ? ["delivered", "shown", "submitted", "later", "dismissed"]
      : ["delivered", "shown", "cta", "later", "dismissed"];

  const daily = useMemo(() => {
    const byDay = new Map<string, Partial<Record<MessageEvent, number>>>();
    for (const d of s?.daily ?? []) byDay.set(d.day, { ...byDay.get(d.day), [d.event]: d.unique });
    return [...byDay.entries()].sort(([a], [b]) => b.localeCompare(a));
  }, [s]);
  const maxShown = Math.max(1, ...daily.map(([, v]) => v.shown ?? 0));

  if (error) return <div className="admin-wrap"><Notice kind="danger">{error}</Notice></div>;
  if (!s) return <div className="admin-wrap"><Spinner /></div>;

  const shown = uniq(s.totals, "shown");
  const title = isWhatsNew ? `${t("internal.whatsNewShort")} ${id.replace(/^whatsnew-/, "")}` : m?.content.en?.title ?? id;

  function exportCsv() {
    const csv = responsesCsv((m?.fields ?? []).map((f) => f.id), s!.responses ?? []);
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    a.download = `${id}-responses.csv`;
    a.click();
    URL.revokeObjectURL(a.href);
  }

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <Link className="im-back" to="/internal/messages">← {t("internal.back")}</Link>
      <div className="admin-pagehead">
        <div className="admin-pagehead__main">
          <h1 className="admin-h1">{title}</h1>
          <p className="admin-sub">
            {id}
            {m && <> · {t(`internal.kind.${m.kind}`)} · <span className={`im-status im-status--${messageStatus(m)}`}>{t(`internal.status.${messageStatus(m)}`)}</span></>}
          </p>
        </div>
        {m && (
          <div className="im-actions">
            {m.active && (
              <button type="button" className="bibo-btn bibo-btn--ghost" onClick={() => toggleActive(false)}>
                {t("internal.pause")}
              </button>
            )}
            <Link className="bibo-btn bibo-btn--secondary" to={`/internal/messages/${id}/edit`}>{t("internal.edit")}</Link>
          </div>
        )}
      </div>

      {m && !m.active && (
        <div className="im-paused">
          <span>{t("internal.pausedBanner")}</span>
          <button type="button" className="bibo-btn bibo-btn--primary bibo-btn--sm" onClick={() => toggleActive(true)}>
            {t("internal.activate")}
          </button>
        </div>
      )}

      <div className="im-kpis">
        {events.map((e) => {
          const c = s.totals[e];
          const rate = e === "cta" ? pct(uniq(s.totals, "cta"), shown) : e === "submitted" ? pct(uniq(s.totals, "submitted"), shown) : null;
          return (
            <div key={e} className="bibo-card bibo-card--default admin-cardpad im-kpi">
              <div className="im-kpi__lbl">{t(`internal.stat.${e}`)}</div>
              <div className="im-kpi__val num">{c?.unique ?? 0}</div>
              <div className="im-kpi__sub">
                {rate && <strong>{rate} {t(e === "cta" ? "internal.stat.ctr" : "internal.stat.completion")} · </strong>}
                {t("internal.stat.totalEvents", { count: c?.total ?? 0 })}
              </div>
            </div>
          );
        })}
      </div>

      <div className="bibo-card bibo-card--default admin-cardpad im-section">
        <div className="admin-paneltitle">{t("internal.stat.daily")}</div>
        {daily.length === 0 ? (
          <Empty>{t("internal.stat.noData")}</Empty>
        ) : (
          <table className="admin-table im-table">
            <thead>
              <tr>
                <th>{t("internal.stat.day")}</th>
                {events.map((e) => <th key={e} className="r">{t(`internal.stat.${e}`)}</th>)}
                <th style={{ width: "30%" }}></th>
              </tr>
            </thead>
            <tbody>
              {daily.map(([day, v]) => (
                <tr key={day}>
                  <td>{day}</td>
                  {events.map((e) => <td key={e} className="r num">{v[e] ?? 0}</td>)}
                  <td><div className="im-bar"><span style={{ width: `${((v.shown ?? 0) / maxShown) * 100}%` }} /></div></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {s.breakdown.length > 0 && (
        <div className="im-breakdowns">
          {DIMS.map((dim) => {
            const rows = new Map<string, Partial<Record<MessageEvent, number>>>();
            for (const b of s.breakdown.filter((x) => x.dim === dim)) {
              rows.set(b.value, { ...rows.get(b.value), [b.event]: b.unique });
            }
            const conv: MessageEvent = isSurvey ? "submitted" : "cta";
            return (
              <div key={dim} className="bibo-card bibo-card--default admin-cardpad">
                <div className="admin-paneltitle">{t(`internal.stat.dim.${dim}`)}</div>
                <table className="admin-table im-table">
                  <thead>
                    <tr>
                      <th></th>
                      <th className="r">{t("internal.stat.shown")}</th>
                      {!isWhatsNew && <th className="r">{t(`internal.stat.${conv}`)}</th>}
                    </tr>
                  </thead>
                  <tbody>
                    {[...rows.entries()].sort(([, a], [, b]) => (b.shown ?? 0) - (a.shown ?? 0)).map(([value, v]) => (
                      <tr key={value}>
                        <td>{value || "—"}</td>
                        <td className="r num">{v.shown ?? 0}</td>
                        {!isWhatsNew && <td className="r num">{v[conv] ?? 0} <span className="im-pct">{pct(v[conv] ?? 0, v.shown ?? 0)}</span></td>}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            );
          })}
        </div>
      )}

      {isSurvey && (
        <div className="bibo-card bibo-card--default admin-cardpad im-section">
          <div className="im-survey-head">
            <div className="admin-paneltitle">
              {t("internal.stat.survey")} · {t("internal.stat.answers", { count: s.responses?.length ?? 0 })}
            </div>
            {(s.responses?.length ?? 0) > 0 && (
              <button type="button" className="bibo-btn bibo-btn--secondary bibo-btn--sm" onClick={exportCsv}>
                {t("internal.stat.exportCsv")}
              </button>
            )}
          </div>
          {(s.responses?.length ?? 0) === 0 ? (
            <Empty>{t("internal.stat.noAnswers")}</Empty>
          ) : (
            (s.summary ?? []).map((q) => <QuestionSummary key={q.id} q={q} />)
          )}
        </div>
      )}

      {isSurvey && m && (s.responses?.length ?? 0) > 0 && <Responses message={m} rows={s.responses!} />}
    </div>
  );
}

function QuestionSummary({ q }: { q: MessageFieldSummary }) {
  const { t } = useTranslation("dashboard");
  // Full axis (0–10 for NPS, 1–5 for stars) so gaps are visible.
  const [lo, hi] = q.type === "nps" ? [0, 10] : [1, 5];
  const scale = q.scale ? Array.from({ length: hi - lo + 1 }, (_, i) => [lo + i, q.scale![lo + i] ?? 0] as const) : [];
  const maxScale = Math.max(1, ...scale.map(([, v]) => v));
  return (
    <div className="im-qsum">
      <div className="im-qsum__label">
        {q.label} <span className="admin-muted">· {t("internal.stat.answers", { count: q.answered })}</span>
      </div>
      {q.options?.map((o) => (
        <div key={o.value} className="im-optbar">
          <span className="im-optbar__lbl">{o.label}</span>
          <div className="im-bar"><span style={{ width: pct(o.count, q.answered) === "—" ? 0 : pct(o.count, q.answered) }} /></div>
          <span className="im-optbar__n num">{o.count} <span className="im-pct">{pct(o.count, q.answered)}</span></span>
        </div>
      ))}
      {q.average != null && (
        <div className="im-scale">
          <div className="im-scale__head">
            {t("internal.stat.average")}: <strong className="num">{q.average.toFixed(1)}</strong>
            {q.nps != null && <> · {t("internal.stat.nps")}: <strong className="num">{q.nps > 0 ? `+${q.nps}` : q.nps}</strong></>}
          </div>
          <div className="im-scale__bars">
            {scale.map(([k, v]) => (
              <div key={k} className="im-scale__col" title={`${k}: ${v}`}>
                <div className="im-scale__bar" style={{ height: `${(v / maxScale) * 100}%` }} />
                <span>{k}</span>
              </div>
            ))}
          </div>
        </div>
      )}
      {q.texts && q.texts.length > 0 && (
        <ul className="im-texts">
          {q.texts.map((x, i) => <li key={i}>{x}</li>)}
        </ul>
      )}
    </div>
  );
}

const PAGE = 25;

/** Every submission, newest first: who (or Anonymous), when, app, and each answer
 *  shown with its option labels (English) rather than raw values. */
function Responses({ message, rows }: { message: MessageDef; rows: MessageResponseRow[] }) {
  const { t, i18n } = useTranslation("dashboard");
  const [limit, setLimit] = useState(PAGE);
  const fields = message.fields ?? [];
  const when = new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium", timeStyle: "short" });
  // Anonymous answers are stored to the day only (so they can't be matched to a
  // user by time) — show just the date for those.
  const day = new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium" });

  const show = (fieldId: string, v: MessageResponseRow["answers"][string] | undefined) => {
    if (v === undefined || v === "") return "—";
    const f = fields.find((x) => x.id === fieldId);
    const label = (val: string) => {
      const o = f?.options?.find((x) => x.value === val);
      return o ? o.label.en || val : val;
    };
    if (Array.isArray(v)) return v.map(label).join(", ");
    if (f?.type === "rating") return `${"★".repeat(Number(v))} (${v}/5)`;
    if (f?.type === "nps") return `${v}/10`;
    return typeof v === "string" ? label(v) : String(v);
  };

  return (
    <div className="bibo-card bibo-card--default admin-cardpad im-section">
      <div className="admin-paneltitle">
        {t("internal.resp.title")} · {rows.length}
      </div>
      <div className="im-resp-scroll">
        <table className="admin-table im-table im-resp">
          <thead>
            <tr>
              <th>{t("internal.resp.when")}</th>
              <th>{t("internal.resp.who")}</th>
              {fields.map((f) => (
                <th key={f.id} className="im-resp__q" title={f.label.en}>{f.label.en || f.id}</th>
              ))}
              <th>{t("internal.resp.app")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.slice(0, limit).map((r, i) => (
              <tr key={i}>
                <td className="im-resp__when">{(r.user_id ? when : day).format(new Date(r.created_at))}</td>
                <td>
                  {r.user_name || r.user_login ? (
                    <>
                      <div className="im-title">{r.user_name}</div>
                      <div className="im-id">{r.user_login}</div>
                    </>
                  ) : (
                    <span className="admin-muted">{t("internal.resp.anonymous")}</span>
                  )}
                </td>
                {fields.map((f) => (
                  <td key={f.id} className="im-resp__ans">{show(f.id, r.answers[f.id])}</td>
                ))}
                <td className="im-resp__app admin-muted">
                  {[r.app_version, r.platform, r.locale].filter(Boolean).join(" · ") || "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {rows.length > limit && (
        <button type="button" className="bibo-btn bibo-btn--secondary bibo-btn--sm im-more" onClick={() => setLimit(limit + PAGE)}>
          {t("internal.resp.more", { count: Math.min(PAGE, rows.length - limit) })}
        </button>
      )}
    </div>
  );
}
