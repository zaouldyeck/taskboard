package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zaouldyeck/taskboard/sys/api/gateway/grpcclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Store details about a new user.
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserHandler struct {
	userClient *grpcclient.UserClient
}

// extractToken is a helper which gets JWT from http request
// for usage with token validation.
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	// Extract token string from Authorization Header,
	// as we are not interested in "Bearer" in the string, only the token itself.
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

// Instantiate a user handler with depency injection.
func NewUserHandler(userClient *grpcclient.UserClient) *UserHandler {
	return &UserHandler{userClient: userClient}
}

// Register new user via user-service.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate that the required fields have been passed in from the client.
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Register user info with user-service so that it can be stored in the DB
	// and be used for login.
	authResp, err := h.userClient.Register(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		// Extract gRPC status code from err and send equivalent http error response back to http client.
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.AlreadyExists:
				http.Error(w, "Email already registered", http.StatusConflict)
				return
			case codes.InvalidArgument:
				http.Error(w, st.Message(), http.StatusBadRequest)
				return
			}
		}
		// Default to 500 HTTP response for anything else unknown.
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(authResp)
}

// Login user via user-service.
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Validate that the required fields have been passed in from the client.
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}
	// Login user  with user-service
	authResp, err := h.userClient.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		// Extract gRPC status code from err and send equivalent http error response back to http client.
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound, codes.Unauthenticated:
				http.Error(w, "Invalid email or password", http.StatusUnauthorized)
				return
			}
		}
		// Default to 500 HTTP response for anything else unknown.
		http.Error(w, "Login failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(authResp)
}

// ValidateToken checks JWT for validity in order to handle auth for users.
func (h *UserHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}
	tokenValidation, err := h.userClient.ValidateToken(r.Context(), token)
	if err != nil {
		// Extract gRPC status code from err and send equivalent http error response back to http client.
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.Unauthenticated:
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
		}
		// Default to 500 HTTP response for anything else unknown.
		http.Error(w, "Token validation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tokenValidation)
}

// GetCurrentUser is used to display the logged-in user's profile.
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	tokenValidation, err := h.userClient.ValidateToken(r.Context(), token)
	if err != nil {
		// Extract gRPC status code from err and send equivalent http error response back to http client.
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.Unauthenticated:
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
		}
		// Default to 500 HTTP response for anything else unknown.
		http.Error(w, "Token validation failed", http.StatusInternalServerError)
		return
	}

	if !tokenValidation.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	user, err := h.userClient.GetUser(r.Context(), tokenValidation.UserID)
	if err != nil {
		// Extract gRPC status code from err and send equivalent http error response back to http client.
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.NotFound:
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
		}
		// Default to 500 HTTP response for anything else unknown.
		http.Error(w, "User profile unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
