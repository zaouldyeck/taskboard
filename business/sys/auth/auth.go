package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Auth is used for JWT generation and validation.
type Auth struct {
	secret []byte
}

// TokenClaims is data payload stored in the JWT.
type TokenClaims struct {
	UserId int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

func NewAuth(secret string) *Auth {
	return &Auth{
		secret: []byte(secret),
	}
}

// GenerateToken creates new JWT for user.
func (a *Auth) GenerateToken(userId int64, email string, duration time.Duration) (string, error) {
	// Create claims with data payload.
	now := time.Now()
	claims := &TokenClaims{
		UserId: userId,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create unsigned token with claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret.
	tokenString, err := token.SignedString(a.secret)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken verifies a JWT and returns claims.
func (a *Auth) ValidateToken(tokenString string) (*TokenClaims, error) {
	// Parse and validate JWT.
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (any, error) {
		// Verify signing method.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	// Extract and validate claims.
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
