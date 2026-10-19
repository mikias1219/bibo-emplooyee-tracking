//go:build integration

// Run against a local Postgres: DATABASE_URL=... go test -tags integration ./internal/store/
package store

import (
	"context"
	"os"
	"testing"

	"ctracking/backend/internal/db"
	"ctracking/backend/internal/orgemp"

	"github.com/google/uuid"
)

func TestOrgProvisionAndSync(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	if err := db.Migrate(url); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	s := New(pool)

	tenant := "it-" + uuid.NewString()
	owner := orgemp.Profile{ID: uuid.NewString(), FirebaseUID: "fb-" + tenant, Email: tenant + "-owner@example.com",
		FullName: "Owner One", TenantID: tenant, TenantName: "Acme", RoleSlug: "owner"}
	emp := orgemp.Profile{ID: uuid.NewString(), Email: tenant + "-emp@example.com", FullName: "Emp One",
		TenantID: tenant, DepartmentID: "d1", DepartmentName: "Engineering"}
	gone := orgemp.Profile{ID: uuid.NewString(), Email: tenant + "-gone@example.com", FullName: "Gone", TenantID: tenant}
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM businesses WHERE tenant_id = $1`, tenant)
		pool.Exec(ctx, `DELETE FROM users WHERE email LIKE $1`, tenant+"%")
	})

	u, biz, err := s.ProvisionOrgUser(ctx, OrgMember{Profile: owner, Role: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if biz.Name != "Acme" {
		t.Fatalf("business name %q", biz.Name)
	}
	if ok, _ := s.IsBusinessOwner(ctx, u.ID, biz.ID); !ok {
		t.Fatal("owner should own the tenant business")
	}
	// Second sign-in is idempotent (same user, same business).
	u2, biz2, err := s.ProvisionOrgUser(ctx, OrgMember{Profile: owner, Role: "owner"})
	if err != nil || u2.ID != u.ID || biz2.ID != biz.ID {
		t.Fatalf("re-provision: %v %s/%s", err, u2.ID, biz2.ID)
	}

	all := []OrgMember{{Profile: owner, Role: "owner"}, {Profile: emp}, {Profile: gone}}
	if n, err := s.SyncDirectory(ctx, tenant, all); err != nil || n != 3 {
		t.Fatalf("sync: n=%d err=%v", n, err)
	}
	// "gone" leaves the org → deactivated, hidden, can't refresh.
	if _, err := s.SyncDirectory(ctx, tenant, all[:2]); err != nil {
		t.Fatal(err)
	}
	emps, err := s.ListEmployees(ctx, biz.ID)
	if err != nil || len(emps) != 1 || emps[0].DepartmentName != "Engineering" {
		t.Fatalf("employees: %+v err=%v", emps, err)
	}
	var goneID string
	pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, gone.Email).Scan(&goneID)
	if active, _ := s.MembershipActive(ctx, goneID); active {
		t.Fatal("removed user should be inactive")
	}
	if ok, _ := s.IsBusinessOwner(ctx, emps[0].ID, biz.ID); ok {
		t.Fatal("employee must not own the business")
	}
	if ok, _ := s.OwnsEmployee(ctx, u.ID, emps[0].ID); !ok {
		t.Fatal("owner should see the employee")
	}
	if bid, err := s.ResolveBusinessForUser(ctx, emps[0].ID, nil); err != nil || bid != biz.ID {
		t.Fatalf("resolve: %s %v", bid, err)
	}
	r, err := s.Roster(ctx, biz.ID, 0, 1, "d1")
	if err != nil || len(r) != 1 {
		t.Fatalf("dept roster: %d %v", len(r), err)
	}
	if d, _ := s.ListDepartments(ctx, biz.ID); len(d) != 1 || d[0].Members != 1 {
		t.Fatalf("departments: %+v", d)
	}
}
