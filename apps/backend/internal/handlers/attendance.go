package handlers

import (
	"net/http"
	"time"

	"ctracking/backend/internal/access"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// AttendanceHandler serves observed attendance (roadmap M6).
type AttendanceHandler struct{ store *store.Store }

// NewAttendanceHandler wires the attendance handler.
func NewAttendanceHandler(s *store.Store) *AttendanceHandler { return &AttendanceHandler{store: s} }

// List returns first/last activity and active time per employee per day:
// GET /v1/businesses/:id/attendance?from=&to=&department_id= (managers: own department).
func (h *AttendanceHandler) List(c *gin.Context) {
	scope, ok := access.Require(c, h.store, "manager")
	if !ok {
		return
	}
	now := time.Now().Unix()
	to := parseInt64(c.Query("to"), now+1)
	from := parseInt64(c.Query("from"), to-7*86400)
	if from >= to || to-from > maxReportRange {
		badRequest(c, "from/to must span at most 93 days")
		return
	}
	days, err := h.store.Attendance(c.Request.Context(), scope.BusinessID, from, to, scope.DepartmentFilter(c.Query("department_id")))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": days})
}
