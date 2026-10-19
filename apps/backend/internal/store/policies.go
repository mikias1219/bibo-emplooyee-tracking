package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// DepartmentPolicy overrides the organization monitoring policy for one department
// (roadmap M10). nil fields inherit the organization value.
type DepartmentPolicy struct {
	DepartmentID          string   `json:"department_id"`
	WorkStart             *string  `json:"work_start"` // "HH:MM"
	WorkEnd               *string  `json:"work_end"`
	WorkDays              []int16  `json:"work_days"` // ISO 1=Mon … 7=Sun; nil = inherit
	IdleThresholdS        *int     `json:"idle_threshold_s"`
	ScreenshotIntervalS   *int     `json:"screenshot_interval_s"`
	ScreenshotSkipApps    []string `json:"screenshot_skip_apps"`
	ScreenshotSkipDomains []string `json:"screenshot_skip_domains"`
}

const policyCols = `department_id, to_char(work_start, 'HH24:MI'), to_char(work_end, 'HH24:MI'), work_days,
	idle_threshold_s, screenshot_interval_s, screenshot_skip_apps, screenshot_skip_domains`

func scanPolicy(r pgx.Row) (DepartmentPolicy, error) {
	var p DepartmentPolicy
	err := r.Scan(&p.DepartmentID, &p.WorkStart, &p.WorkEnd, &p.WorkDays,
		&p.IdleThresholdS, &p.ScreenshotIntervalS, &p.ScreenshotSkipApps, &p.ScreenshotSkipDomains)
	return p, err
}

// ListDepartmentPolicies returns every department override of a business.
func (s *Store) ListDepartmentPolicies(ctx context.Context, businessID string) ([]DepartmentPolicy, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+policyCols+` FROM department_policies WHERE business_id = $1 ORDER BY department_id`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DepartmentPolicy{}
	for rows.Next() {
		p, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetDepartmentPolicy returns one department's override, or ErrNotFound.
func (s *Store) GetDepartmentPolicy(ctx context.Context, businessID, departmentID string) (DepartmentPolicy, error) {
	p, err := scanPolicy(s.pool.QueryRow(ctx,
		`SELECT `+policyCols+` FROM department_policies WHERE business_id = $1 AND department_id = $2`, businessID, departmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return DepartmentPolicy{}, ErrNotFound
	}
	return p, err
}

// UpsertDepartmentPolicy replaces a department's override.
func (s *Store) UpsertDepartmentPolicy(ctx context.Context, businessID string, p DepartmentPolicy) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO department_policies (business_id, department_id, work_start, work_end, work_days,
		       idle_threshold_s, screenshot_interval_s, screenshot_skip_apps, screenshot_skip_domains, updated_at)
		VALUES ($1, $2, $3::time, $4::time, $5, $6, $7, $8, $9, now())
		ON CONFLICT (business_id, department_id) DO UPDATE SET
		  work_start = EXCLUDED.work_start, work_end = EXCLUDED.work_end, work_days = EXCLUDED.work_days,
		  idle_threshold_s = EXCLUDED.idle_threshold_s, screenshot_interval_s = EXCLUDED.screenshot_interval_s,
		  screenshot_skip_apps = EXCLUDED.screenshot_skip_apps,
		  screenshot_skip_domains = EXCLUDED.screenshot_skip_domains, updated_at = now()`,
		businessID, p.DepartmentID, p.WorkStart, p.WorkEnd, p.WorkDays, p.IdleThresholdS,
		p.ScreenshotIntervalS, p.ScreenshotSkipApps, p.ScreenshotSkipDomains)
	return err
}

// DeleteDepartmentPolicy removes an override (the department inherits again).
func (s *Store) DeleteDepartmentPolicy(ctx context.Context, businessID, departmentID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM department_policies WHERE business_id = $1 AND department_id = $2`, businessID, departmentID)
	return err
}
