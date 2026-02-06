package grpcclient

import (
	"context"
	"fmt"
	"log"

	pb "github.com/zaouldyeck/taskboard/api/proto/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt int64  `json:"created_at,omitempty"`
}

// AuthResponse is returned after successful login or register new user.
type AuthResponse struct {
	User  *User  `json:"user"`
	Token string `json:"token"`
}

// TokenValidation is used as a return type when validating JWT.
type TokenValidation struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
}

// UserClient is used for wrapping user service gRPC client.
type UserClient struct {
	client pb.UserServiceClient
	conn   *grpc.ClientConn
}

// NewUserClient wraps grpc connection to user service with context
// and gives us a user service grpc client, so we can send messages over the connection.
func NewUserClient(userServiceAddr string) (*UserClient, error) {
	conn, err := grpc.NewClient(
		userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user service client for %s: %w", userServiceAddr, err)
	}

	log.Printf("Connected to user service at %s", userServiceAddr)

	return &UserClient{
		client: pb.NewUserServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *UserClient) Close() error {
	return c.conn.Close()
}

func (c *UserClient) Register(ctx context.Context, email, username, password string) (*AuthResponse,
	error,
) {
	resp, err := c.client.Register(ctx, &pb.RegisterRequest{
		Email:    email,
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("register response missing user data")
	}

	return &AuthResponse{
		User: &User{
			ID:        resp.User.Id,
			Email:     resp.User.Email,
			Username:  resp.User.Username,
			CreatedAt: resp.User.CreatedAt,
		},
		Token: resp.Token,
	}, nil
}

func (c *UserClient) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	resp, err := c.client.Login(ctx, &pb.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to login user: %w", err)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("login response missing user data")
	}

	return &AuthResponse{
		User: &User{
			ID:        resp.User.Id,
			Email:     resp.User.Email,
			Username:  resp.User.Username,
			CreatedAt: resp.User.CreatedAt,
		},
		Token: resp.Token,
	}, nil
}

func (c *UserClient) ValidateToken(ctx context.Context, token string) (*TokenValidation, error) {
	resp, err := c.client.ValidateToken(ctx, &pb.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	return &TokenValidation{
		Valid:  resp.Valid,
		UserID: resp.UserId,
		Email:  resp.Email,
	}, nil
}

// GetUser pulls the *User to be used where that information is needed, e.g
// to view user profile page.
func (c *UserClient) GetUser(ctx context.Context, userID string) (*User, error) {
	resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{
		Id: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if resp.User == nil {
		return nil, fmt.Errorf("user not found")
	}

	return &User{
		ID:        resp.User.Id,
		Email:     resp.User.Email,
		Username:  resp.User.Username,
		CreatedAt: resp.User.CreatedAt,
	}, nil
}
