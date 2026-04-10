package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/zaouldyeck/taskboard/sys/auth"
)

// Service handles business logic for users.
type Service struct {
	store       *Store
	auth        *auth.Auth
	tokenExpiry time.Duration
}

type Config struct {
	Store       *Store
	Auth        *auth.Auth
	TokenExpiry time.Duration
}

func NewService(cfg Config) *Service {
	return &Service{
		store:       cfg.Store,
		auth:        cfg.Auth,
		tokenExpiry: cfg.TokenExpiry,
	}
}

// Register creates a new user.
func (s *Service) Register(ctx context.Context, email, username, password string) (User, string, error) {
	// Validate input.
	if email == "" {
		return User{}, "", ErrEmptyEmail
	}
	if username == "" {
		return User{}, "", ErrEmptyUsername
	}
	if password == "" {
		return User{}, "", ErrEmptyPassword
	}

	// Create user object.
	usr, err := New(email, username, password)
	if err != nil {
		// Return domain level errors during user creation validation.
		if errors.Is(err, ErrInvalidEmail) {
			return User{}, "", ErrInvalidEmail
		}
		if errors.Is(err, ErrWeakPassword) {
			return User{}, "", ErrWeakPassword
		}
		return User{}, "", fmt.Errorf("unable to create user: %w", err)
	}

	// Save user to DB.
	if err := s.store.Create(ctx, usr); err != nil {
		// Return domain level error during user storing validation.
		if errors.Is(err, ErrEmailTaken) {
			return User{}, "", ErrEmailTaken
		}
		return User{}, "", fmt.Errorf("unable to store user in DB: %w", err)
	}

	// Generate JWT.
	token, err := s.auth.GenerateToken(usr.Id(), usr.Email(), s.tokenExpiry)
	if err != nil {
		return User{}, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return usr, token, nil
}

// Login auths a user and returns a token.
func (s *Service) Login(ctx context.Context, email, password string) (User, string, error) {
	// Validate input.
	if email == "" {
		return User{}, "", ErrEmptyEmail
	}
	if password == "" {
		return User{}, "", ErrEmptyPassword
	}

	// Get user by querying for the email address.
	usr, err := s.store.QueryByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, "", ErrInvalidCredentials
		}
		return User{}, "", fmt.Errorf("querying user by email: %w", err)
	}

	// Verify password.
	if !usr.Authenticate(password) {
		return User{}, "", ErrInvalidCredentials
	}

	// Generate JWT.
	token, err := s.auth.GenerateToken(usr.Id(), usr.Email(), s.tokenExpiry)
	if err != nil {
		return User{}, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Return response.
	return usr, token, nil
}

// ValidateToken checks JWT validity.
func (s *Service) ValidateToken(ctx context.Context, token string) (auth.TokenClaims, bool, error) {
	if token == "" {
		return auth.TokenClaims{}, false, nil
	}

	// Validate token.
	claims, err := s.auth.ValidateToken(token)
	if err != nil {
		return auth.TokenClaims{}, false, nil
	}

	// Validation result.
	return *claims, true, nil
}

// GetUser fetches user info by querying id.
func (s *Service) GetUser(ctx context.Context, userID string) (User, error) {
	if userID == "" {
		return User{}, ErrEmptyID
	}

	// Get user from DB.
	usr, err := s.store.QueryById(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("failed to get user: %w", err)
	}

	// Return user info to caller.
	return usr, nil
}
