package handlers

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"ctracking/backend/internal/auth"
	"ctracking/backend/internal/productivity"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// maxReportRange caps a performance/usage window so one request can't scan years.
const maxReportRange = 93 * 86400

// ProductivityHandler serves productivity rules and performance reports.
type ProductivityHandler struct {
	store *store.Store
}

// NewProductivityHandler wires the productivity handler.
func NewProductivityHandler(s *store.Store) *ProductivityHandler {
	return &ProductivityHandler{store: s}
}

// requireOwner checks the caller owns business :id, writing the error otherwise.
func (h *ProductivityHandler) requireOwner(c *gin.Context) (string, bool) {
	userID, _ := auth.UserID(c)
	businessID := c.Param("id")
	owns, err := h.store.IsBusinessOwner(c.Request.Context(), userID, businessID)
	if err != nil {
		serverError(c, err)
		return "", false
	}
	if !owns {
		c.JSON(http.StatusForbidden, gin.H{"error": "not your business"})
		return "", false
	}
	return businessID, true
}

// window parses from/to (unix seconds), defaulting to the last 7 days.
func window(c *gin.Context) (from, to int64, ok bool) {
	now := time.Now().Unix()
	to = parseInt64(c.Query("to"), now+1)
	from = parseInt64(c.Query("from"), to-7*86400)
	if from >= to {
		badRequest(c, "from must be before to")
		return 0, 0, false
	}
	if to-from > maxReportRange {
		badRequest(c, "range must be 93 days or less")
		return 0, 0, false
	}
	return from, to, true
}

// ListRules returns the business's productivity rules.
func (h *ProductivityHandler) ListRules(c *gin.Context) {
	businessID, ok := h.requireOwner(c)
	if !ok {
		return
	}
	rules, err := h.store.ListProductivityRules(c.Request.Context(), businessID)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

// SetRule creates or updates the rule for an app or domain. Body:
// {match_type: "app"|"domain", pattern, category}. category "unclassified"
// removes the rule.
func (h *ProductivityHandler) SetRule(c *gin.Context) {
	businessID, ok := h.requireOwner(c)
	if !ok {
		return
	}
	var r productivity.Rule
	if err := c.ShouldBindJSON(&r); err != nil {
		badRequest(c, "invalid body")
		return
	}
	switch r.MatchType {
	case productivity.MatchApp:
		r.Pattern = productivity.NormalizeApp(r.Pattern)
	case productivity.MatchDomain:
		r.Pattern = productivity.NormalizeDomain(r.Pattern)
	default:
		badRequest(c, "match_type must be 'app' or 'domain'")
		return
	}
	if r.Pattern == "" || len(r.Pattern) > 255 {
		badRequest(c, "pattern must be a valid app name or domain up to 255 characters")
		return
	}

	if r.Category == productivity.Unclassified {
		if err := h.deleteByPattern(c, businessID, r); err != nil {
			serverError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"rule": nil})
		return
	}
	if !productivity.ValidRuleCategory(r.Category) {
		badRequest(c, "category must be productive, unproductive, neutral or unclassified")
		return
	}
	saved, err := h.store.UpsertProductivityRule(c.Request.Context(), businessID, r)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": saved})
}

func (h *ProductivityHandler) deleteByPattern(c *gin.Context, businessID string, r productivity.Rule) error {
	rules, err := h.store.ListProductivityRules(c.Request.Context(), businessID)
	if err != nil {
		return err
	}
	for _, x := range rules {
		if x.MatchType == r.MatchType && x.Pattern == r.Pattern {
			err := h.store.DeleteProductivityRule(c.Request.Context(), businessID, x.ID)
			if errors.Is(err, store.ErrNotFound) {
				return nil
			}
			return err
		}
	}
	return nil
}

// DeleteRule removes one rule by id.
func (h *ProductivityHandler) DeleteRule(c *gin.Context) {
	businessID, ok := h.requireOwner(c)
	if !ok {
		return
	}
	err := h.store.DeleteProductivityRule(c.Request.Context(), businessID, c.Param("ruleId"))
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// Usage lists the apps and domains used in a period with their categories.
func (h *ProductivityHandler) Usage(c *gin.Context) {
	businessID, ok := h.requireOwner(c)
	if !ok {
		return
	}
	from, to, ok := window(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	rules, err := h.store.ListProductivityRules(ctx, businessID)
	if err != nil {
		serverError(c, err)
		return
	}
	apps, domains, err := h.store.ProductivityUsage(ctx, businessID, from, to, productivity.NewMatcher(rules))
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"apps": apps, "domains": domains})
}

