package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// ErrForbidden is returned when a user acts on a business they don't own.
var ErrForbidden = errors.New("forbidden")

// Business is a company/team owned by a user.
type Business struct {
	ID                      string   `json:"id"`
	Name                    string   `json:"name"`
	Kind                    string   `json:"kind"` // 'team' | 'family'
	OwnerUserID             string   `json:"owner_user_id"`
	ScreenshotRetentionDays *int     `json:"screenshot_retention_days"`
	ScreenshotIntervalS     int      `json:"screenshot_interval_s"`
	IdleThresholdS          int      `json:"idle_threshold_s"`
	AllowEmployeeOverride   bool     `json:"allow_employee_override"`
	ScreenshotMode          string   `json:"screenshot_mode"` // 'privacy' | 'normal'
	ScreenshotSkipApps      []string `json:"screenshot_skip_apps"`
	ProductiveTargetS       int      `json:"productive_target_s"` // expected productive seconds per day
}

// businessCols is the column list backing a Business scan (see scanBusiness).
const businessCols = "id, name, kind, owner_user_id, screenshot_retention_days, screenshot_interval_s, idle_threshold_s, allow_employee_override, screenshot_mode, screenshot_skip_apps, productive_target_s"

// Employee is a member with the employee role within a business.
type Employee struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	// Org&Emp directory fields (empty for self-serve accounts).
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	JobTitle       string `json:"job_title"`
	AvatarURL      string `json:"avatar_url"`
}

// scanner is satisfied by pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanBusiness(s scanner) (Business, error) {
	var b Business
	err := s.Scan(&b.ID, &b.Name, &b.Kind, &b.OwnerUserID, &b.ScreenshotRetentionDays,
		&b.ScreenshotIntervalS, &b.IdleThresholdS, &b.AllowEmployeeOverride,
		&b.ScreenshotMode, &b.ScreenshotSkipApps, &b.ProductiveTargetS)
	return b, err
}

// ListBusinessesOwnedBy returns the businesses a user manages (owner membership).
func (s *Store) ListBusinessesOwnedBy(ctx context.Context, ownerID string) ([]Business, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+businessCols+` FROM businesses
		  WHERE id IN (SELECT business_id FROM memberships
		                WHERE user_id = $1 AND role = 'owner' AND deactivated_at IS NULL)
		  ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Business{}
	for rows.Next() {
		b, err := scanBusiness(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetBusiness returns a business by id.
func (s *Store) GetBusiness(ctx context.Context, id string) (Business, error) {
	return getBusiness(ctx, s.pool, id)
}

// ListEmployees returns the employee members of a business.
func (s *Store) ListEmployees(ctx context.Context, businessID string) ([]Employee, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, COALESCE(u.email, ''), COALESCE(u.username, ''), u.display_name,
		        COALESCE(m.department_id, ''), COALESCE(m.department_name, ''),
		        COALESCE(m.job_title, ''), COALESCE(u.avatar_url, '')
		   FROM memberships m
		   JOIN users u ON u.id = m.user_id
		  WHERE m.business_id = $1 AND m.role = 'employee' AND m.deactivated_at IS NULL
		  ORDER BY u.display_name`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Employee{}
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.Email, &e.Username, &e.DisplayName,
			&e.DepartmentID, &e.DepartmentName, &e.JobTitle, &e.AvatarURL); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// settableColumns whitelists the business columns owners may PATCH, guarding the
// dynamic UPDATE against arbitrary column names.
var settableColumns = map[string]bool{
	"screenshot_retention_days": true,
	"screenshot_interval_s":     true,
	"idle_threshold_s":          true,
	"allow_employee_override":   true,
	"screenshot_mode":           true,
	"screenshot_skip_apps":      true,
	"productive_target_s":       true,
}

// UpdateBusinessSettings updates only the provided columns (keys must be in
// settableColumns; values are already typed by the caller). A nil value sets NULL
// (used for "keep screenshots forever").
func (s *Store) UpdateBusinessSettings(ctx context.Context, businessID string, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	sets := make([]string, 0, len(fields))
	args := []any{businessID}
	for col, val := range fields {
		if !settableColumns[col] {
			return fmt.Errorf("not a settable column: %s", col)
		}
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	q := fmt.Sprintf("UPDATE businesses SET %s WHERE id = $1", strings.Join(sets, ", "))
	ct, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CapturePolicy is the org-controlled capture configuration the desktop applies.
type CapturePolicy struct {
	ScreenshotIntervalS     int      `json:"screenshot_interval_s"`
	IdleThresholdS          int      `json:"idle_threshold_s"`
	ScreenshotRetentionDays *int     `json:"screenshot_retention_days"`
	AllowEmployeeOverride   bool     `json:"allow_employee_override"`
	Kind                    string   `json:"kind"` // 'team' | 'family' — drives onboarding copy
	ScreenshotMode          string   `json:"screenshot_mode"`
	ScreenshotSkipApps      []string `json:"screenshot_skip_apps"`
}

// PolicyForUser returns the capture policy for the user's business, or nil when the
// user belongs to no business (a standalone user → the desktop keeps local defaults).
func (s *Store) PolicyForUser(ctx context.Context, userID string) (*CapturePolicy, error) {
	bizID, err := s.ResolveBusinessForUser(ctx, userID, nil)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p CapturePolicy
	err = s.pool.QueryRow(ctx,
		`SELECT screenshot_interval_s, idle_threshold_s, screenshot_retention_days, allow_employee_override, kind, screenshot_mode, screenshot_skip_apps
		   FROM businesses WHERE id = $1`, bizID,
	).Scan(&p.ScreenshotIntervalS, &p.IdleThresholdS, &p.ScreenshotRetentionDays, &p.AllowEmployeeOverride, &p.Kind, &p.ScreenshotMode, &p.ScreenshotSkipApps)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// --- transaction-scoped helpers (work with both *pgxpool.Pool and pgx.Tx) ---

// rowQuerier is satisfied by *pgxpool.Pool and pgx.Tx.
type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getBusiness(ctx context.Context, q rowQuerier, id string) (Business, error) {
	b, err := scanBusiness(q.QueryRow(ctx, `SELECT `+businessCols+` FROM businesses WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Business{}, ErrNotFound
	}
	if err != nil {
		return Business{}, err
	}
	return b, nil
}
