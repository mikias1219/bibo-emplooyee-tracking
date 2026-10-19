// Package identity talks to the shared Selamnew Firebase project: it verifies
// email/password credentials (Identity Toolkit REST), refreshes ID tokens, and
// verifies ID tokens minted by other Selamnew apps (SSO exchange).
package identity

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidCredentials is returned for a wrong email/password or disabled account.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidToken is returned for a malformed, expired or foreign ID token.
var ErrInvalidToken = errors.New("invalid firebase token")

const (
	signInURL  = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword"
	refreshURL = "https://securetoken.googleapis.com/v1/token"
	certsURL   = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"
)

// Session is a Firebase sign-in result.
type Session struct {
	UID          string
	Email        string
	IDToken      string // short-lived (1h) bearer accepted by Org&Emp
	RefreshToken string // long-lived; mints new ID tokens
}

// Firebase is a client for one Firebase project.
type Firebase struct {
	apiKey    string
	projectID string
	http      *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	keysUntil time.Time
}

// NewFirebase builds a client for the given web API key + project id.
func NewFirebase(apiKey, projectID string) *Firebase {
	return &Firebase{apiKey: apiKey, projectID: projectID, http: &http.Client{Timeout: 15 * time.Second}}
}

// SignIn verifies email/password against Firebase.
func (f *Firebase) SignIn(ctx context.Context, email, password string) (Session, error) {
	body, _ := json.Marshal(map[string]any{
		"email": strings.TrimSpace(email), "password": password, "returnSecureToken": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signInURL+"?key="+url.QueryEscape(f.apiKey), bytes.NewReader(body))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out struct {
		LocalID      string `json:"localId"`
		Email        string `json:"email"`
		IDToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
		Error        *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	status, err := f.doJSON(req, &out)
	if err != nil {
		return Session{}, err
	}
	if status != http.StatusOK || out.IDToken == "" {
		msg := ""
		if out.Error != nil {
			msg = out.Error.Message
		}
		if isCredentialError(msg) {
			return Session{}, ErrInvalidCredentials
		}
		// e.g. "API key not valid" — a server misconfiguration, not the user's fault.
		return Session{}, fmt.Errorf("firebase sign-in: HTTP %d %s", status, msg)
	}
	return Session{UID: out.LocalID, Email: strings.ToLower(out.Email), IDToken: out.IDToken, RefreshToken: out.RefreshToken}, nil
}

// isCredentialError reports whether a sign-in error code is about the user's
// credentials (codes may carry a suffix, e.g. "TOO_MANY_ATTEMPTS_TRY_LATER : …").
func isCredentialError(msg string) bool {
	code, _, _ := strings.Cut(msg, " ")
	switch code {
	case "EMAIL_NOT_FOUND", "INVALID_PASSWORD", "INVALID_LOGIN_CREDENTIALS", "USER_DISABLED",
		"INVALID_EMAIL", "MISSING_PASSWORD", "MISSING_EMAIL", "TOO_MANY_ATTEMPTS_TRY_LATER":
		return true
	}
	return false
}

// Refresh exchanges a Firebase refresh token for a fresh ID token.
func (f *Firebase) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, refreshURL+"?key="+url.QueryEscape(f.apiKey), strings.NewReader(form.Encode()))
	if err != nil {
		return Session{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var out struct {
		UserID       string `json:"user_id"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
	}
	status, err := f.doJSON(req, &out)
	if err != nil {
		return Session{}, err
	}
	if status != http.StatusOK || out.IDToken == "" {
		return Session{}, ErrInvalidToken
	}
	return Session{UID: out.UserID, IDToken: out.IDToken, RefreshToken: out.RefreshToken}, nil
}

// VerifyIDToken validates an ID token issued for this project and returns the
// session it represents (no refresh token).
func (f *Firebase) VerifyIDToken(ctx context.Context, idToken string) (Session, error) {
	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrInvalidToken
		}
		kid, _ := t.Header["kid"].(string)
		return f.publicKey(ctx, kid)
	},
		jwt.WithAudience(f.projectID),
		jwt.WithIssuer("https://securetoken.google.com/"+f.projectID),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return Session{}, ErrInvalidToken
	}
	uid, _ := claims["sub"].(string)
	if uid == "" {
		return Session{}, ErrInvalidToken
	}
	email, _ := claims["email"].(string)
	return Session{UID: uid, Email: strings.ToLower(email), IDToken: idToken}, nil
}

func (f *Firebase) publicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.keys == nil || time.Now().After(f.keysUntil) || f.keys[kid] == nil {
		if err := f.loadKeysLocked(ctx); err != nil {
			return nil, err
		}
	}
	if k := f.keys[kid]; k != nil {
		return k, nil
	}
	return nil, ErrInvalidToken
}

func (f *Firebase) loadKeysLocked(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certsURL, nil)
	if err != nil {
		return err
	}
	resp, err := f.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var certs map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(certs))
	for kid, certPEM := range certs {
		block, _ := pem.Decode([]byte(certPEM))
		if block == nil {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		if k, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			keys[kid] = k
		}
	}
	f.keys = keys
	f.keysUntil = time.Now().Add(cacheMaxAge(resp.Header.Get("Cache-Control")))
	return nil
}

func cacheMaxAge(cc string) time.Duration {
	for _, part := range strings.Split(cc, ",") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(part), "max-age="); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				return time.Duration(n) * time.Second
			}
		}
	}
	return time.Hour
}

func (f *Firebase) doJSON(req *http.Request, out any) (int, error) {
	resp, err := f.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && resp.StatusCode == http.StatusOK {
		return resp.StatusCode, err
	}
	return resp.StatusCode, nil
}
