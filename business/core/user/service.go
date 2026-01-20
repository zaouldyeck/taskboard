package user

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/zaouldyeck/taskboard/api/proto/user/v1"
	"github.com/zaouldyeck/taskboard/business/sys/auth"
)

// Service implements UserServiceServer.
type Service struct {
	pb.UnimplementedUserServiceServer
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

// parseUserId converts string UUID to int64 for JWT claims.
func parseUserId(id string) int64 {
	// For LAB only. Simple hashing of UUID.
	hash := int64(0)
	for _, c := range id {
		hash = hash*31 + int64(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// Register creates a new user.
func (s *Service) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Validate input.
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Create user object.
	usr, err := New(req.Email, req.Username, req.Password)
	if err != nil {
		// Translate domain level errors to gRPC.
		if err == ErrInvalidEmail {
			return nil, status.Error(codes.InvalidArgument, "invalid email format")
		}
		if err == ErrWeakPassword {
			return nil, status.Error(codes.InvalidArgument, "minimum password length is 8")
		}
		return nil, status.Error(codes.Internal,
			fmt.Sprintf("failed to create user: %v", err))
	}

	// Save user to DB.
	if err := s.store.Create(ctx, usr); err != nil {
		// Translate storage errors to gRPC errors.
		if err == ErrEmailTaken {
			return nil, status.Error(codes.AlreadyExists, "email already registered")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to save user: %v", err))
	}

	// Generate JWT.
	token, err := s.auth.GenerateToken(parseUserId(usr.Id()), usr.Email(), s.tokenExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to generate token: %v", err))
	}

	return &pb.RegisterResponse{
		User: &pb.User{
			Id:        usr.Id(),
			Email:     usr.Email(),
			Username:  usr.Username(),
			CreatedAt: time.Now().Unix(),
		},
		Token: token,
	}, nil
}

// Login auths a user and returns a token.
func (s *Service) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Validate input.
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Get user by querying for the email address.
	usr, err := s.store.QueryByEmail(ctx, req.Email)
	if err != nil {
		if err == ErrNotFound {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user: %v", err))
	}

	// Verify password.
	if !usr.Authenticate(req.Password) {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	// Generate JWT.
	token, err := s.auth.GenerateToken(parseUserId(usr.Id()), usr.Email(), s.tokenExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to generate token: %v", err))
	}

	// Return response.
	return &pb.LoginResponse{
		User: &pb.User{
			Id:       usr.Id(),
			Email:    usr.Email(),
			Username: usr.Username(),
		},
		Token: token,
	}, nil
}

// ValidateToken checks JWT validity.
func (s *Service) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	// Validate token.
	claims, err := s.auth.ValidateToken(req.Token)
	if err != nil {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	// Validation result.
	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: fmt.Sprintf("%d", claims.UserId),
		Email:  claims.Email,
	}, nil
}

// GetUser fetches user info by querying id.
func (s *Service) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Get user from DB.
	usr, err := s.store.QueryById(ctx, req.Id)
	if err != nil {
		if err == ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get user: %v", err))
	}

	// Return user info to caller.
	return &pb.GetUserResponse{
		User: &pb.User{
			Id:       usr.Id(),
			Email:    usr.Email(),
			Username: usr.Username(),
		},
	}, nil
}
