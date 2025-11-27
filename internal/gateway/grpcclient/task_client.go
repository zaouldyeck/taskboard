package grpcclient

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/zaouldyeck/taskboard/api/proto/task/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TaskClient struct {
	client pb.TaskServiceClient
	conn   *grpc.ClientConn
}

func NewTaskClient(taskServiceAddr string) (*TaskClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect to task service.
	conn, err := grpc.DialContext(
		ctx,
		taskServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to task service at %s: %w",
			taskServiceAddr, err)
	}

	log.Printf("Connected to task service at %s", taskServiceAddr)

	return &TaskClient{
		client: pb.NewTaskServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *TaskClient) Close() error {
	return c.conn.Close()
}

func (c *TaskClient) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.Task, error) {
	resp, err := c.client.CreateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	return resp.Task, nil
}

func (c *TaskClient) GetTask(ctx context.Context, taskID int64) (*pb.Task, error) {
	resp, err := c.client.GetTask(ctx, &pb.GetTaskRequest{Id: taskID})
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return resp.Task, nil
}

func (c *TaskClient) ListTasks(ctx context.Context, boardID int64, completed *bool, pageSize,
	pageNumber int32,
) ([]*pb.Task, int32, error) {
	req := &pb.ListTasksRequest{
		BoardId:    boardID,
		PageSize:   pageSize,
		PageNumber: pageNumber,
	}

	// Only set completed filter if provided.
	if completed != nil {
		req.Completed = completed
	}

	resp, err := c.client.ListTasks(ctx, req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	return resp.Tasks, resp.TotalCount, nil
}

func (c *TaskClient) DeleteTask(ctx context.Context, taskID int64) (bool, error) {
	resp, err := c.client.DeleteTask(ctx, &pb.DeleteTaskRequest{Id: taskID})
	if err != nil {
		return false, fmt.Errorf("failed to delete task: %w", err)
	}
	return resp.Success, nil
}

// UpdateTask updates an existing task.
func (c *TaskClient) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.Task, error) {
	resp, err := c.client.UpdateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	return resp.Task, nil
}
