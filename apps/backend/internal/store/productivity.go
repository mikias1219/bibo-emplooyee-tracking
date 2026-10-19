package store

import (
	"context"
	"errors"
	"sort"

	"ctracking/backend/internal/productivity"

	"github.com/jackc/pgx/v5"
)

// hostExpr extracts a lowercase host from a stored browser URL in SQL so
// grouping happens per host, not per full URL; productivity.NormalizeDomain
// finishes the job in Go (www., trailing dots). NULL for scheme-less URLs.
const hostExpr = `substring(lower(url) from '^[a-z][a-z0-9+.-]*://(?:[^@/?#]*@)?([^/?#:]+)')`

// ListProductivityRules returns a business's rules, apps first then domains.
func (s *Store) ListProductivityRules(ctx context.Context, businessID string) ([]productivity.Rule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, match_type, pattern, category FROM productivity_rules
		  WHERE business_id = $1 ORDER BY match_type, pattern`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []productivity.Rule{}
	for rows.Next() {
		var r productivity.Rule
		if err := rows.Scan(&r.ID, &r.MatchType, &r.Pattern, &r.Category); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpsertProductivityRule sets the category for (match_type, pattern); the
// pattern must already be normalised.
func (s *Store) UpsertProductivityRule(ctx context.Context, businessID string, r productivity.Rule) (productivity.Rule, error) {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO productivity_rules (business_id, match_type, pattern, category)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (business_id, match_type, pattern)
		 DO UPDATE SET category = EXCLUDED.category, updated_at = now()
		 RETURNING id`, businessID, r.MatchType, r.Pattern, r.Category).Scan(&r.ID)
	return r, err
}

