package handlers

import (
	"errors"
	"net/http"
	"strings"

	"ctracking/backend/internal/auth"
	"ctracking/backend/internal/identity"
	"ctracking/backend/internal/obs"
	"ctracking/backend/internal/orgauth"
	"ctracking/backend/internal/store"

	"github.com/gin-gonic/gin"
)

// AuthHandler serves organization sign-in, token refresh and /me.
type AuthHandler struct {
	store  *store.Store
	tok    *auth.Manager
	org    *orgauth.Service
	admins []string // SUPER_ADMINS, surfaced as is_super_admin on the user
}

// NewAuthHandler wires the auth handler.
func NewAuthHandler(s *store.Store, tok *auth.Manager, org *orgauth.Service, admins []string) *AuthHandler {
	return &AuthHandler{store: s, tok: tok, org: org, admins: admins}
}

type loginReq struct {
	Identifier string `json:"identifier"` // organization email
	Email      string `json:"email"`      // legacy field (desktop); treated as the identifier
	Password   string `json:"password"`
	BusinessID string `json:"business_id"` // optional: must be one of the user's businesses
}

// Login verifies the organization credentials with the shared Selamnew Firebase
// project, resolves the user (tenant, role, department) from Org&Emp, mirrors them
// locally, and returns this API's own token pair.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid body")
		return
	}
	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.Email)
	}
	if identifier == "" || req.Password == "" {
		badRequest(c, "email and password are required")
		return
	}
	u, _, err := h.org.SignInPassword(c.Request.Context(), identifier, req.Password)
	if !h.signInError(c, err) {
		return
	}
	if req.BusinessID != "" && !h.requireMember(c, u.ID, req.BusinessID) {
		return
	}
	h.issue(c, http.StatusOK, u)
}

type firebaseLoginReq struct {
	IDToken string `json:"id_token"`
}

// FirebaseLogin exchanges a Firebase ID token from another Selamnew app (the
// shared session on the same origin) for this API's token pair.
func (h *AuthHandler) FirebaseLogin(c *gin.Context) {
	var req firebaseLoginReq
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.IDToken) == "" {
		badRequest(c, "id_token is required")
		return
	}
	u, _, err := h.org.SignInIDToken(c.Request.Context(), strings.TrimSpace(req.IDToken))
	if !h.signInError(c, err) {
		return
	}
	h.issue(c, http.StatusOK, u)
}

// signInError writes the response for a failed org sign-in and reports whether
// the caller may continue (err == nil).
func (h *AuthHandler) signInError(c *gin.Context, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, identity.ErrInvalidCredentials), errors.Is(err, identity.ErrInvalidToken):
		unauthorized(c, "invalid credentials")
	case errors.Is(err, orgauth.ErrNotInDirectory), errors.Is(err, orgauth.ErrNoTenant):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error() + ". Contact your administrator."})
	case errors.Is(err, store.ErrDeactivated):
		c.JSON(http.StatusForbidden, gin.H{"error": "your account is no longer active in your organization"})
	case errors.Is(err, store.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "this account conflicts with an existing user; contact support"})
	default:
		obs.Error("org sign-in failed", "err", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "the organization directory is unavailable, try again shortly"})
	}
	return false
}

func (h *AuthHandler) requireMember(c *gin.Context, userID, businessID string) bool {
	member, err := h.store.IsMember(c.Request.Context(), userID, businessID)
	if err != nil {
		serverError(c, err)
		return false
	}
	if !member {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of that business"})
		return false
	}
	return true
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh exchanges a valid refresh token for a new token pair.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "invalid body")
		return
	}
	userID, err := h.tok.ParseRefresh(req.RefreshToken)
	if err != nil {
		unauthorized(c, "invalid refresh token")
		return
	}
	// Users removed from the Org&Emp directory lose access at their next refresh.
	active, err := h.store.MembershipActive(c.Request.Context(), userID)
	if err != nil {
		serverError(c, err)
		return
	}
	if !active {
		unauthorized(c, "account deactivated")
		return
	}
	pair, err := h.tok.Issue(userID)
	if err != nil {
		serverError(c, err)
		return
	}
	obs.Info("login ok", "user", userID)
	c.JSON(http.StatusOK, pair)
}

// Me returns the authenticated user (protected route).
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := auth.UserID(c)
	if !ok {
		unauthorized(c, "unauthenticated")
		return
	}
	u, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":             u.ID,
		"email":          u.Email,
		"username":       u.Username,
		"display_name":   u.DisplayName,
		"account_type":   u.AccountType,
		"is_super_admin": isSuperAdmin(u, h.admins),
	})
}

func (h *AuthHandler) issue(c *gin.Context, status int, u store.User) {
	pair, err := h.tok.Issue(u.ID)
	if err != nil {
		serverError(c, err)
		return
	}
	c.JSON(status, gin.H{
		"user": gin.H{
			"id":             u.ID,
			"email":          u.Email,
			"username":       u.Username,
			"display_name":   u.DisplayName,
			"account_type":   u.AccountType,
			"is_super_admin": isSuperAdmin(u, h.admins),
		},
		"tokens": pair,
	})
}
