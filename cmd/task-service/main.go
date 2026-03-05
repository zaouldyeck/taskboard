package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/zaouldyeck/taskboard/business/core/task"
	pb "github.com/zaouldyeck/taskboard/proto/task/v1"
	database "github.com/zaouldyeck/taskboard/sys/db"
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

	// Connect to NATS.
	natsURL := getEnv("NATS_URL", "nats://nats:4222")
	log.Printf("Connecting to NATS at %s...", natsURL)

	nc, err := nats.Connect(natsURL,
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1), // Forever reconnect.
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("Connected to NATS successfully. Server: %s", nc.ConnectedUrl())

	// Bootstrap store and service.
	store := task.NewTaskDB(db)
	taskService := task.NewService(store)

	// Create gRPC handler.
	grpcHandler := NewGRPCHandler(taskService, nc)

	grpcServer := grpc.NewServer()
	pb.RegisterTaskServiceServer(grpcServer, grpcHandler)
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
