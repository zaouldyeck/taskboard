package main

import (
	"context"
	"errors"

	"github.com/zaouldyeck/taskboard/business/core/user"
	pb "github.com/zaouldyeck/taskboard/proto/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCHandler struct {
	pb.UnimplementedUserServiceServer
	service *user.Service
}

func NewGRPCHandler(service *user.Service) *GRPCHandler {
	return &GRPCHandler{service: service}
}

func domainToProto(usr user.User) *pb.User {
	return &pb.User{
		Id:       usr.Id(),
		Email:    usr.Email(),
		Username: usr.Username(),
	}
}

// toGRPCError maps domain errors to gRPC error codes
// so that we can return error to calling grpc api-gateway code.
func toGRPCError(err error) error {
	switch {

	case errors.Is(err, user.ErrEmptyEmail),
		errors.Is(err, user.ErrEmptyUsername),
		errors.Is(err, user.ErrEmptyPassword),
		errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrWeakPassword):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, user.ErrNotFound),
		errors.Is(err, user.ErrEmptyID):
		return status.Error(codes.NotFound, err.Error())

	case errors.Is(err, user.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, err.Error())

	case errors.Is(err, user.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())

	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func (h *GRPCHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	usr, token, err := h.service.Register(ctx, req.Email, req.Username, req.Password)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.RegisterResponse{
		User:  domainToProto(usr),
		Token: token,
	}, nil
}

func (h *GRPCHandler) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	claims, valid, err := h.service.ValidateToken(ctx, req.Token)
	if err != nil {
		return nil, toGRPCError(err)
	}

	if !valid {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}

	return &pb.ValidateTokenResponse{
		Valid:  true,
		UserId: claims.UserId,
		Email:  claims.Email,
	}, nil
}

func (h *GRPCHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	usr, token, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.LoginResponse{
		User:  domainToProto(usr),
		Token: token,
	}, nil
}

func (h *GRPCHandler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	usr, err := h.service.GetUser(ctx, req.Id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetUserResponse{
		User: domainToProto(usr),
	}, nil
}
