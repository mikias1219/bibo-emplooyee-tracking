// Package orgemp is an HTTP client for the Selamnew Org&Emp (core) service — the
// shared source of users, organizations (tenants) and departments used by the
// Operation and IE products. It mirrors Operation's OrgEmpClientService:
// requests forward the caller's Firebase bearer token plus a `tenantid` header.
package orgemp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrNotFound is returned when Org&Emp has no user for the given identity.
var ErrNotFound = errors.New("org user not found")

// Profile is the subset of an Org&Emp user this app mirrors.
type Profile struct {
	ID               string // Org&Emp user id (uuid)
	FirebaseUID      string
	Email            string
	FullName         string
	AvatarURL        string
	TenantID         string
	TenantName       string
	RoleSlug         string // Core role slug (falls back to role name)
	DepartmentID     string
	DepartmentName   string
	JobTitle         string
	IsDepartmentLead bool
	ManagerOrgUserID string // reportingTo: the Org&Emp user this person reports to
	Deleted          bool
}

// Client calls one Org&Emp deployment.
type Client struct {
	baseURL string
	http    *http.Client
	crypto  *Decrypter // nil = payloads are plain JSON (ENVIRONMENT=test)
}

// New builds a client. dec may be nil when payload encryption is off.
func New(baseURL string, dec *Decrypter) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: 20 * time.Second}, crypto: dec}
}

// UserByFirebaseID resolves the signed-in user (GET /users/firebase/{uid}). No
// tenant header: this is how the caller's tenant is discovered.
func (c *Client) UserByFirebaseID(ctx context.Context, firebaseUID, bearer string) (Profile, error) {
	raw, err := c.get(ctx, "/users/firebase/"+url.PathEscape(firebaseUID), nil, "", bearer)
	if err != nil {
		return Profile{}, err
	}
	rec := unwrapRecord(raw)
	if rec == nil {
		return Profile{}, ErrNotFound
	}
	p := mapProfile(rec)
	if p.FirebaseUID == "" {
		p.FirebaseUID = firebaseUID
	}
	if p.ID == "" {
		return Profile{}, ErrNotFound
	}
	return p, nil
}

// Users lists the tenant's employees (GET /users?deletedAt=null).
func (c *Client) Users(ctx context.Context, tenantID, bearer string) ([]Profile, error) {
	raw, err := c.get(ctx, "/users", url.Values{"deletedAt": {"null"}}, tenantID, bearer)
	if err != nil {
		return nil, err
	}
	list := unwrapList(raw)
	out := make([]Profile, 0, len(list))
	for _, rec := range list {
		p := mapProfile(rec)
		if p.ID == "" {
			continue
		}
		if p.TenantID == "" {
			p.TenantID = tenantID
		}
		out = append(out, p)
	}
	return out, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, tenantID, bearer string) (any, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if bearer != "" {
		if !strings.HasPrefix(strings.ToLower(bearer), "bearer ") {
			bearer = "Bearer " + bearer
		}
		req.Header.Set("Authorization", bearer)
	}
	if tenantID != "" {
		req.Header.Set("tenantid", tenantID)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("org&emp %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("org&emp %s: HTTP %d: %s", path, resp.StatusCode, truncate(string(body), 300))
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("org&emp %s: decode: %w", path, err)
	}
	return c.maybeDecrypt(payload), nil
}

// maybeDecrypt replaces an encrypted top-level `data` string with its JSON value.
func (c *Client) maybeDecrypt(payload any) any {
	top, ok := payload.(map[string]any)
	if !ok || c.crypto == nil {
		return payload
	}
	s, ok := top["data"].(string)
	if !ok || !LooksEncrypted(s) {
		return payload
	}
	plain, err := c.crypto.Decrypt(s)
	if err != nil {
		return payload
	}
	var v any
	if json.Unmarshal([]byte(plain), &v) == nil {
		top["data"] = v
	}
	return top
}

// --- payload shape helpers (Org&Emp wraps responses inconsistently) ---

func unwrapRecord(v any) map[string]any {
	for depth := 0; depth < 4; depth++ {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		if _, hasID := m["id"]; hasID {
			if _, hasEmail := m["email"]; hasEmail {
				return m
			}
		}
		next, ok := firstOf(m, "data", "user", "success")
		if !ok {
			return m
		}
		v = next
	}
	return nil
}

func unwrapList(v any) []map[string]any {
	for depth := 0; depth < 4; depth++ {
		switch t := v.(type) {
		case []any:
			out := make([]map[string]any, 0, len(t))
			for _, e := range t {
				if m, ok := e.(map[string]any); ok {
					out = append(out, m)
				}
			}
			return out
		case map[string]any:
			next, ok := firstOf(t, "data", "items", "users")
			if !ok {
				return nil
			}
			v = next
		default:
			return nil
		}
	}
	return nil
}

func firstOf(m map[string]any, keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v, true
		}
	}
	return nil, false
}

