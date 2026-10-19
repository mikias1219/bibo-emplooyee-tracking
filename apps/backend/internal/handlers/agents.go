package handlers

import (
	"net/http"
	"strings"

	"ctracking/backend/internal/access"
	"ctracking/backend/internal/auth"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AgentsHandler serves agent check-in and the admin's Agent health view (roadmap M1/M10).
type AgentsHandler struct{ store *store.Store }

// NewAgentsHandler wires the agents handler.
func NewAgentsHandler(s *store.Store) *AgentsHandler { return &AgentsHandler{store: s} }

// Checkin records the calling agent's version/OS/host: POST /v1/agent/checkin
func (h *AgentsHandler) Checkin(c *gin.Context) {
	userID, _ := auth.UserID(c)
	var req store.AgentCheckin
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid body")
		return
	}
	if _, err := uuid.Parse(req.DeviceID); err != nil {
		badRequest(c, "device_id must be a uuid")
		return
	}
	req.OS = strings.ToLower(strings.TrimSpace(req.OS))
	switch req.OS {
	case "", "windows", "macos", "linux":
	default:
		badRequest(c, "os must be windows, macos or linux")
		return
	}
	if err := h.store.RecordAgentCheckin(c.Request.Context(), userID, req); err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// List returns the business's agents: GET /v1/businesses/:id/agents (managers: own department).
func (h *AgentsHandler) List(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "manager")
	if !ok {
		return
	}
	list, err := h.store.ListAgents(c.Request.Context(), scope.BusinessID, scope.DepartmentFilter(c.Query("department_id")))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"agents": list})
}
