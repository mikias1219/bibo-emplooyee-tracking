package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Membership is a user's active role in one business.
type Membership struct {
	Role         string // owner | manager | employee
	DepartmentID string
}

// MembershipOf returns the caller's active membership in a business, or ErrNotFound.
func (s *Store) MembershipOf(ctx context.Context, userID, businessID string) (Membership, error) {
	var m Membership
	err := s.pool.QueryRow(ctx, `
		SELECT role, COALESCE(department_id, '') FROM memberships
		 WHERE user_id = $1 AND business_id = $2::uuid AND deactivated_at IS NULL`,
		userID, businessID).Scan(&m.Role, &m.DepartmentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrNotFound
	}
	return m, err
}
