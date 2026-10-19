package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"ctracking/backend/internal/auth"
	"ctracking/backend/internal/obs"
	"ctracking/backend/internal/orgauth"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// OwnerHandler serves business settings + the directory-backed roster for owners.
type OwnerHandler struct {
	store *store.Store
	org   *orgauth.Service
}

// NewOwnerHandler wires the owner handler.
func NewOwnerHandler(s *store.Store, org *orgauth.Service) *OwnerHandler {
	return &OwnerHandler{store: s, org: org}
}

// ListMine returns the businesses the caller owns.
func (h *OwnerHandler) ListMine(c *gin.Context) {
	userID, _ := auth.UserID(c)
	list, err := h.store.ListBusinessesOwnedBy(c.Request.Context(), userID)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"businesses": list})
}

// Departments lists the Org&Emp departments present in the business's roster.
func (h *OwnerHandler) Departments(c *gin.Context) {
	if !h.requireOwner(c) {
		return
	}
	list, err := h.store.ListDepartments(c.Request.Context(), c.Param("id"))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"departments": list})
}

// SyncDirectory refreshes the roster from Org&Emp right away.
func (h *OwnerHandler) SyncDirectory(c *gin.Context) {
	if !h.requireOwner(c) {
		return
	}
	n, err := h.org.SyncBusiness(c.Request.Context(), c.Param("id"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusConflict, gin.H{"error": "this workspace is not linked to an organization"})
		return
	case errors.Is(err, orgauth.ErrNoSyncCredential):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	case err != nil:
		obs.Error("directory sync failed", "business", c.Param("id"), "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "the organization directory is unavailable, try again shortly"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "members": n})
}

// ListEmployees returns the roster of a business the caller owns.
func (h *OwnerHandler) ListEmployees(c *gin.Context) {
	if !h.requireOwner(c) {
		return
	}
	list, err := h.store.ListEmployees(c.Request.Context(), c.Param("id"))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"employees": list})
}

// UpdateSettings updates a business's capture policy. Only the keys present in the
// body are changed; screenshot_retention_days accepts null ("keep forever").
func (h *OwnerHandler) UpdateSettings(c *gin.Context) {
	if !h.requireOwner(c) {
		return
	}
	var body map[string]json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		badRequest(c, "invalid body")
		return
	}

	fields := map[string]any{}

	// Retention is nullable: present-as-null means "keep forever".
	if raw, ok := body["screenshot_retention_days"]; ok {
		if string(raw) == "null" {
			fields["screenshot_retention_days"] = nil
		} else {
			var n int
			if json.Unmarshal(raw, &n) != nil || n < 0 {
				badRequest(c, "screenshot_retention_days must be a non-negative integer or null")
				return
			}
			fields["screenshot_retention_days"] = n
		}
	}
	for _, key := range []string{"screenshot_interval_s", "idle_threshold_s"} {
		if raw, ok := body[key]; ok {
			var n int
			if json.Unmarshal(raw, &n) != nil || n <= 0 {
				badRequest(c, key+" must be a positive integer")
				return
			}
			fields[key] = n
		}
	}
	if raw, ok := body["productive_target_s"]; ok {
		var n int
		if json.Unmarshal(raw, &n) != nil || n < 3600 || n > 86400 {
			badRequest(c, "productive_target_s must be between 3600 and 86400 seconds")
			return
		}
		fields["productive_target_s"] = n
	}
	if raw, ok := body["allow_employee_override"]; ok {
		var b bool
		if json.Unmarshal(raw, &b) != nil {
			badRequest(c, "allow_employee_override must be a boolean")
			return
		}
		fields["allow_employee_override"] = b
	}
	if raw, ok := body["screenshot_mode"]; ok {
		var m string
		if json.Unmarshal(raw, &m) != nil || (m != "privacy" && m != "normal") {
			badRequest(c, "screenshot_mode must be 'privacy' or 'normal'")
			return
		}
		fields["screenshot_mode"] = m
	}
	if raw, ok := body["screenshot_skip_apps"]; ok {
		var apps []string
		if json.Unmarshal(raw, &apps) != nil || len(apps) > 100 {
			badRequest(c, "screenshot_skip_apps must be an array of up to 100 app names")
			return
		}
		cleaned := make([]string, 0, len(apps))
		for _, a := range apps {
			a = strings.TrimSpace(a)
			if a == "" || len(a) > 200 {
				badRequest(c, "screenshot_skip_apps entries must be non-empty names up to 200 characters")
				return
			}
			cleaned = append(cleaned, a)
		}
		fields["screenshot_skip_apps"] = cleaned
	}

	if err := h.store.UpdateBusinessSettings(c.Request.Context(), c.Param("id"), fields); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Policy returns the capture policy for the authenticated user's business, or
// {managed:false} when the user has no single business (standalone → local defaults).
func (h *OwnerHandler) Policy(c *gin.Context) {
	userID, _ := auth.UserID(c)
	p, err := h.store.PolicyForUser(c.Request.Context(), userID)
	if errors.Is(err, store.ErrAmbiguousBusiness) {
		c.JSON(http.StatusOK, gin.H{"managed": false})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	if p == nil {
		c.JSON(http.StatusOK, gin.H{"managed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"managed":                   true,
		"screenshot_interval_s":     p.ScreenshotIntervalS,
		"idle_threshold_s":          p.IdleThresholdS,
		"screenshot_retention_days": p.ScreenshotRetentionDays,
		"allow_employee_override":   p.AllowEmployeeOverride,
		"kind":                      p.Kind,
		"screenshot_mode":           p.ScreenshotMode,
		"screenshot_skip_apps":      p.ScreenshotSkipApps,
	})
}

// requireOwner verifies the authenticated caller owns the :id business. It writes
// the appropriate error response and returns false when not allowed.
func (h *OwnerHandler) requireOwner(c *gin.Context) bool {
	userID, _ := auth.UserID(c)
	if _, err := h.store.GetBusiness(c.Request.Context(), c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "business not found"})
		return false
	} else if err != nil {
		serverError(c, err)
		return false
	}
	owns, err := h.store.IsBusinessOwner(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		serverError(c, err)
		return false
	}
	if !owns {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your business"})
		return false
	}
	return true
}
