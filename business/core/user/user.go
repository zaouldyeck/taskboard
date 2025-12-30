package user

import (
	"errors"
	"regexp"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmptyUsername      = errors.New("username cannot be empty")
)

type User struct {
	id       string
	email    string
	username string
	hash     string
}

func New(email, username, password string) (User, error) {
	// Validate email format.
	if !isValidEmail(email) {
		return User{}, ErrInvalidEmail
	}

	// username cannot be nil.
	if username == "" {
		return User{}, ErrEmptyUsername
	}

	// Validate password strength.
	if len(password) < 8 {
		return User{}, ErrWeakPassword
	}

	// Hash password. bcrypt.DefaultCost = 10 (balancing security and speed).
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	// Generate unique ID.
	id := uuid.New().String()

	return User{
		id:       id,
		email:    email,
		username: username,
		hash:     string(hash),
	}, nil
}

func isValidEmail(email string) bool {
	// Format: user@name.domain
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// Validates that the provided password is correct.
func (u User) Authenticate(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.hash), []byte(password))
	return err == nil
}

// Getter functions for uid, email and username below.

func (u User) ID() string {
	return u.id
}

func (u User) Email() string {
	return u.email
}

func (u User) Username() string {
	return u.username
}

// Unmarshal reconstructs a User from DB data.
// This function used only by the db layer.
func Unmarshal(id, email, username, hash string) (User, error) {
	// Sanity checks.
	if id == "" || email == "" || username == "" || hash == "" {
		return User{}, errors.New("invalid user data from database")
	}

	return User{
		id:       id,
		email:    email,
		username: username,
		hash:     hash,
	}, nil
}

// PasswordHash returns pw hash for db persistance.
func (u User) PasswordHash() string {
	return u.hash
}
