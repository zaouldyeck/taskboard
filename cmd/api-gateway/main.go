package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"

	"github.com/zaouldyeck/taskboard/internal/gateway/grpcclient"
	"github.com/zaouldyeck/taskboard/internal/gateway/handlers"
	ws "github.com/zaouldyeck/taskboard/internal/gateway/websocket"
)

// HTTP to WebSocket upgrader config.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // All origins. DEV ONLY!
	},
}

// // HELPER FUNCTIONS // //

// setupRoutes configures HTTP routes.
func setupRoutes(taskHandler *handlers.TaskHandler, hub *ws.Hub) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// WebSocket endpoint.
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	log.Println("✅ WebSocket endpoint registered at /ws")

	// Task endpoints - /api/tasks
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS for development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Route based on HTTP method
		switch r.Method {
		case http.MethodGet:
			taskHandler.ListTasks(w, r)
		case http.MethodPost:
			taskHandler.CreateTask(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Task endpoints - /api/tasks/:id
	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Route based on HTTP method
		switch r.Method {
		case http.MethodGet:
			taskHandler.GetTask(w, r)
		case http.MethodPut:
			taskHandler.UpdateTask(w, r)
		case http.MethodDelete:
			taskHandler.DeleteTask(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

// responseWriter wraps http.ResponseWriter to include status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// loggingMiddleware logs HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Implement http.Hijacker interface for WebSocket support.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
	}
	return h.Hijack()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// serveWs handles websocket requests from clients.
func serveWs(hub *ws.Hub, w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create new websocket client.
	client := &ws.Client{
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	client.Hub.Register <- client

	go client.WriteMsgToWebSocket()
	go client.ReadMsgFromWebSocket()
}

// // END OF HELPER FUNCTIONS // //

func main() {
	log.Println("Starting API Gateway...")

	// Read config from env vars.
	httpPort := getEnv("HTTP_PORT", "8080")
	taskServiceAddr := getEnv("TASK_SERVICE_ADDR", "localhost:50051")

	// Connect NATS message broker.
	natsURL := getEnv("NATS_URL", "nats://nats:4222")

	log.Printf("Connecting to NATS at %s...", natsURL)
	nc, err := nats.Connect(natsURL,
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
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

	// Create WebSocket hub.
	hub := ws.NewHub(nc)
	go hub.Run()
	log.Println("✅ WebSocket Hub started")

	// Init gRPC client.
	log.Printf("Connecting to task svc at %s...", taskServiceAddr)
	taskClient, err := grpcclient.NewTaskClient(taskServiceAddr)
	if err != nil {
		log.Fatalf("Failed to create task client: %v", err)
	}
	defer taskClient.Close()

	// Init handlers.
	taskHandler := handlers.NewTaskHandler(taskClient)

	// Setup HTTP router.
	mux := setupRoutes(taskHandler, hub)

	// Create HTTP server.
	server := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP server.
	go func() {
		log.Printf("API Gateway listening on :%s", httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start http server: %v", err)
		}
	}()

	// Shutdown gracefully when interrupted.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Shutdown gracefully within a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited.")
}