// DeleteProductivityRule removes a rule (its target becomes unclassified).
func (s *Store) DeleteProductivityRule(ctx context.Context, businessID, ruleID string) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM productivity_rules WHERE business_id = $1 AND id = $2`, businessID, ruleID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ProductiveTargetS returns the business's expected productive seconds per day.
func (s *Store) ProductiveTargetS(ctx context.Context, businessID string) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx,
		`SELECT productive_target_s FROM businesses WHERE id = $1`, businessID).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return v, err
}

// UsageItem is one app or domain seen in a period, with its current category.
type UsageItem struct {
	Name      string                `json:"name"`
	DurationS int64                 `json:"duration_s"`
	Category  productivity.Category `json:"category"`
}

// ProductivityUsage lists every app and domain the business's members used in
// [from, to), most-used first, so owners can classify what actually matters.
func (s *Store) ProductivityUsage(ctx context.Context, businessID string, from, to int64, m *productivity.Matcher) (apps, domains []UsageItem, err error) {
	rows, err := s.pool.Query(ctx,
		`SELECT app_name, sum(duration_s) FROM activity_samples
		  WHERE business_id = $1 AND ts >= $2 AND ts < $3
		  GROUP BY app_name`, businessID, from, to)
	if err != nil {
		return nil, nil, err
	}
	// App names can differ only by case across OSes; merge them.
	appSum := map[string]*UsageItem{}
	for rows.Next() {
		var name string
		var d int64
		if err := rows.Scan(&name, &d); err != nil {
			rows.Close()
			return nil, nil, err
		}
		key := productivity.NormalizeApp(name)
		if it, ok := appSum[key]; ok {
			it.DurationS += d
		} else {
			appSum[key] = &UsageItem{Name: name, DurationS: d, Category: m.App(name)}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	rows, err = s.pool.Query(ctx,
		`SELECT `+hostExpr+` AS host, sum(duration_s) FROM browser_visits
		  WHERE business_id = $1 AND ts >= $2 AND ts < $3
		  GROUP BY host`, businessID, from, to)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	domSum := map[string]*UsageItem{}
	for rows.Next() {
		var host *string
		var d int64
		if err := rows.Scan(&host, &d); err != nil {
			return nil, nil, err
		}
		if host == nil {
			continue
		}
		h := productivity.NormalizeDomain(*host)
		if h == "" {
			continue
		}
		if it, ok := domSum[h]; ok {
			it.DurationS += d
		} else {
			domSum[h] = &UsageItem{Name: h, DurationS: d, Category: m.Domain(h)}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return sortedUsage(appSum), sortedUsage(domSum), nil
}

func sortedUsage(m map[string]*UsageItem) []UsageItem {
	out := make([]UsageItem, 0, len(m))
	for _, it := range m {
		out = append(out, *it)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DurationS != out[j].DurationS {
			return out[i].DurationS > out[j].DurationS
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// DayUsage is one employee-day of raw usage for productivity.Classify.
type DayUsage struct {
	UserID  string
	Day     string // YYYY-MM-DD in the requested time zone
	Apps    []productivity.Usage
	Domains []productivity.Usage
}

// PerformanceUsage returns per employee-day app and domain usage for a
// business in [from, to), with days bucketed in the IANA time zone tz
// (validated by the caller). employeeID, when non-empty, restricts to one member.
func (s *Store) PerformanceUsage(ctx context.Context, businessID string, from, to int64, tz, employeeID string) ([]DayUsage, error) {
	type key struct{ user, day string }
	days := map[key]*DayUsage{}
	get := func(u, d string) *DayUsage {
		k := key{u, d}
		if du, ok := days[k]; ok {
			return du
		}
		du := &DayUsage{UserID: u, Day: d}
		days[k] = du
		return du
	}

	userFilter := ""
	args := []any{businessID, from, to, tz}
	if employeeID != "" {
		userFilter = " AND user_id = $5"
		args = append(args, employeeID)
	}

	rows, err := s.pool.Query(ctx,
		`SELECT user_id::text, to_char(to_timestamp(ts) AT TIME ZONE $4, 'YYYY-MM-DD') AS day,
		        app_name, sum(duration_s)
		   FROM activity_samples
		  WHERE business_id = $1 AND ts >= $2 AND ts < $3`+userFilter+`
		  GROUP BY 1, 2, 3`, args...)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var u, d, app string
		var dur int64
		if err := rows.Scan(&u, &d, &app, &dur); err != nil {
			rows.Close()
			return nil, err
		}
		du := get(u, d)
		du.Apps = append(du.Apps, productivity.Usage{Name: app, DurationS: dur})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx,
		`SELECT user_id::text, to_char(to_timestamp(ts) AT TIME ZONE $4, 'YYYY-MM-DD') AS day,
		        `+hostExpr+` AS host, sum(duration_s)
		   FROM browser_visits
		  WHERE business_id = $1 AND ts >= $2 AND ts < $3`+userFilter+`
		  GROUP BY 1, 2, 3`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u, d string
		var host *string
		var dur int64
		if err := rows.Scan(&u, &d, &host, &dur); err != nil {
			return nil, err
		}
		h := ""
		if host != nil {
			h = *host
		}
		// Only days that also have app time matter: domain time never adds
		// active time (see productivity.Classify).
		if du, ok := days[key{u, d}]; ok {
			du.Domains = append(du.Domains, productivity.Usage{Name: h, DurationS: dur})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]DayUsage, 0, len(days))
	for _, du := range days {
		out = append(out, *du)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Day != out[j].Day {
			return out[i].Day < out[j].Day
		}
		return out[i].UserID < out[j].UserID
	})
	return out, nil
}

// Member is a business member shown in performance reports.
type Member struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	Role        string `json:"role"`
}

// ListMembers returns a business's owner and employees.
func (s *Store) ListMembers(ctx context.Context, businessID string) ([]Member, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.display_name, COALESCE(u.email, ''), COALESCE(u.username, ''), m.role
		   FROM memberships m JOIN users u ON u.id = m.user_id
		  WHERE m.business_id = $1 AND m.role IN ('owner','employee') AND m.deactivated_at IS NULL
		  ORDER BY u.display_name`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.DisplayName, &m.Email, &m.Username, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
