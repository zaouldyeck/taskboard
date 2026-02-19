package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zaouldyeck/taskboard/internal/gateway/grpcclient"
)

// Custom type used for defining context keys to avoid collision
// in another package.
type contextKey string

const (
	UserIDKey contextKey = "userID"
	EmailKey  contextKey = "email"
)

// HELPER FUNCTION

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// AuthMiddleware validates JWT so that routes are protected from non-auth users.
func AuthMiddleware(userClient *grpcclient.UserClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				http.Error(w, "Missing authorization token", http.StatusUnauthorized)
				return
			}

			tokenValidation, err := userClient.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, "Validation of authorization token failed", http.StatusUnauthorized)
				return
			}

			if !tokenValidation.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// In order for handler to set things like created_by we pass user info to request context.
			ctx := context.WithValue(r.Context(), UserIDKey, tokenValidation.UserID)
			ctx = context.WithValue(ctx, EmailKey, tokenValidation.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
