package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/zaouldyeck/taskboard/api/proto/task/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()), // No TLS. Only in dev!

	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewTaskServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create task.
	fmt.Println("Creating a task...")
	createResp, err := client.CreateTask(ctx, &pb.CreateTaskRequest{
		BoardId:     1,
		Title:       "Learn grpc",
		Description: "build a task svc with grpc and postgres",
		CreatedBy:   100,
	})
	if err != nil {
		log.Fatalf("Failed to create task: %v", err)
	}

	taskID := createResp.Task.Id
	fmt.Printf("✓ Created task with ID: %d\n", taskID)
	fmt.Printf("  Title: %s\n", createResp.Task.Title)
	fmt.Printf("  Created at: %s\n", createResp.Task.CreatedAt.AsTime().Format(time.RFC3339))

	// Get the task.
	fmt.Println("\nGetting the task...")
	getResp, err := client.GetTask(ctx, &pb.GetTaskRequest{
		Id: taskID,
	})
	if err != nil {
		log.Fatalf("Failed to get task: %v", err)
	}

	fmt.Printf("✓ Retrieved task: %s\n", getResp.Task.Title)
	fmt.Printf("  Completed: %v\n", getResp.Task.Completed)

	// Update the task.
	fmt.Println("\nUpdating the task...")
	completed := true
	updateResp, err := client.UpdateTask(ctx, &pb.UpdateTaskRequest{
		Id:        taskID,
		Completed: &completed,
	})
	if err != nil {
		log.Fatalf("Failed to update task: %v", err)
	}

	fmt.Printf("✓ Updated task\n")
	fmt.Printf("  Completed: %v\n", updateResp.Task.Completed)
	fmt.Printf("  Updated at: %s\n", updateResp.Task.UpdatedAt.AsTime().Format(time.RFC3339))

	// List tasks.
	fmt.Println("\nListing all tasks for board 1...")
	listResp, err := client.ListTasks(ctx, &pb.ListTasksRequest{
		BoardId:    1,
		PageSize:   10,
		PageNumber: 1,
	})
	if err != nil {
		log.Fatalf("Failed to list tasks: %v", err)
	}

	fmt.Printf("✓ Found %d tasks (total: %d)\n", len(listResp.Tasks), listResp.TotalCount)
	for i, task := range listResp.Tasks {
		status := "⬜"
		if task.Completed {
			status = "✅"
		}
		fmt.Printf("  %d. %s %s\n", i+1, status, task.Title)
	}

	// Delete the task.
	fmt.Println("\nDeleting the task...")
	deleteResp, err := client.DeleteTask(ctx, &pb.DeleteTaskRequest{
		Id: taskID,
	})
	if err != nil {
		log.Fatalf("Failed to delete task: %v", err)
	}

	if deleteResp.Success {
		fmt.Printf("✓ Deleted task %d\n", taskID)
	}

	fmt.Println("\n✨ All operations completed successfully!")
}
