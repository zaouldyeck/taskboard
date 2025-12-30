package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Store manages storing users in DB.
type Store struct {
	db *sql.DB
}

// dbUser represents db schema.
// User internally for sql queryrow scanning.
type dbUser struct {
	Id           string `db:"id"`
	Email        string `db:"email"`
	Username     string `db:"username"`
	PasswordHash string `db:"password_hash"`
	CreatedAt    string `db:"created_at"`
	UpdatedAt    string `db:"updated_at"`
}

// toDomain converts dbUser to domain User type.
func (dbu dbUser) toDomain() (User, error) {
	return Unmarshal(dbu.Id, dbu.Email, dbu.Username, dbu.PasswordHash)
}

// NewStore constructs a user store.
func NewStore(db *sql.DB) *Store {
	return &Store{
		db: db,
	}
}

// Create inserts a new user into DB.
func (s *Store) Create(ctx context.Context, user User) error {
	const q = `
		INSERT INTO users (id, email, username, password_hash)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.db.ExecContext(
		ctx,
		q,
		user.ID(),
		user.Email(),
		user.Username(),
		user.PasswordHash(),
	)
	if err != nil {
		// If email is already registered.
		if isUniqueViolation(err) {
			return ErrEmailTaken
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

// isUniqueViolation checks for unique constraint violation.
// Related to postgres error code 23505.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())

	// Common postgres unique violation messages.
	return strings.Contains(errMsg, "unique") || strings.Contains(errMsg, "duplicate")
}

func (s *Store) QueryByEmail(ctx context.Context, email string) (User, error) {
	const q = `
		SELECT id, email, username, password_hash
		FROM users
		WHERE email = $1
	`

	var dbu dbUser

	err := s.db.QueryRowContext(ctx, q, email).Scan(&dbu.Id, &dbu.Email, &dbu.Username, &dbu.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("selecting user by email: %w", err)
	}

	// Convert DB data to domain representation.
	return dbu.toDomain()
}

func (s *Store) QueryById(ctx context.Context, id string) (User, error) {
	const q = `
		SELECT id, email, username, password_hash
		FROM users
		WHERE id = $1
	`

	var dbu dbUser

	err := s.db.QueryRowContext(ctx, q, id).Scan(&dbu.Id, &dbu.Email, &dbu.Username, &dbu.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("selecting user by id: %w", err)
	}

	// Convert DB data to domain representation.
	return dbu.toDomain()
}

func (s *Store) QueryByUsername(ctx context.Context, username string) (User, error) {
	const q = `
		SELECT id, email, username, password_hash
		FROM users
		WHERE username = $1
	`

	var dbu dbUser

	err := s.db.QueryRowContext(ctx, q, username).Scan(&dbu.Id, &dbu.Email, &dbu.Username, &dbu.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("selecting user by username: %w", err)
	}

	// Convert from DB data to domain representation.
	return dbu.toDomain()
}
