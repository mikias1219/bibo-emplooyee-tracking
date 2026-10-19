import { useTranslation } from "react-i18next";
import { useAdmin } from "../../context";
import { useWorkspace } from "../../hooks";
import { fmtDuration } from "../../format";
import { Empty, Notice, Spinner } from "../../ui";
import { PageHead } from "../../components/PageHead";
import { PeriodPicker, usePeriod } from "../../components/PeriodPicker";
import { useLoad } from "../../components/useLoad";

const clock = (unix: number) => new Date(unix * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });

/**
 * Roadmap M6 — Observed attendance (foundation): first/last computer activity and
 * active vs idle time per employee per day. Schedule comparison (late start, early
 * finish, overtime) and no-activity days build on this in the attendance sprint.
 */
export function Attendance() {
  const { t } = useTranslation("admin");
  const { api } = useAdmin();
  const { selectedId } = useWorkspace();
  const period = usePeriod();
  const { from, to } = period.window;
  const { data, loading, error } = useLoad(
    () => api.listAttendance(selectedId!, from, to),
    [api, selectedId, from, to],
    !!selectedId,
  );
  const days = data?.days ?? [];

  return (
    <div className="admin-wrap" style={{ paddingBottom: 32 }}>
      <PageHead title={t("attendance.title")} subtitle={t("attendance.subtitle")} actions={<PeriodPicker period={period} />} />
      {error && <Notice kind="danger">{error}</Notice>}
      {loading && <Spinner />}
      {!loading && !error && days.length === 0 && <Empty>{t("attendance.empty")}</Empty>}
      {days.length > 0 && (
        <div className="bibo-card bibo-card--default admin-tablecard">
          <table className="admin-table">
            <thead>
              <tr>
                <th>{t("attendance.day")}</th>
                <th>{t("common.employee")}</th>
                <th>{t("common.department")}</th>
                <th>{t("attendance.first")}</th>
                <th>{t("attendance.last")}</th>
                <th className="r">{t("attendance.observed")}</th>
                <th className="r">{t("attendance.active")}</th>
                <th className="r">{t("attendance.idle")}</th>
              </tr>
            </thead>
            <tbody>
              {days.map((d) => (
                <tr key={`${d.user_id}-${d.day}`}>
                  <td className="num">{d.day}</td>
                  <td>{d.display_name}</td>
                  <td>{d.department_name || "—"}</td>
                  <td className="num">{clock(d.first_activity)}</td>
                  <td className="num">{clock(d.last_activity)}</td>
                  <td className="r num">{fmtDuration(d.observed_s)}</td>
                  <td className="r num">{fmtDuration(d.active_s)}</td>
                  <td className="r num">{fmtDuration(Math.max(0, d.observed_s - d.active_s))}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
