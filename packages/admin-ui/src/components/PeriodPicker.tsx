import { useState } from "react";
import { useTranslation } from "react-i18next";
import { dayRangeToUnix, daysAgoIso, isoDate } from "../format";

/** Period picker: last 7 / 30 days or a custom range. Returns the unix window. */
export function usePeriod() {
  const [preset, setPreset] = useState<"7" | "30" | "custom">("7");
  const [from, setFrom] = useState(() => daysAgoIso(6));
  const [to, setTo] = useState(() => isoDate(new Date()));
  const [fromDate, toDate] =
    preset === "custom" ? [from, to] : [daysAgoIso(Number(preset) - 1), isoDate(new Date())];
  const window = dayRangeToUnix(fromDate, toDate);
  return { preset, setPreset, from, setFrom, to, setTo, window };
}

export function PeriodPicker({ period }: { period: ReturnType<typeof usePeriod> }) {
  const { t } = useTranslation("admin-performance");
  const today = isoDate(new Date());
  const opts = [
    ["7", t("period.last7")],
    ["30", t("period.last30")],
    ["custom", t("period.custom")],
  ] as const;
  return (
    <div className="admin-datemode">
      <div className="bibo-seg bibo-seg--sm" role="tablist" aria-label={t("period.label")}>
        {opts.map(([k, label]) => (
          <button
            key={k}
            role="tab"
            aria-selected={period.preset === k}
            className={`bibo-seg__opt${period.preset === k ? " bibo-seg__opt--on" : ""}`}
            onClick={() => period.setPreset(k)}
          >
            {label}
          </button>
        ))}
      </div>
      {period.preset === "custom" && (
        <>
          <span className="admin-datefield">
            <span className="admin-datefield__lbl">{t("period.from")}</span>
            <input type="date" value={period.from} max={period.to} onChange={(e) => period.setFrom(e.target.value)} />
          </span>
          <span className="admin-datefield">
            <span className="admin-datefield__lbl">{t("period.to")}</span>
            <input type="date" value={period.to} min={period.from} max={today} onChange={(e) => period.setTo(e.target.value)} />
          </span>
        </>
      )}
    </div>
  );
}
