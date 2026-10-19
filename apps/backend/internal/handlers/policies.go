package handlers

import (
	"errors"
	"net/http"
	"regexp"

	"ctracking/backend/internal/access"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// PoliciesHandler serves per-department monitoring policies (roadmap M10).
type PoliciesHandler struct{ store *store.Store }

// NewPoliciesHandler wires the policies handler.
func NewPoliciesHandler(s *store.Store) *PoliciesHandler { return &PoliciesHandler{store: s} }

var hhmm = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// List: GET /v1/businesses/:id/department-policies
func (h *PoliciesHandler) List(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "owner")
	if !ok {
		return
	}
	list, err := h.store.ListDepartmentPolicies(c.Request.Context(), scope.BusinessID)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": list})
}

// Get: GET /v1/businesses/:id/department-policies/:dept (404 = inherits the org policy)
func (h *PoliciesHandler) Get(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "manager")
	if !ok {
		return
	}
	dept := c.Param("dept")
	if !scope.IsOwner() && dept != scope.DepartmentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your department"})
		return
	}
	p, err := h.store.GetDepartmentPolicy(c.Request.Context(), scope.BusinessID, dept)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no override; the organization policy applies"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// Put replaces a department override: PUT /v1/businesses/:id/department-policies/:dept
func (h *PoliciesHandler) Put(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "owner")
	if !ok {
		return
	}
	var p store.DepartmentPolicy
	if err := c.ShouldBindJSON(&p); err != nil {
		badRequest(c, "invalid body")
		return
	}
	p.DepartmentID = c.Param("dept")
	for _, t := range []*string{p.WorkStart, p.WorkEnd} {
		if t != nil && !hhmm.MatchString(*t) {
			badRequest(c, "work_start/work_end must be HH:MM")
			return
		}
	}
	for _, d := range p.WorkDays {
		if d < 1 || d > 7 {
			badRequest(c, "work_days must be ISO weekdays 1-7")
			return
		}
	}
	if err := h.store.UpsertDepartmentPolicy(c.Request.Context(), scope.BusinessID, p); err != nil {
		serverError(c, err)
		return
	}
	recordAudit(c, h.store, scope.BusinessID, scope.UserID, auditPolicyUpdate, "department", p.DepartmentID, p)
	c.JSON(http.StatusOK, p)
}

// Delete removes an override: DELETE /v1/businesses/:id/department-policies/:dept
func (h *PoliciesHandler) Delete(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "owner")
	if !ok {
		return
	}
	if err := h.store.DeleteDepartmentPolicy(c.Request.Context(), scope.BusinessID, c.Param("dept")); err != nil {
		serverError(c, err)
		return
	}
	recordAudit(c, h.store, scope.BusinessID, scope.UserID, auditPolicyUpdate, "department", c.Param("dept"), gin.H{"deleted": true})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
