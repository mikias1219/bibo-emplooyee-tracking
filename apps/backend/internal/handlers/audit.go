package handlers

import (
	"net/http"

	"ctracking/backend/internal/access"
	"ctracking/backend/internal/obs"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// Audit actions (roadmap M9). Keep values stable: they are stored and filtered on.
const (
	auditReportView     = "report.view"
	auditScreenshotView = "screenshot.view"
	auditSettingsUpdate = "settings.update"
	auditPolicyUpdate   = "policy.update"
)

// recordAudit writes an audit entry without failing the request (best effort).
func recordAudit(c *gin.Context, st *store.Store, businessID, actorID, action, targetType, targetID string, meta any) {
	if err := st.RecordAudit(c.Request.Context(), businessID, actorID, action, targetType, targetID, meta); err != nil {
		obs.Warn("audit write failed", "action", action, "err", err)
	}
}

// AuditHandler serves the audit log (owners only).
type AuditHandler struct{ store *store.Store }

// NewAuditHandler wires the audit handler.
func NewAuditHandler(s *store.Store) *AuditHandler { return &AuditHandler{store: s} }

// List returns recent audit entries: GET /v1/businesses/:id/audit?limit=&offset=
func (h *AuditHandler) List(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "owner")
	if !ok {
		return
	}
	limit := clampInt(c.Query("limit"), 100, 1, 500)
	offset := clampInt(c.Query("offset"), 0, 0, 1<<31)
	list, err := h.store.ListAudit(c.Request.Context(), scope.BusinessID, limit, offset)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": list, "limit": limit, "offset": offset})
}
