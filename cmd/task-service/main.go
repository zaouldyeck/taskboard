package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/zaouldyeck/taskboard/internal/task/repository"
	"github.com/zaouldyeck/taskboard/internal/task/service"
	pb "github.com/zaouldyeck/taskboard/pkg/api/task/v1"
	"github.com/zaouldyeck/taskboard/pkg/database"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Setup logging.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting task service...")

	// Connect to DB.
	cfg := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "taskboard"),
		Password: getEnv("DB_PASSWORD", "taskboard"),
		Database: getEnv("DB_NAME", "taskboard"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
	log.Printf("Connecting to db at %s:%d...", cfg.Host, cfg.Port)
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to DB successfully.")

	// Init of DB schema.
	log.Println("Initializing DB schema...")
	if err := database.InitSchema(db); err != nil {
		log.Fatalf("Failed to init schema: %v", err)
	}
	log.Println("DB schema initialized.")

	// Bootstrap postgres repo and services.
	repo := repository.NewPostgresRepository(db)
	taskService := service.NewTaskService(repo)
	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, taskService)
	reflection.Register(grpcServer)

	// TCP listener.
	port := getEnv("PORT", "50051")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	// Shutdown gracefully when interrupt signal is caught.
	// (Ctrl + C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("gRPC Server listening on port %s", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down gracefully...")
	grpcServer.GracefulStop()
	log.Println("Server stopped.")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
