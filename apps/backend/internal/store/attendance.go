package store

import "context"

// AttendanceDay is one employee's observed attendance for one local day (roadmap M6).
type AttendanceDay struct {
	UserID         string `json:"user_id"`
	DisplayName    string `json:"display_name"`
	DepartmentName string `json:"department_name"`
	Day            string `json:"day"`            // YYYY-MM-DD in the business time zone
	FirstActivity  int64  `json:"first_activity"` // unix seconds
	LastActivity   int64  `json:"last_activity"`  // unix seconds (end of the last interval)
	ActiveS        int64  `json:"active_s"`
	ObservedS      int64  `json:"observed_s"` // last - first; idle = observed - active
}

// Attendance derives first/last activity and active time per employee per day from
// activity samples, bucketed in the business time zone. departmentID "" = all.
func (s *Store) Attendance(ctx context.Context, businessID string, from, to int64, departmentID string) ([]AttendanceDay, error) {
	rows, err := s.pool.Query(ctx, `
		WITH tz AS (SELECT timezone FROM businesses WHERE id = $1)
		SELECT u.id::text, u.display_name, COALESCE(m.department_name, ''),
		       to_char(to_timestamp(a.ts) AT TIME ZONE (SELECT timezone FROM tz), 'YYYY-MM-DD') AS day,
		       min(a.ts), max(a.ts + a.duration_s), sum(a.duration_s)
		  FROM activity_samples a
		  JOIN memberships m ON m.user_id = a.user_id AND m.business_id = a.business_id AND m.deactivated_at IS NULL
		  JOIN users u ON u.id = a.user_id
		 WHERE a.business_id = $1 AND a.ts >= $2 AND a.ts < $3 AND ($4 = '' OR m.department_id = $4)
		 GROUP BY u.id, u.display_name, m.department_name, day
		 ORDER BY day DESC, u.display_name`, businessID, from, to, departmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttendanceDay{}
	for rows.Next() {
		var d AttendanceDay
		if err := rows.Scan(&d.UserID, &d.DisplayName, &d.DepartmentName, &d.Day,
			&d.FirstActivity, &d.LastActivity, &d.ActiveS); err != nil {
			return nil, err
		}
		d.ObservedS = d.LastActivity - d.FirstActivity
		out = append(out, d)
	}
	return out, rows.Err()
}
