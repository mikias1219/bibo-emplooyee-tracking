import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ProductivityCategory, ProductivityRule, RuleMatchType, UsageItem } from "../../types";
import { Notice, Spinner } from "../../ui";
import { fmtDuration } from "../../format";
import { useAdmin } from "../../context";
import { useWorkspace } from "../../hooks";
import { CATEGORY_COLOR, PerformanceTabs } from "./Performance";
import { PeriodPicker, usePeriod } from "../../components/PeriodPicker";

const CATEGORIES: ProductivityCategory[] = ["productive", "neutral", "unproductive", "unclassified"];

type Row = UsageItem & { key: string };

/** Usage in the period plus rules for items not used in it (duration 0). */
function mergeRows(usage: UsageItem[], rules: ProductivityRule[], type: RuleMatchType): Row[] {
  const seen = new Set<string>();
  const rows: Row[] = usage.map((u) => {
    const key = u.name.toLowerCase();
    seen.add(key);
    return { ...u, key };
  });
  for (const r of rules) {
    if (r.match_type === type && !seen.has(r.pattern)) {
      rows.push({ name: r.pattern, duration_s: 0, category: r.category, key: r.pattern });
    }
  }
  return rows;
}

export function ProductivityRules() {
  const { t } = useTranslation("admin-performance");
  const { api } = useAdmin();
  const { selected, selectedId, reload } = useWorkspace();
  const period = usePeriod();
  const { from, to } = period.window;

  const [type, setType] = useState<RuleMatchType>("app");
  const [filter, setFilter] = useState<"all" | "unclassified">("all");
  const [usage, setUsage] = useState<{ apps: UsageItem[]; domains: UsageItem[] } | null>(null);
  const [rules, setRules] = useState<ProductivityRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState<string | null>(null);
  const [newPattern, setNewPattern] = useState("");
  const [targetH, setTargetH] = useState<string>("");
  const [targetMsg, setTargetMsg] = useState<{ kind: "success" | "danger"; text: string } | null>(null);

  useEffect(() => {
    if (selected) setTargetH(String(Math.round((selected.productive_target_s / 3600) * 10) / 10));
  }, [selected]);

  const load = useCallback(async () => {
    if (!selectedId || from > to) return;
    setLoading(true);
    setError(null);
    try {
      const [u, r] = await Promise.all([
        api.productivityUsage(selectedId, from, to),
        api.listProductivityRules(selectedId),
      ]);
      setUsage(u);
      setRules(r.rules);
    } catch {
      setError(t("rules.error"));
    } finally {
      setLoading(false);
    }
  }, [selectedId, from, to, t]);

  useEffect(() => {
    load();
  }, [load]);

  const rows = useMemo(() => {
    if (!usage) return [];
    const all = mergeRows(type === "app" ? usage.apps : usage.domains, rules, type);
    return filter === "unclassified" ? all.filter((r) => r.category === "unclassified") : all;
  }, [usage, rules, type, filter]);

  const unclassifiedCount = useMemo(() => {
    if (!usage) return 0;
    return [...usage.apps, ...usage.domains].filter((u) => u.category === "unclassified").length;
  }, [usage]);

  async function classify(name: string, category: ProductivityCategory) {
    if (!selectedId) return;
    setSaving(name);
    setError(null);
    try {
      const res = await api.setProductivityRule(selectedId, type, name, category);
      // Update locally so the list doesn't jump while the user works down it.
      setUsage((u) => {
        if (!u) return u;
        const k = type === "app" ? "apps" : "domains";
        return { ...u, [k]: u[k].map((x) => (x.name === name ? { ...x, category } : x)) };
      });
      setRules((rs) => {
        const pattern = res.rule?.pattern ?? name.toLowerCase();
        const rest = rs.filter((r) => !(r.match_type === type && r.pattern === pattern));
        return res.rule ? [...rest, res.rule] : rest;
      });
    } catch {
      setError(t("rules.saveError"));
    } finally {
      setSaving(null);
    }
  }

  async function addManual(e: React.FormEvent) {
    e.preventDefault();
    const p = newPattern.trim();
    if (!p) return;
    await classify(p, "productive");
    setNewPattern("");
    load();
  }

  async function saveTarget(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedId) return;
    const h = Number(targetH);
    if (!Number.isFinite(h) || h < 1 || h > 24) {
      setTargetMsg({ kind: "danger", text: t("rules.targetInvalid") });
      return;
    }
    try {
      await api.updateBusinessSettings(selectedId, { productive_target_s: Math.round(h * 3600) });
      await reload();
      setTargetMsg({ kind: "success", text: t("rules.targetSaved") });
    } catch {
      setTargetMsg({ kind: "danger", text: t("rules.saveError") });
    }
  }

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <div className="admin-pagehead">
        <div className="admin-pagehead__main">
          <h1 className="admin-h1">{t("title")}</h1>
          <p className="admin-sub">{t("rules.subtitle")}</p>
        </div>
        <div className="admin-pagehead__actions">
          <PeriodPicker period={period} />
        </div>
      </div>

      <PerformanceTabs />

      <div className="bibo-card bibo-card--default admin-cardpad" style={{ marginBottom: 14 }}>
        <form className="admin-perftarget" onSubmit={saveTarget}>
          <label htmlFor="perf-target">
            <span className="admin-paneltitle">{t("rules.target")}</span>
            <span className="admin-muted">{t("rules.targetHelp")}</span>
          </label>
          <span className="admin-perftarget__input">
            <input
              id="perf-target"
              className="bibo-input"
              type="number"
              min={1}
              max={24}
              step={0.5}
              value={targetH}
              onChange={(e) => {
                setTargetH(e.target.value);
                setTargetMsg(null);
              }}
            />
            <span>{t("rules.hoursPerDay")}</span>
          </span>
          <button type="submit" className="bibo-btn bibo-btn--secondary bibo-btn--sm">
            {t("rules.save")}
          </button>
        </form>
        {targetMsg && <Notice kind={targetMsg.kind}>{targetMsg.text}</Notice>}
      </div>

      <div className="admin-perfrules__bar">
        <div className="bibo-tabs bibo-tabs--pill" role="tablist">
          {(["app", "domain"] as const).map((k) => (
            <button
              key={k}
              role="tab"
              aria-selected={type === k}
              className={`bibo-tab${type === k ? " bibo-tab--on" : ""}`}
              onClick={() => setType(k)}
            >
              {t(k === "app" ? "rules.apps" : "rules.websites")}
            </button>
          ))}
        </div>
        <div className="bibo-seg bibo-seg--sm" role="tablist">
          {(["all", "unclassified"] as const).map((k) => (
            <button
              key={k}
              role="tab"
              aria-selected={filter === k}
              className={`bibo-seg__opt${filter === k ? " bibo-seg__opt--on" : ""}`}
              onClick={() => setFilter(k)}
            >
              {k === "all" ? t("rules.showAll") : t("rules.showUnclassified", { count: unclassifiedCount })}
            </button>
          ))}
        </div>
      </div>

      {error && <Notice kind="danger">{error}</Notice>}

      <div className="bibo-card bibo-card--default admin-tablecard">
        <form className="admin-perfrules__add" onSubmit={addManual}>
          <input
            className="bibo-input"
            placeholder={type === "app" ? t("rules.addAppPlaceholder") : t("rules.addDomainPlaceholder")}
            value={newPattern}
            onChange={(e) => setNewPattern(e.target.value)}
            maxLength={255}
          />
          <button type="submit" className="bibo-btn bibo-btn--secondary bibo-btn--sm" disabled={!newPattern.trim()}>
            {t("rules.add")}
          </button>
        </form>
        {loading && !usage ? (
          <div className="admin-cardpad">
            <Spinner label={t("rules.loading")} />
          </div>
        ) : rows.length === 0 ? (
          <div className="admin-cardpad admin-muted">{t("rules.empty")}</div>
        ) : (
          <table className="admin-table">
            <thead>
              <tr>
                <th>{type === "app" ? t("rules.colApp") : t("rules.colDomain")}</th>
                <th className="r">{t("rules.colTime")}</th>
                <th className="r">{t("rules.colCategory")}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.key}>
                  <td>
                    <span className="admin-perfname">
                      <span className="dot" style={{ background: CATEGORY_COLOR[r.category] }} />
                      {r.name}
                    </span>
                  </td>
                  <td className="r admin-dur">{r.duration_s > 0 ? fmtDuration(r.duration_s) : "—"}</td>
                  <td className="r">
                    <div
                      className="bibo-seg bibo-seg--sm admin-perfcat"
                      role="radiogroup"
                      aria-label={t("rules.colCategory")}
                      aria-busy={saving === r.name}
                    >
                      {CATEGORIES.map((c) => (
                        <button
                          key={c}
                          type="button"
                          role="radio"
                          aria-checked={r.category === c}
                          disabled={saving === r.name}
                          className={`bibo-seg__opt${r.category === c ? " bibo-seg__opt--on" : ""}`}
                          style={r.category === c ? { color: CATEGORY_COLOR[c] } : undefined}
                          onClick={() => r.category !== c && classify(r.name, c)}
                        >
                          {t(`category.${c}`)}
                        </button>
                      ))}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <p className="admin-muted" style={{ marginTop: 12, fontSize: 12.5 }}>
        {type === "domain" ? t("rules.domainHelp") : t("rules.appHelp")}
      </p>
    </div>
  );
}
