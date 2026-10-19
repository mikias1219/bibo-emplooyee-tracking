import { useEffect, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { PerfEmployee, PerfFigures, PerformanceResponse } from "../../types";
import { Empty, Notice, Spinner } from "../../ui";
import { fmtDuration } from "../../format";
import { PeriodPicker, usePeriod } from "../../components/PeriodPicker";
import { useAdmin } from "../../context";
import { useMemberTerms, useWorkspace } from "../../hooks";

// Category colours, shared with the Rules page.
export const CATEGORY_COLOR = {
  productive: "var(--positive)",
  neutral: "var(--data-sky)",
  unproductive: "var(--negative)",
  unclassified: "var(--text-muted)",
} as const;

const SEGMENTS = ["productive", "neutral", "unproductive", "unclassified"] as const;

const svg = (children: ReactNode) => (
  <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
    {children}
  </svg>
);
const IconTarget = svg(<><circle cx="12" cy="12" r="10" /><circle cx="12" cy="12" r="6" /><circle cx="12" cy="12" r="2" /></>);
const IconTrend = svg(<><path d="M22 7 13.5 15.5 8.5 10.5 2 17" /><path d="M16 7h6v6" /></>);
const IconClock = svg(<><circle cx="12" cy="12" r="10" /><path d="M12 6v6l4 2" /></>);
const IconTag = svg(<><path d="M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z" /><circle cx="7.5" cy="7.5" r=".5" fill="currentColor" /></>);
const IconArrowRight = svg(<><path d="M5 12h14" /><path d="m12 5 7 7-7 7" /></>);

const pct = (v: number | null) => (v == null ? "—" : `${Math.round(v)}%`);

function scoreColor(v: number | null): string {
  if (v == null) return "var(--text-muted)";
  if (v >= 75) return "var(--positive)";
  if (v >= 50) return "var(--data-amber)";
  return "var(--negative)";
}

/** Tabs shared by the Performance report and the Rules page. */
export function PerformanceTabs() {
  const { t } = useTranslation("admin-performance");
  const { view, navigate } = useAdmin();
  const tab = (v: "performance" | "rules", label: string) => (
    <button
      type="button"
      role="tab"
      aria-selected={view === v}
      className={`bibo-tab${view === v ? " bibo-tab--on" : ""}`}
      onClick={() => navigate(v)}
    >
      {label}
    </button>
  );
  return (
    <div className="bibo-tabs bibo-tabs--pill" role="tablist" style={{ marginBottom: 16 }}>
      {tab("performance", t("tabs.report"))}
      {tab("rules", t("tabs.rules"))}
    </div>
  );
}


/** Horizontal stacked bar of the four categories. */
function CategoryBar({ f, height = 10 }: { f: PerfFigures; height?: number }) {
  const { t } = useTranslation("admin-performance");
  const total = f.active_s;
  if (total <= 0) return <div className="admin-perfbar admin-perfbar--empty" style={{ height }} />;
  const parts = SEGMENTS.map((k) => ({ k, s: f[`${k}_s` as const] }));
  return (
    <div
      className="admin-perfbar"
      style={{ height }}
      role="img"
      aria-label={parts.map((p) => `${t(`category.${p.k}`)} ${fmtDuration(p.s)}`).join(", ")}
    >
      {parts.map((p) =>
        p.s > 0 ? (
          <span
            key={p.k}
            title={`${t(`category.${p.k}`)}: ${fmtDuration(p.s)}`}
            style={{ width: `${(p.s / total) * 100}%`, background: CATEGORY_COLOR[p.k] }}
          />
        ) : null,
      )}
    </div>
  );
}

function Legend() {
  const { t } = useTranslation("admin-performance");
  return (
    <div className="admin-perflegend">
      {SEGMENTS.map((k) => (
        <span key={k}>
          <span className="dot" style={{ background: CATEGORY_COLOR[k] }} />
          {t(`category.${k}`)}
        </span>
      ))}
    </div>
  );
}

function StatCard(props: { icon: ReactNode; label: string; value: ReactNode; focal?: boolean; sub?: string }) {
  const { icon, label, value, focal, sub } = props;
  return (
    <div className={`bibo-card ${focal ? "bibo-card--focal" : "bibo-card--default"} admin-cardpad`}>
      <div className={`bibo-stat${focal ? " bibo-stat--focal" : ""}`}>
        <div className="bibo-stat__top">
          <div className="bibo-stat__icon">{icon}</div>
          <div className="bibo-stat__label">{label}</div>
        </div>
        <div className="bibo-stat__value">{value}</div>
        <div className="bibo-stat__foot">{sub && <span className="bibo-stat__sub">{sub}</span>}</div>
      </div>
    </div>
  );
}

/** Top / least productive lists (EmpMonitor-style), ranked by productive time. */
function RankList({ title, rows, onOpen }: { title: string; rows: PerfEmployee[]; onOpen: (id: string) => void }) {
  const { t } = useTranslation("admin-performance");
  return (
    <div className="bibo-card bibo-card--default admin-cardpad">
      <div className="admin-panelhead">
        <span className="admin-paneltitle">{title}</span>
      </div>
      {rows.length === 0 ? (
        <div className="admin-muted">{t("report.noRank")}</div>
      ) : (
        <div className="admin-perfrank">
          {rows.map((e) => (
            <button type="button" key={e.id} className="admin-perfrank__row" onClick={() => onOpen(e.id)}>
              <span className="admin-perfrank__name">{e.display_name}</span>
              <span className="admin-perfrank__val">{fmtDuration(e.productive_s)}</span>
              <span className="admin-perfrank__pct" style={{ color: scoreColor(e.score_pct) }}>
                {pct(e.score_pct)}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

export function Performance() {
  const { t } = useTranslation("admin-performance");
  const { api, navigate, openEmployee } = useAdmin();
  const { selectedId, loading: bizLoading } = useWorkspace();
  const terms = useMemberTerms();
  const period = usePeriod();
  const { from, to } = period.window;

  const [data, setData] = useState<PerformanceResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!selectedId) return;
    if (from > to) {
      setError(t("report.errorRange"));
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    api.reportPerformance(selectedId, from, to)
      .then((r) => !cancelled && setData(r))
      .catch(() => !cancelled && setError(t("report.error")))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [api, selectedId, from, to, t]);

  const open = openEmployee;
  // Rank members with classified time by score; the two lists never overlap
  // (top half vs bottom half, at most 5 each).
  const ranked = (data?.employees ?? [])
    .filter((e) => e.score_pct != null)
    .sort((a, b) => (b.score_pct ?? 0) - (a.score_pct ?? 0));
  const half = Math.ceil(ranked.length / 2);
  const top = ranked.slice(0, Math.min(5, half));
  const least = ranked.slice(half).reverse().slice(0, 5);
  const team = data?.team;
  const maxDay = Math.max(1, ...(data?.days.map((d) => d.active_s) ?? [0]));

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <div className="admin-pagehead">
        <div className="admin-pagehead__main">
          <h1 className="admin-h1">{t("title")}</h1>
          <p className="admin-sub">{t("subtitle")}</p>
        </div>
        <div className="admin-pagehead__actions">
          <PeriodPicker period={period} />
        </div>
      </div>

      <PerformanceTabs />

      {bizLoading && <Spinner />}
      {error && <Notice kind="danger">{error}</Notice>}
      {loading && !data && <Spinner label={t("report.loading")} />}

      {data && team && (
        <>
          {team.coverage_pct != null && team.coverage_pct < 80 && (
            <Notice kind="info">
              {t("report.lowCoverage", { pct: Math.round(team.coverage_pct) })}{" "}
              <button type="button" className="admin-link" onClick={() => navigate("rules")}>{t("report.classifyNow")}</button>
            </Notice>
          )}

          <div className="admin-stats">
            <StatCard
              focal
              icon={IconTrend}
              label={t("report.score")}
              value={team.score_pct == null ? "—" : <>{Math.round(team.score_pct)}<span className="bibo-stat__unit">%</span></>}
              sub={t("report.scoreSub")}
            />
            <StatCard
              icon={IconClock}
              label={t("report.productiveTime")}
              value={fmtDuration(team.productive_s)}
              sub={t("report.ofActive", { active: fmtDuration(team.active_s) })}
            />
            <StatCard
              icon={IconTarget}
              label={t("report.target")}
              value={pct(team.target_pct)}
              sub={t("report.targetSub", { target: fmtDuration(data.target_per_day_s) })}
            />
            <StatCard
              icon={IconTag}
              label={t("report.coverage")}
              value={pct(team.coverage_pct)}
              sub={t("report.coverageSub")}
            />
          </div>

          {data.days.length > 0 && (
            <div className="bibo-card bibo-card--default admin-cardpad" style={{ marginBottom: 14 }}>
              <div className="admin-panelhead">
                <span className="admin-paneltitle">{t("report.byDay")}</span>
                <span style={{ marginLeft: "auto" }}>
                  <Legend />
                </span>
              </div>
              <div className="admin-perfdays">
                {data.days.map((d) => (
                  <div className="admin-perfday" key={d.day}>
                    <span className="admin-perfday__lbl">{d.day.slice(5)}</span>
                    <div style={{ width: `${(d.active_s / maxDay) * 100}%`, minWidth: 4 }}>
                      <CategoryBar f={d} />
                    </div>
                    <span className="admin-perfday__val" style={{ color: scoreColor(d.score_pct) }}>
                      {pct(d.score_pct)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="admin-twocol" style={{ gridTemplateColumns: "1fr 1fr", marginBottom: 14 }}>
            <RankList title={t("report.top")} rows={top} onOpen={open} />
            <RankList title={t("report.least")} rows={least} onOpen={open} />
          </div>

          {data.employees.length === 0 ? (
            <Empty>{t("report.empty", { members: terms.lowerMany })}</Empty>
          ) : (
            <div className="bibo-card bibo-card--default admin-tablecard">
              <table className="admin-table">
                <thead>
                  <tr>
                    <th>{t("table.name")}</th>
                    <th style={{ width: "34%" }}>{t("table.breakdown")}</th>
                    <th className="r">{t("table.active")}</th>
                    <th className="r">{t("table.productive")}</th>
                    <th className="r">{t("table.score")}</th>
                    <th className="r">{t("table.target")}</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {data.employees.map((e) => (
                    <tr key={e.id}>
                      <td>
                        <span className="admin-name__txt">{e.display_name}</span>
                        <div className="admin-login">{e.email || e.username}</div>
                      </td>
                      <td>
                        <CategoryBar f={e} />
                      </td>
                      <td className="r admin-dur">{fmtDuration(e.active_s)}</td>
                      <td className="r admin-dur">{fmtDuration(e.productive_s)}</td>
                      <td className="r" style={{ color: scoreColor(e.score_pct), fontWeight: 700 }}>
                        {pct(e.score_pct)}
                      </td>
                      <td className="r">{pct(e.target_pct)}</td>
                      <td className="r">
                        <button type="button" className="admin-viewlink" onClick={() => open(e.id)}>
                          {t("table.view")}
                          {IconArrowRight}
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <p className="admin-muted" style={{ marginTop: 12, fontSize: 12.5 }}>
            {t("report.disclaimer")}
          </p>
        </>
      )}
    </div>
  );
}
