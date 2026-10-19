// Package access implements the roadmap's role model (module 9):
//
//	owner    — admin: all data and settings of the organization
//	manager  — own department only (department lead in Org&Emp)
//	employee — no admin access (optionally their own activity)
//
// Handlers resolve a Scope once per request and pass its DepartmentFilter to
// queries, so a manager can never read outside their department.
package access

import (
	"errors"
	"net/http"

	"ctracking/backend/internal/auth"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// Scope is what the caller may see in one business.
type Scope struct {
	UserID     string
	BusinessID string
	Role       string
	// DepartmentID is the manager's department; "" for owners.
	DepartmentID string
}

// IsOwner reports whether the caller administers the whole organization.
func (s Scope) IsOwner() bool { return s.Role == "owner" }

// DepartmentFilter returns the department a query must be limited to: the
// manager's own department, or the requested one for owners ("" = all).
func (s Scope) DepartmentFilter(requested string) string {
	if s.IsOwner() {
		return requested
	}
	return s.DepartmentID
}

// Require resolves the caller's scope for :id and enforces a minimum role
// ("owner" or "manager"). It writes the error response and returns false when
// access is denied.
func Require(c *gin.Context, st *store.Store, minRole string) (Scope, bool) {
	userID, _ := auth.UserID(c)
	bizID := c.Param("id")
	m, err := st.MembershipOf(c.Request.Context(), userID, bizID)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your business"})
		return Scope{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return Scope{}, false
	}
	allowed := m.Role == "owner" || (minRole == "manager" && m.Role == "manager" && m.DepartmentID != "")
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return Scope{}, false
	}
	return Scope{UserID: userID, BusinessID: bizID, Role: m.Role, DepartmentID: m.DepartmentID}, true
}
