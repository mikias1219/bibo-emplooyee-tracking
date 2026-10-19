// Package orgauth signs users in with the shared Selamnew identity (Firebase +
// Org&Emp) and keeps each tenant's local roster in sync with the Org&Emp
// directory. Owners (Core owner/admin roles) manage the tenant's workspace;
// everyone else in the directory is an employee.
package orgauth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"ctracking/backend/internal/identity"
	"ctracking/backend/internal/obs"
	"ctracking/backend/internal/orgemp"
	"ctracking/backend/internal/store"
)

var (
	// ErrNotInDirectory: Firebase accepted the user but Org&Emp has no profile.
	ErrNotInDirectory = errors.New("account not found in the organization directory")
	// ErrNoTenant: the Org&Emp profile has no tenant assigned.
	ErrNoTenant = errors.New("account has no organization assigned")
	// ErrNoSyncCredential: no owner session/service token to call Org&Emp with.
	ErrNoSyncCredential = errors.New("no Org&Emp credential for this organization; an owner must sign in again")
)

// minSyncGap throttles directory refreshes per tenant.
const minSyncGap = time.Minute

// Service wires Firebase, Org&Emp and the local store.
type Service struct {
	fb           *identity.Firebase
	org          *orgemp.Client
	st           *store.Store
	ownerRoles   map[string]bool
	ownerEmails  map[string]bool
	serviceToken string

	mu       sync.Mutex
	creds    map[string]string    // tenantID → latest owner's Firebase refresh token (memory only)
	lastSync map[string]time.Time // tenantID → last directory sync
}

// New builds the service. ownerRoles are Core role slugs/names (case-insensitive);
// ownerEmails force specific accounts to owner regardless of role.
func New(fb *identity.Firebase, org *orgemp.Client, st *store.Store, ownerRoles, ownerEmails []string, serviceToken string) *Service {
	return &Service{
		fb: fb, org: org, st: st,
		ownerRoles:   lowerSet(ownerRoles),
		ownerEmails:  lowerSet(ownerEmails),
		serviceToken: serviceToken,
		creds:        map[string]string{},
		lastSync:     map[string]time.Time{},
	}
}

// SignInPassword verifies email/password with Firebase and provisions the user.
func (s *Service) SignInPassword(ctx context.Context, email, password string) (store.User, store.Business, error) {
	sess, err := s.fb.SignIn(ctx, email, password)
	if err != nil {
		return store.User{}, store.Business{}, err
	}
	return s.provision(ctx, sess)
}

// SignInIDToken accepts a Firebase ID token from another Selamnew app (SSO on the
// shared origin) and provisions the user.
func (s *Service) SignInIDToken(ctx context.Context, idToken string) (store.User, store.Business, error) {
	sess, err := s.fb.VerifyIDToken(ctx, idToken)
	if err != nil {
		return store.User{}, store.Business{}, err
	}
	return s.provision(ctx, sess)
}

func (s *Service) provision(ctx context.Context, sess identity.Session) (store.User, store.Business, error) {
	p, err := s.org.UserByFirebaseID(ctx, sess.UID, sess.IDToken)
	if errors.Is(err, orgemp.ErrNotFound) {
		return store.User{}, store.Business{}, ErrNotInDirectory
	}
	if err != nil {
		return store.User{}, store.Business{}, err
	}
	if p.Deleted {
		return store.User{}, store.Business{}, store.ErrDeactivated
	}
	if p.TenantID == "" {
		return store.User{}, store.Business{}, ErrNoTenant
	}
	if p.Email == "" {
		p.Email = sess.Email
	}
	isOwner := s.isOwner(p)
	u, biz, err := s.st.ProvisionOrgUser(ctx, store.OrgMember{Profile: p, Role: s.roleOf(p)})
	if err != nil {
		return store.User{}, store.Business{}, err
	}
	obs.Info("org sign-in", "user", u.ID, "tenant", p.TenantID, "owner", isOwner)

	// An owner's session doubles as the credential for directory syncs, so the
	// roster is fresh by the time the dashboard loads.
	if isOwner {
		if sess.RefreshToken != "" {
			s.mu.Lock()
			s.creds[p.TenantID] = sess.RefreshToken
			s.mu.Unlock()
		}
		go func(tenant, bearer string) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if _, err := s.syncTenant(ctx, tenant, bearer, false); err != nil {
				obs.Warn("directory sync failed", "tenant", tenant, "err", err)
			}
		}(p.TenantID, sess.IDToken)
	}
	return u, biz, nil
}

