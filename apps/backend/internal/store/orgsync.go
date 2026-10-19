package store

import (
	"context"
	"errors"
	"strings"

	"ctracking/backend/internal/orgemp"
	"ctracking/backend/internal/privacyapps"

	"github.com/jackc/pgx/v5"
)

// ErrDeactivated is returned when an org user was removed from the directory.
var ErrDeactivated = errors.New("membership deactivated")

// OrgMember is the role + department a directory user holds in their tenant.
type OrgMember struct {
	Profile orgemp.Profile
	// Role is 'owner' (admin: all data + settings), 'manager' (own department only)
	// or 'employee'. Resolved from Org&Emp by orgauth.
	Role string
}

// ProvisionOrgUser mirrors a signed-in Org&Emp user locally: it upserts the user
// (matched by firebase uid, then org id, then email — linking a pre-existing local
// account), the tenant's business, and the membership. Returns the local user and
// business. All in one transaction.
func (s *Store) ProvisionOrgUser(ctx context.Context, m OrgMember) (User, Business, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return User{}, Business{}, err
	}
	defer tx.Rollback(ctx)

	userID, err := upsertOrgUserTx(ctx, tx, m.Profile)
	if err != nil {
		return User{}, Business{}, err
	}
	biz, err := tenantBusinessTx(ctx, tx, m.Profile.TenantID, m.Profile.TenantName, userID)
	if err != nil {
		return User{}, Business{}, err
	}
	if err := upsertMembershipTx(ctx, tx, userID, biz.ID, m); err != nil {
		return User{}, Business{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, Business{}, err
	}
	u, err := s.GetUserByID(ctx, userID)
	return u, biz, err
}

// SyncDirectory reconciles a tenant's business with the full Org&Emp roster: every
// listed user is upserted, and org-linked members missing from the list are
// deactivated (their history is kept). Returns how many members are active.
func (s *Store) SyncDirectory(ctx context.Context, tenantID string, members []OrgMember) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var bizID string
	err = tx.QueryRow(ctx, `SELECT id FROM businesses WHERE tenant_id = $1`, tenantID).Scan(&bizID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound // business is created on the first sign-in
	}
	if err != nil {
		return 0, err
	}

	seen := make([]string, 0, len(members))
	for _, m := range members {
		if m.Profile.Deleted || m.Profile.Email == "" {
			continue
		}
		userID, err := upsertOrgUserTx(ctx, tx, m.Profile)
		if err != nil {
			return 0, err
		}
		if err := upsertMembershipTx(ctx, tx, userID, bizID, m); err != nil {
			return 0, err
		}
		seen = append(seen, userID)
	}

	// Only org-linked users are deactivated; never wipe the roster on an empty list.
	if len(seen) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE memberships m SET deactivated_at = now()
			  FROM users u
			 WHERE u.id = m.user_id AND m.business_id = $1
			   AND u.org_user_id IS NOT NULL AND m.deactivated_at IS NULL
			   AND NOT (m.user_id = ANY($2::uuid[]))`, bizID, seen); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(seen), nil
}

// MembershipActive reports whether the user has at least one non-deactivated
// membership (or none at all — a standalone account).
func (s *Store) MembershipActive(ctx context.Context, userID string) (bool, error) {
	var total, active int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*), count(*) FILTER (WHERE deactivated_at IS NULL) FROM memberships WHERE user_id = $1`,
		userID).Scan(&total, &active)
	return total == 0 || active > 0, err
}

// TenantForBusiness returns the Org&Emp tenant id behind a business ("" if none).
func (s *Store) TenantForBusiness(ctx context.Context, businessID string) (string, error) {
	var tenant *string
	err := s.pool.QueryRow(ctx, `SELECT tenant_id FROM businesses WHERE id = $1`, businessID).Scan(&tenant)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil || tenant == nil {
		return "", err
	}
	return *tenant, nil
}

// Department is one department with its active member count.
type Department struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Members int    `json:"members"`
}

