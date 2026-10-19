//go:build integration

package store

import (
	"context"
	"os"
	"testing"
	"time"

	"ctracking/backend/internal/db"
	"ctracking/backend/internal/orgemp"

	"github.com/google/uuid"
)

// Roadmap foundation: roles/scope, attendance, agent health, audit, department policies.
func TestRoadmapFoundation(t *testing.T) {
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
	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM businesses WHERE tenant_id = $1`, tenant)
		pool.Exec(ctx, `DELETE FROM users WHERE email LIKE $1`, tenant+"%")
	})
	owner := orgemp.Profile{ID: uuid.NewString(), Email: tenant + "-o@x.co", FullName: "Owner", TenantID: tenant}
	lead := orgemp.Profile{ID: uuid.NewString(), Email: tenant + "-m@x.co", FullName: "Lead", TenantID: tenant,
		DepartmentID: "eng", DepartmentName: "Engineering", IsDepartmentLead: true}
	emp := orgemp.Profile{ID: uuid.NewString(), Email: tenant + "-e@x.co", FullName: "Emp", TenantID: tenant,
		DepartmentID: "eng", DepartmentName: "Engineering", ManagerOrgUserID: lead.ID}
	other := orgemp.Profile{ID: uuid.NewString(), Email: tenant + "-s@x.co", FullName: "Sales", TenantID: tenant,
		DepartmentID: "sales", DepartmentName: "Sales"}
	ou, biz, err := s.ProvisionOrgUser(ctx, OrgMember{Profile: owner, Role: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SyncDirectory(ctx, tenant, []OrgMember{
		{Profile: owner, Role: "owner"}, {Profile: lead, Role: "manager"}, {Profile: emp}, {Profile: other},
	}); err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, p := range []orgemp.Profile{lead, emp, other} {
		var id string
		pool.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, p.Email).Scan(&id)
		ids[p.FullName] = id
	}

	// M9 roles: department lead → manager scoped to their department.
	m, err := s.MembershipOf(ctx, ids["Lead"], biz.ID)
	if err != nil || m.Role != "manager" || m.DepartmentID != "eng" {
		t.Fatalf("manager membership: %+v %v", m, err)
	}

	// M6 attendance from activity samples; a department filter hides other departments.
	dev := uuid.NewString()
	day := time.Now().Add(-2 * time.Hour).Unix()
	for i, u := range []string{ids["Emp"], ids["Sales"]} {
		if _, err := pool.Exec(ctx, `INSERT INTO activity_samples (client_uuid, user_id, business_id, device_id, ts, app_name, duration_s, client_updated_at)
			VALUES ($1, $2, $3, $4, $5, 'code', 600, 0)`, uuid.NewString(), u, biz.ID, dev, day+int64(i)); err != nil {
			t.Fatal(err)
		}
	}
	all, err := s.Attendance(ctx, biz.ID, day-3600, day+7200, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("attendance all: %d %v", len(all), err)
	}
	eng, _ := s.Attendance(ctx, biz.ID, day-3600, day+7200, "eng")
	if len(eng) != 1 || eng[0].ActiveS != 600 || eng[0].ObservedS != 600 {
		t.Fatalf("attendance eng: %+v", eng)
	}

	// M1/M10 agent health.
	if err := s.RecordAgentCheckin(ctx, ids["Emp"], AgentCheckin{DeviceID: dev, AgentVersion: "1.6.0", OS: "windows", Hostname: "PC-01"}); err != nil {
		t.Fatal(err)
	}
	agents, err := s.ListAgents(ctx, biz.ID, "sales")
	if err != nil || len(agents) != 0 {
		t.Fatalf("agents sales: %+v %v", agents, err)
	}
	agents, _ = s.ListAgents(ctx, biz.ID, "eng")
	if len(agents) != 1 || agents[0].OS != "windows" || agents[0].AgentVersion != "1.6.0" {
		t.Fatalf("agents eng: %+v", agents)
	}

	// M9 audit.
	if err := s.RecordAudit(ctx, biz.ID, ou.ID, "screenshot.view", "screenshot", "x", map[string]string{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	if list, err := s.ListAudit(ctx, biz.ID, 10, 0); err != nil || len(list) != 1 || list[0].ActorName != "Owner" {
		t.Fatalf("audit: %+v %v", list, err)
	}

	// M10 department policy override.
	start, end := "09:00", "18:00"
	idle := 300
	if err := s.UpsertDepartmentPolicy(ctx, biz.ID, DepartmentPolicy{DepartmentID: "eng", WorkStart: &start, WorkEnd: &end,
		WorkDays: []int16{1, 2, 3, 4, 5}, IdleThresholdS: &idle}); err != nil {
		t.Fatal(err)
	}
	p, err := s.GetDepartmentPolicy(ctx, biz.ID, "eng")
	if err != nil || *p.WorkStart != "09:00" || *p.IdleThresholdS != 300 || p.ScreenshotIntervalS != nil {
		t.Fatalf("policy: %+v %v", p, err)
	}
}
