// Package store is the data-access layer over Postgres.
package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a lookup matches no row.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on a unique-constraint violation (e.g. duplicate email).
var ErrConflict = errors.New("conflict")

// Store wraps the connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a Store over the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// User is an application account.
type User struct {
	ID          string
	Email       string // may be empty for members who log in by username
	Username    string // may be empty for owners who log in by email
	DisplayName string
	AccountType string // 'manager' | 'parent' (default 'manager')
}

// GetUserByID returns the user by id (without the password hash).
func (s *Store) GetUserByID(ctx context.Context, id string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT id, COALESCE(email, ''), COALESCE(username, ''), display_name, account_type FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.AccountType)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// IsMember reports whether the user belongs to the business.
func (s *Store) IsMember(ctx context.Context, userID, businessID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id = $1 AND business_id = $2 AND deactivated_at IS NULL)`,
		userID, businessID,
	).Scan(&exists)
	return exists, err
}

// nullableLower lowercases/trims an identifier, returning nil (SQL NULL) when blank
// so unique indexes don't collide on empty strings.
func nullableLower(s string) any {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	return s
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