func str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func obj(m map[string]any, key string) map[string]any {
	o, _ := m[key].(map[string]any)
	return o
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return strings.EqualFold(t, "true") || t == "1"
	}
	return false
}

func mapProfile(m map[string]any) Profile {
	p := Profile{
		ID:          str(m, "id", "userId"),
		FirebaseUID: str(m, "firebaseId", "uid", "firebaseUid"),
		Email:       strings.ToLower(str(m, "email")),
		AvatarURL:   str(m, "profileImage", "avatarUrl"),
		TenantID:    str(m, "tenantId"),
		Deleted:     m["deletedAt"] != nil,
	}
	names := []string{}
	for _, k := range []string{"firstName", "middleName", "lastName"} {
		if s := str(m, k); s != "" {
			names = append(names, s)
		}
	}
	p.FullName = strings.Join(names, " ")
	if p.FullName == "" {
		p.FullName = str(m, "fullName", "name")
	}
	if p.FullName == "" && p.Email != "" {
		p.FullName = strings.Split(p.Email, "@")[0]
	}
	if t := obj(m, "tenant"); t != nil {
		if p.TenantID == "" {
			p.TenantID = str(t, "id", "tenantId")
		}
		p.TenantName = str(t, "name", "companyName", "tenantName")
	}
	if r := obj(m, "reportingTo"); r != nil {
		p.ManagerOrgUserID = str(r, "id", "userId")
	}
	if p.ManagerOrgUserID == "" {
		p.ManagerOrgUserID = str(m, "reportingToId", "managerId")
	}
	if p.TenantName == "" {
		p.TenantName = str(m, "tenantName", "companyName")
	}
	if r := obj(m, "role"); r != nil {
		p.RoleSlug = str(r, "slug", "name")
	} else {
		p.RoleSlug = str(m, "roleSlug", "role")
	}

	// Active job: first with isPositionActive != false, else the first job.
	if jobs, ok := m["employeeJobInformation"].([]any); ok && len(jobs) > 0 {
		var job map[string]any
		for _, j := range jobs {
			if jm, ok := j.(map[string]any); ok {
				if active, set := jm["isPositionActive"]; !set || truthy(active) {
					job = jm
					break
				}
			}
		}
		if job == nil {
			job, _ = jobs[0].(map[string]any)
		}
		if job != nil {
			dept := obj(job, "department")
			p.DepartmentID = str(job, "departmentId")
			if dept != nil {
				if p.DepartmentID == "" {
					p.DepartmentID = str(dept, "id")
				}
				p.DepartmentName = str(dept, "name")
			}
			if pos := obj(job, "position"); pos != nil {
				p.JobTitle = str(pos, "name")
			}
			if p.JobTitle == "" {
				p.JobTitle = str(job, "jobTitle", "positionName")
			}
			p.IsDepartmentLead = truthy(job["departmentLeadOrNot"])
		}
	}
	return p
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
