package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zaouldyeck/taskboard/business/core/user"
	pb "github.com/zaouldyeck/taskboard/proto/user/v1"
	"github.com/zaouldyeck/taskboard/sys/auth"
	database "github.com/zaouldyeck/taskboard/sys/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Logging format for seeing source code file and line number.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 Starting User Service...")

	cfg := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "taskboard"),
		Password: getEnv("DB_PASSWORD", "taskboard"),
		Database: getEnv("DB_NAME", "taskboard"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	log.Printf("📊 Connecting to database at %s:%d...", cfg.Host, cfg.Port)

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✅ Connected to database successfully")

	// Init DB schema for users table so that we can store user
	// login session data.
	log.Println("📋 Initializing database schema...")

	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id				TEXT PRIMARY KEY,
			email			TEXT UNIQUE NOT NULL,
			username		TEXT UNIQUE NOT NULL,
			password_hash	TEXT NOT NULL,
			created_at		TIMESTAMP DEFAULT NOW(),
			updated_at		TIMESTAMP DEFAULT NOW()
		);
	`

	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("❌ Failed to initialize schema: %v", err)
	}
	log.Println("✅ Database schema initialized")

	userStore := user.NewStore(db)
	log.Println("✅ User store initialized")

	// Init auth handler for JWT handling.
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-not-in-prod")
	authHandler := auth.NewAuth(jwtSecret)
	log.Println("✅ Auth handler initialized")

	// Set token validity duration needed for JWT to function.
	tokenExpiry := 24 * time.Hour

	userService := user.NewService(user.Config{
		Store:       userStore,
		Auth:        authHandler,
		TokenExpiry: tokenExpiry,
	})
	log.Println("✅ User service initialized")

	grpcServer := grpc.NewServer()

	// Register user-service with grpc, connecting gRPC protobuf with our
	// internal handler code.
	pb.RegisterUserServiceServer(grpcServer, userService)

	// Enable reflection to use grpcurl and other debugging for testing,
	// exposing the API.
	reflection.Register(grpcServer)
	log.Println("✅ gRPC server configured")

	// Allow for incoming connections.
	port := getEnv("PORT", "50052")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("❌ Failed to listen on port %s: %v", port, err)
	}
	log.Printf("✅ TCP listener created on port %s", port)

	// Setup for graceful shutdown, so that we have time to exit cleanly.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run gRPC server concurrently, and allow for shutdown signaling.
	go func() {
		log.Printf("🎧 User Service listening on port %s", port)
		log.Println("📡 Press Ctrl+C to shutdown")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("❌ Failed to serve: %v", err)
		}
	}()

	<-sigChan
	log.Println("\n🛑 Shutdown signal received, stopping server...")

	// Shutting down gracefully.
	grpcServer.GracefulStop()

	log.Println("👋 User Service stopped gracefully")
}

// getEnv to use env vars with a specified default value, unless it is set
// with a custom value. Convenience helper function and to avoid hardcode
// user provided environment variable values.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
