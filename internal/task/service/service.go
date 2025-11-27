package service

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/zaouldyeck/taskboard/api/proto/task/v1"
	"github.com/zaouldyeck/taskboard/internal/task/repository"
)

// TaskService implements gRPC TaskServiceServer interface.
type TaskService struct {
	// For compatibility; if new rpc methods are added to proto,
	// older servers wont break.
	pb.UnimplementedTaskServiceServer

	repo repository.Repository
}

func NewTaskService(repo repository.Repository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	if req.Title == "" {
		// Return gRPC error for a bad request.
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.BoardId == 0 {
		return nil, status.Error(codes.InvalidArgument, "board_id is required")
	}

	// Convert protobuf to domain code, for domain/API separation.
	domainTask := &repository.Task{
		BoardID:     req.BoardId,
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedBy:   req.CreatedBy,
	}

	// Persist do DB.
	err := s.repo.Create(ctx, domainTask)
	if err != nil {
		log.Printf("Failed to create task: %v\n", err)

		// Return gRPC error for internal error.
		return nil, status.Error(codes.Internal, "failed to create task")
	}

	// Return protobuf response for gRPC API.
	pbTask := domainToProto(domainTask)

	return &pb.CreateTaskResponse{Task: pbTask}, nil
}

func domainToProto(task *repository.Task) *pb.Task {
	return &pb.Task{
		Id:          task.ID,
		BoardId:     task.BoardID,
		Title:       task.Title,
		Description: task.Description,
		Completed:   task.Completed,
		CreatedBy:   task.CreatedBy,
		CreatedAt:   timestamppb.New(task.CreatedAt),
		UpdatedAt:   timestamppb.New(task.UpdatedAt),
	}
}

func (s *TaskService) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Fetch from DB.
	task, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		fmt.Printf("Failed to get task: %v\n", err)
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	return &pb.GetTaskResponse{
		Task: domainToProto(task),
	}, nil
}

func (s *TaskService) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	if req.BoardId == 0 {
		return nil, status.Error(codes.InvalidArgument, "board_id is required")
	}

	// Setup of default pagination of the taskboard.
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 50 // Default page size.
	}
	if pageSize > 100 {
		pageSize = 100 // Max page size.
	}

	pageNumber := req.PageNumber
	if pageNumber < 1 {
		pageNumber = 1
	}
	offset := (pageNumber - 1) * pageSize

	// Optional "completed" filter.
	var completedFilter *bool
	if req.Completed != nil {
		completedFilter = req.Completed
	}

	// Fetch from DB.
	tasks, totalCount, err := s.repo.List(ctx, req.BoardId, completedFilter, int(pageSize), int(offset))
	if err != nil {
		fmt.Printf("Failed to list tasks: %v\n", err)
		return nil, status.Error(codes.Internal, "failed to list tasks")
	}

	// Convert domain model list of tasks to protobuf format.
	pbTasks := make([]*pb.Task, len(tasks))
	for i, task := range tasks {
		pbTasks[i] = domainToProto(task)
	}

	return &pb.ListTasksResponse{
		Tasks:      pbTasks,
		TotalCount: int32(totalCount),
	}, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	existingTask, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		fmt.Printf("Failed to get task: %v\n", err)
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	// Update any of the optional fields.
	if req.Title != nil {
		existingTask.Title = *req.Title
	}
	if req.Description != nil {
		existingTask.Description = *req.Description
	}
	if req.Completed != nil {
		existingTask.Completed = *req.Completed
	}

	// Save changes to task to DB.
	err = s.repo.Update(ctx, existingTask)
	if err != nil {
		fmt.Printf("Failed to update task: %v\n", err)
		return nil, status.Error(codes.Internal, "failed to update task")
	}

	return &pb.UpdateTaskResponse{
		Task: domainToProto(existingTask),
	}, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskResponse, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.repo.Delete(ctx, req.Id)
	if err != nil {
		if err.Error() == "task not found" {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		fmt.Printf("Failed to delete task: %v\n", err)
		return nil, status.Error(codes.Internal, "failed to delete task")
	}

	return &pb.DeleteTaskResponse{
		Success: true,
	}, nil
}

// WatchTasks handles server-side streaming for real-time updates.
func (s *TaskService) WatchTasks(req *pb.ListTasksRequest, stream pb.TaskService_WatchTasksServer) error {
	// TODO: Implement NATS pub/sub
	return status.Error(codes.Unimplemented, "watch tasks not yet implemented")
}