// perfFigures are the derived percentages attached to every totals row.
type perfFigures struct {
	productivity.Totals
	DaysActive  int      `json:"days_active"`
	ScorePct    *float64 `json:"score_pct"`
	CoveragePct *float64 `json:"coverage_pct"`
	TargetPct   *float64 `json:"target_pct"`
}

func figures(t productivity.Totals, days int, targetS int64) perfFigures {
	return perfFigures{
		Totals:      t,
		DaysActive:  days,
		ScorePct:    t.ScorePct(),
		CoveragePct: t.CoveragePct(),
		TargetPct:   t.TargetPct(targetS, days),
	}
}

type perfEmployee struct {
	store.Member
	perfFigures
}

type perfDay struct {
	Day string `json:"day"`
	perfFigures
}

// Performance returns per-employee and per-day productivity for a business.
// Query: from, to (unix seconds), tz (IANA zone for day buckets, default UTC),
// employee_id (optional, one member only).
func (h *ProductivityHandler) Performance(c *gin.Context) {
	businessID, ok := h.requireOwner(c)
	if !ok {
		return
	}
	from, to, ok := window(c)
	if !ok {
		return
	}
	tz := c.DefaultQuery("tz", "UTC")
	if tz == "" || tz == "Local" {
		tz = "UTC"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		badRequest(c, "tz must be an IANA time zone such as Africa/Addis_Ababa")
		return
	}
	employeeID := c.Query("employee_id")

	ctx := c.Request.Context()
	targetS, err := h.store.ProductiveTargetS(ctx, businessID)
	if err != nil {
		serverError(c, err)
		return
	}
	rules, err := h.store.ListProductivityRules(ctx, businessID)
	if err != nil {
		serverError(c, err)
		return
	}
	members, err := h.store.ListMembers(ctx, businessID)
	if err != nil {
		serverError(c, err)
		return
	}
	if employeeID != "" {
		filtered := members[:0]
		for _, m := range members {
			if m.ID == employeeID {
				filtered = append(filtered, m)
			}
		}
		if len(filtered) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "not your employee"})
			return
		}
		members = filtered
	}
	usage, err := h.store.PerformanceUsage(ctx, businessID, from, to, tz, employeeID)
	if err != nil {
		serverError(c, err)
		return
	}

	matcher := productivity.NewMatcher(rules)
	byUser := map[string]*productivity.Totals{}
	userDays := map[string]int{}
	byDay := map[string]*productivity.Totals{}
	dayPeople := map[string]int{}
	var team productivity.Totals
	teamDays := 0
	for _, du := range usage {
		t := productivity.Classify(matcher, du.Apps, du.Domains)
		if t.ActiveS == 0 {
			continue
		}
		if byUser[du.UserID] == nil {
			byUser[du.UserID] = &productivity.Totals{}
		}
		byUser[du.UserID].Add(t)
		userDays[du.UserID]++
		if byDay[du.Day] == nil {
			byDay[du.Day] = &productivity.Totals{}
		}
		byDay[du.Day].Add(t)
		dayPeople[du.Day]++
		team.Add(t)
		teamDays++
	}

	employees := make([]perfEmployee, 0, len(members))
	for _, m := range members {
		var t productivity.Totals
		if p := byUser[m.ID]; p != nil {
			t = *p
		}
		employees = append(employees, perfEmployee{Member: m, perfFigures: figures(t, userDays[m.ID], targetS)})
	}
	// Most productive time first; members with no activity sink to the bottom.
	sort.SliceStable(employees, func(i, j int) bool {
		return employees[i].ProductiveS > employees[j].ProductiveS
	})

	days := make([]perfDay, 0, len(byDay))
	for d, t := range byDay {
		days = append(days, perfDay{Day: d, perfFigures: figures(*t, dayPeople[d], targetS)})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Day < days[j].Day })

	c.JSON(http.StatusOK, gin.H{
		"target_per_day_s": targetS,
		"team":             figures(team, teamDays, targetS),
		"employees":        employees,
		"days":             days,
	})
}
