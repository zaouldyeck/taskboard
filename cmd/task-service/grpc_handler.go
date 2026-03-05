package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zaouldyeck/taskboard/business/core/task"
	pb "github.com/zaouldyeck/taskboard/proto/task/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCHandler implements the translation between protobuf
// and business logic layer.
type GRPCHandler struct {
	pb.UnimplementedTaskServiceServer
	service *task.Service
	nats    *nats.Conn
}

type TaskEvent struct {
	Type      string `json:"type"`
	TaskID    int64  `json:"task_id"`
	BoardID   int64  `json:"board_id"`
	Title     string `json:"title"`
	Completed *bool  `json:"completed"`
	Timestamp int64  `json:"timestamp"`
}

func NewGRPCHandler(service *task.Service, nc *nats.Conn) *GRPCHandler {
	return &GRPCHandler{service: service, nats: nc}
}

// publishEvent is used for connecting handler methods event publishing with NATS.
func (h *GRPCHandler) publishEvent(eventType string, t *task.Task) {
	event := TaskEvent{
		Type:      eventType,
		TaskID:    t.ID,
		BoardID:   t.BoardID,
		Title:     t.Title,
		Timestamp: time.Now().Unix(),
	}

	if t.Completed {
		completed := true
		event.Completed = &completed
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("ERROR: Failed to marshal event: %v", err)
		return
	}

	subject := fmt.Sprintf("tasks.%s", eventType)
	if err := h.nats.Publish(subject, eventJSON); err != nil {
		log.Printf("ERROR: Failed to publish event to %s: %v", subject, err)
		return
	}

	log.Printf("Published event: %s (task_id=%d, board_id=%d)", subject, t.ID, t.BoardID)
}

// CreateTask translates between gRPC and business logic.
func (h *GRPCHandler) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse,
	error,
) {
	t, err := h.service.Create(ctx, req.BoardId, req.Title, req.Description, req.CreatedBy)
	if err != nil {
		return nil, toGRPCError(err)
	}

	h.publishEvent("created", t)

	return &pb.CreateTaskResponse{Task: domainToProto(t)}, nil
}

// GetTask translates between gRPC and business logic.
func (h *GRPCHandler) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse,
	error,
) {
	t, err := h.service.GetByID(ctx, req.Id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	return &pb.GetTaskResponse{Task: domainToProto(t)}, nil
}

// ListTasks translates between gRPC and business logic.
func (h *GRPCHandler) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse,
	error,
) {
	tasks, totalCount, err := h.service.List(ctx, req.BoardId, req.Completed, req.PageSize, req.PageNumber)
	if err != nil {
		return nil, toGRPCError(err)
	}

	pbTasks := make([]*pb.Task, len(tasks))
	for i, t := range tasks {
		pbTasks[i] = domainToProto(t)
	}

	return &pb.ListTasksResponse{
		Tasks:      pbTasks,
		TotalCount: int32(totalCount),
	}, nil
}

// UpdateTask translates between gRPC and business rules.
func (h *GRPCHandler) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse,
	error,
) {
	t, err := h.service.Update(ctx, req.Id, req.Title, req.Description, req.Completed)
	if err != nil {
		return nil, toGRPCError(err)
	}

	h.publishEvent("updated", t)

	return &pb.UpdateTaskResponse{Task: domainToProto(t)}, nil
}

// DeleteTask translates between gRPC and business rules.
func (h *GRPCHandler) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	t, err := h.service.GetByID(ctx, req.Id)
	if err != nil {
		return nil, toGRPCError(err)
	}

	if err = h.service.Delete(ctx, req.Id); err != nil {
		return nil, toGRPCError(err)
	}

	h.publishEvent("deleted", t)

	return &pb.DeleteTaskResponse{Success: true}, nil
}

func domainToProto(t *task.Task) *pb.Task {
	return &pb.Task{
		Id:          t.ID,
		BoardId:     t.BoardID,
		Title:       t.Title,
		Description: t.Description,
		Completed:   t.Completed,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   timestamppb.New(t.CreatedAt),
		UpdatedAt:   timestamppb.New(t.UpdatedAt),
	}
}

// toGRPCError maps domain errors to gRPC error codes
// so that we can return error to calling grpc api-gateway code.
func toGRPCError(err error) error {
	switch {
	case errors.Is(err, task.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, task.ErrInvalidBoardID),
		errors.Is(err, task.ErrInvalidTitle),
		errors.Is(err, task.ErrInvalidTaskID):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