// ListDepartments returns the departments present in a business's roster.
func (s *Store) ListDepartments(ctx context.Context, businessID string) ([]Department, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT department_id, COALESCE(max(department_name), ''), count(*)
		  FROM memberships
		 WHERE business_id = $1 AND deactivated_at IS NULL AND department_id IS NOT NULL
		 GROUP BY department_id
		 ORDER BY 2`, businessID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Department{}
	for rows.Next() {
		var d Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Members); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// --- tx helpers ---

func upsertOrgUserTx(ctx context.Context, tx pgx.Tx, p orgemp.Profile) (string, error) {
	email := strings.ToLower(strings.TrimSpace(p.Email))
	var id string
	err := tx.QueryRow(ctx, `
		SELECT id FROM users
		 WHERE ($1 <> '' AND firebase_uid = $1) OR org_user_id = $2::uuid OR ($3 <> '' AND email = $3)
		 ORDER BY (firebase_uid = $1) DESC NULLS LAST, (org_user_id = $2::uuid) DESC NULLS LAST
		 LIMIT 1`, p.FirebaseUID, p.ID, email).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, display_name, org_user_id, firebase_uid, avatar_url)
			VALUES ($1, $2, $3::uuid, $4, $5) RETURNING id`,
			nullableLower(email), p.FullName, p.ID, nullable(p.FirebaseUID), nullable(p.AvatarURL)).Scan(&id)
		if isUniqueViolation(err) {
			return "", ErrConflict
		}
		return id, err
	case err != nil:
		return "", err
	}
	// COALESCE keeps a known firebase uid when the directory list omits it.
	_, err = tx.Exec(ctx, `
		UPDATE users SET email = COALESCE($2, email), display_name = $3, org_user_id = $4::uuid,
		       firebase_uid = COALESCE($5, firebase_uid), avatar_url = $6
		 WHERE id = $1`,
		id, nullableLower(email), p.FullName, p.ID, nullable(p.FirebaseUID), nullable(p.AvatarURL))
	if isUniqueViolation(err) {
		return "", ErrConflict
	}
	return id, err
}

func tenantBusinessTx(ctx context.Context, tx pgx.Tx, tenantID, tenantName, creatorID string) (Business, error) {
	b, err := scanBusiness(tx.QueryRow(ctx, `SELECT `+businessCols+` FROM businesses WHERE tenant_id = $1`, tenantID))
	if err == nil {
		if tenantName != "" && tenantName != b.Name {
			if _, err := tx.Exec(ctx, `UPDATE businesses SET name = $2 WHERE id = $1`, b.ID, tenantName); err != nil {
				return Business{}, err
			}
			b.Name = tenantName
		}
		return b, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Business{}, err
	}
	if tenantName == "" {
		tenantName = "Organization"
	}
	// Same defaults as a self-serve business: privacy mode + curated skip-list.
	return scanBusiness(tx.QueryRow(ctx,
		`INSERT INTO businesses (name, owner_user_id, kind, screenshot_skip_apps, tenant_id)
		 VALUES ($1, $2, 'team', $3, $4) RETURNING `+businessCols,
		tenantName, creatorID, privacyapps.Flat(), tenantID))
}

func upsertMembershipTx(ctx context.Context, tx pgx.Tx, userID, businessID string, m OrgMember) error {
	role := m.Role
	if role != "owner" && role != "manager" {
		role = "employee"
	}
	p := m.Profile
	_, err := tx.Exec(ctx, `
		INSERT INTO memberships (user_id, business_id, role, department_id, department_name, job_title,
		                         is_department_lead, manager_org_user_id, deactivated_at, synced_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::uuid, NULL, now())
		ON CONFLICT (user_id, business_id) DO UPDATE SET
		       role = EXCLUDED.role, department_id = EXCLUDED.department_id,
		       department_name = EXCLUDED.department_name, job_title = EXCLUDED.job_title,
		       is_department_lead = EXCLUDED.is_department_lead,
		       manager_org_user_id = EXCLUDED.manager_org_user_id,
		       deactivated_at = NULL, synced_at = now()`,
		userID, businessID, role, nullable(p.DepartmentID), nullable(p.DepartmentName),
		nullable(p.JobTitle), p.IsDepartmentLead, nullable(p.ManagerOrgUserID))
	return err
}

func nullable(s string) any {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	return s
}