// SyncBusiness refreshes a business's roster from Org&Emp now (owner-triggered).
func (s *Service) SyncBusiness(ctx context.Context, businessID string) (int, error) {
	tenant, err := s.st.TenantForBusiness(ctx, businessID)
	if err != nil {
		return 0, err
	}
	if tenant == "" {
		return 0, store.ErrNotFound
	}
	bearer, err := s.bearerFor(ctx, tenant)
	if err != nil {
		return 0, err
	}
	return s.syncTenant(ctx, tenant, bearer, true)
}

// StartSweeper refreshes every tenant with a known credential on an interval.
func (s *Service) StartSweeper(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				s.mu.Lock()
				tenants := make([]string, 0, len(s.creds))
				for id := range s.creds {
					tenants = append(tenants, id)
				}
				s.mu.Unlock()
				for _, tenant := range tenants {
					bearer, err := s.bearerFor(ctx, tenant)
					if err == nil {
						_, err = s.syncTenant(ctx, tenant, bearer, false)
					}
					if err != nil {
						obs.Warn("directory sweep failed", "tenant", tenant, "err", err)
					}
				}
			}
		}
	}()
}

func (s *Service) syncTenant(ctx context.Context, tenant, bearer string, force bool) (int, error) {
	s.mu.Lock()
	if !force && time.Since(s.lastSync[tenant]) < minSyncGap {
		s.mu.Unlock()
		return 0, nil
	}
	s.lastSync[tenant] = time.Now()
	s.mu.Unlock()

	profiles, err := s.org.Users(ctx, tenant, bearer)
	if err != nil {
		return 0, err
	}
	// Fail-soft (like Operation's poller): an empty directory never wipes the roster.
	if len(profiles) == 0 {
		return 0, nil
	}
	members := make([]store.OrgMember, 0, len(profiles))
	for _, p := range profiles {
		members = append(members, store.OrgMember{Profile: p, Role: s.roleOf(p)})
	}
	n, err := s.st.SyncDirectory(ctx, tenant, members)
	if err == nil {
		obs.Info("directory synced", "tenant", tenant, "members", n)
	}
	return n, err
}

// bearerFor mints a fresh ID token from the tenant's stored owner refresh token,
// falling back to the configured service token.
func (s *Service) bearerFor(ctx context.Context, tenant string) (string, error) {
	s.mu.Lock()
	rt := s.creds[tenant]
	s.mu.Unlock()
	if rt != "" {
		sess, err := s.fb.Refresh(ctx, rt)
		if err == nil {
			s.mu.Lock()
			s.creds[tenant] = sess.RefreshToken
			s.mu.Unlock()
			return sess.IDToken, nil
		}
		s.mu.Lock()
		delete(s.creds, tenant)
		s.mu.Unlock()
	}
	if s.serviceToken != "" {
		return s.serviceToken, nil
	}
	return "", ErrNoSyncCredential
}

// roleOf maps an Org&Emp profile to a local role (roadmap M9): Core owner/admin
// (or OWNER_EMAILS) → owner; department lead → manager; everyone else → employee.
func (s *Service) roleOf(p orgemp.Profile) string {
	switch {
	case s.isOwner(p):
		return "owner"
	case p.IsDepartmentLead:
		return "manager"
	}
	return "employee"
}

func (s *Service) isOwner(p orgemp.Profile) bool {
	return s.ownerRoles[strings.ToLower(p.RoleSlug)] || s.ownerEmails[strings.ToLower(p.Email)]
}

func lowerSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, v := range list {
		if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
			m[v] = true
		}
	}
	return m
}
