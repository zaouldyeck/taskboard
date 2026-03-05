# 🚀 Taskboard - Real-Time Microservices Application

A basic real-time task management system built with Go microservices, Kubernetes (Talos), NATS messaging, and WebSocket for live updates.

Instructions for deploying it locally on a single node Docker-based Talos cluster included.

## 📋 Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Testing](#testing)
- [Architecture Deep Dive](#architecture-deep-dive)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

---

## 🏗️ Architecture

### System Overview
```
┌─────────┐                                                    
│ Browser │                                                    
│ (HTTP + │                                                    
│   WS)   │                                                    
└────┬────┘                                                    
     │                                                         
     ▼                                                         
┌─────────────────┐          ┌──────────────┐                
│  API Gateway    │          │     NATS     │                
│                 │◄─────────┤  Message Bus │                
│ • HTTP → gRPC   │ events   │              │                
│ • WebSocket Hub │          └──────▲───────┘                
│ • CORS          │                 │                        
└────┬────────────┘                 │                        
     │ gRPC                         │ publish                
     ▼                              │                        
┌─────────────────┐                 │                        
│  Task Service   │─────────────────┘                        
│                 │                                           
│ • Business logic│                                           
│ • gRPC server   │                                           
│ • Event publish │                                           
└────┬────────────┘                                           
     │ SQL                                                    
     ▼                                                         
┌─────────────────┐                                           
│   PostgreSQL    │                                           
│                 │                                           
│ • Task storage  │                                           
│ • Persistent DB │                                           
└─────────────────┘                                           
```

### Components

| Component | Technology | Port | Purpose |
|-----------|-----------|------|---------|
| **API Gateway** | Go + Gorilla WebSocket | 8080 | HTTP REST, WebSocket, gRPC client |
| **Task Service** | Go + gRPC | 50051 | Business logic, NATS publisher |
| **PostgreSQL** | PostgreSQL 15 | 5432 | Persistent storage |
| **NATS** | NATS Server 2.10 | 4222 | Message broker |
| **Kubernetes** | Talos Linux | - | Container orchestration |

### Event Flow
```
1. User creates task
   ↓
2. Browser → API Gateway (HTTP POST)
   ↓
3. API Gateway → Task Service (gRPC)
   ↓
4. Task Service → PostgreSQL (INSERT)
   ↓
5. Task Service → NATS (publish event)
   ↓
6. NATS → API Gateway (event delivery)
   ↓
7. API Gateway → All WebSocket clients (broadcast)
   ↓
8. All users see update instantly! ✨
```

---

## 🔧 Prerequisites

### Required Software

Ensure you have the following installed:

- Docker 20.10+
- Go 1.23+
- Helm 3.12+
- Helmfile 0.157+
- kubectl 1.28+
- talosctl 1.6+

### Installation Instructions

#### macOS
```bash
# Using Homebrew
brew install docker
brew install go
brew install helm
brew install helmfile
brew install kubectl

# Install talosctl
brew install siderolabs/tap/talosctl
```

#### Linux
```bash
# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Go
wget https://go.dev/dl/go1.23.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Helmfile
wget https://github.com/helmfile/helmfile/releases/download/v0.157.0/helmfile_linux_amd64
chmod +x helmfile_linux_amd64
sudo mv helmfile_linux_amd64 /usr/local/bin/helmfile

# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# talosctl
curl -sL https://talos.dev/install | sh
```

---

## 🚀 Quick Start

Get the entire system running in minutes:
```bash
# Clone repository
git clone https://github.com/zaouldyeck/taskboard.git
cd taskboard

# Deploy everything (creates cluster, registry, and all services)
./scripts/deploy.sh
```

The deploy script will:
1. Create Talos Kubernetes cluster in Docker
2. Set up local Docker registry
3. Build and push service images
4. Deploy PostgreSQL, NATS, Task Service, and API Gateway
5. Wait for all pods to be ready

**Verify deployment:**
```bash
kubectl get pods -n taskboard

# Should show all Running:
# NAME                           READY   STATUS    RESTARTS   AGE
# taskboard-db-postgres-0        1/1     Running   0          2m
# nats-0                         1/1     Running   0          2m
# task-service-xxxxx             1/1     Running   0          1m
# api-gateway-xxxxx              1/1     Running   0          1m
# api-gateway-xxxxx              1/1     Running   0          1m
```

---

## 🧪 Testing

### Test REST API
```bash
# Port forward to access API
kubectl port-forward -n taskboard svc/api-gateway 8080:8080 &

# Health check
curl http://localhost:8080/health
# Returns: OK

# Create a task
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "board_id": 1,
    "title": "My first task",
    "description": "Testing the API",
    "created_by": 100
  }' | jq .

# List tasks
curl "http://localhost:8080/api/tasks?board_id=1" | jq .

# Update task
curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"completed": true}' | jq .

# Delete task
curl -X DELETE http://localhost:8080/api/tasks/1
```

### Test Real-Time WebSocket

**Open the test page:**
```bash
# Copy test page to accessible location
cp test/taskboard-test.html /tmp/

# Open in browser
# macOS:
open /tmp/taskboard-test.html

# Linux:
xdg-open /tmp/taskboard-test.html

# Or manually navigate to: file:///tmp/taskboard-test.html
```

**What to expect:**

1. ✅ Green "Connected to WebSocket" status appears
2. Type a task title and click "Create Task"
3. **Watch the event appear instantly in the Live Events area!**
4. Open the page in 2 browser windows side-by-side
5. Create a task in one window
6. **See it appear in both windows simultaneously!** ✨

### Monitor Event Flow

**Terminal 1 - Watch task service publish events:**
```bash
kubectl logs -n taskboard -l app=task-service -f | grep "📤"
```

**Terminal 2 - Watch API gateway broadcast events:**
```bash
kubectl logs -n taskboard -l app=api-gateway -f | grep "📨"
```

**Terminal 3 - Create a task:**
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"board_id": 1, "title": "Real-time test", "created_by": 100}'
```

**Expected output:**

Terminal 1:
```
📤 Published event: tasks.created (task_id=1, board_id=1)
```

Terminal 2:
```
📨 Received NATS event on tasks.created, broadcasting to N clients
```

### Load Testing

Test system performance under load:
```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Run load test: 1000 requests, 10 concurrent
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"board_id":1,"title":"Load test","created_by":100}' \
  http://localhost:8080/api/tasks
```

---

## 🎯 Architecture Deep Dive

### Why Event-Driven Architecture?

**Traditional polling approach:**
```
Browser: "Any updates?" → Server: "No"
Browser: "Any updates?" → Server: "No"
Browser: "Any updates?" → Server: "No"
Browser: "Any updates?" → Server: "Yes, here's 1 update"

Problems:
- Wastes bandwidth (many empty requests)
- High latency (only checks every N seconds)
- Server load (constant polling)
```

**Our WebSocket + NATS approach:**
```
Browser ←─WebSocket─→ API Gateway ←─NATS─→ Task Service
                           ↓
                    Event arrives
                           ↓
                 Instant push to browser

Benefits:
- Minimal bandwidth (only send actual updates)
- Low latency (instant delivery)
- Lower server load (persistent connections)
```

### Microservices Benefits

**Separation of Concerns:**
- **Task Service**: Business logic only
- **API Gateway**: Protocol translation (HTTP/WebSocket ↔ gRPC)
- **NATS**: Message routing

**Independent Scaling:**
```bash
# Scale API Gateway for more WebSocket connections
kubectl scale deployment api-gateway -n taskboard --replicas=5

# Scale Task Service for more gRPC capacity
kubectl scale deployment task-service -n taskboard --replicas=3
```

**Technology Flexibility:**
- Add mobile app → Just consume NATS events
- Add analytics → Subscribe to NATS topics
- Replace components → One service at a time

### WebSocket Hub Pattern

The Hub manages all WebSocket connections as a central switchboard:
```go
type Hub struct {
    clients    map[*Client]bool  // All connected browsers
    register   chan *Client      // New connections
    unregister chan *Client      // Disconnections
    broadcast  chan []byte       // Messages to broadcast
}
```

**Why this pattern?**
- **Thread-safe**: Single goroutine manages clients map
- **Non-blocking**: Channels buffer messages
- **Efficient**: Broadcast to all clients in one loop

### NATS Pub/Sub Pattern

**Publisher (Task Service):**
```go
nats.Publish("tasks.created", eventJSON)
```

**Subscriber (API Gateway):**
```go
nats.Subscribe("tasks.>", handleEvent)  // Wildcard: all task events
```

**Subject Hierarchy:**
```
tasks.created       ← Task created events
tasks.updated       ← Task updated events
tasks.deleted       ← Task deleted events
tasks.>             ← Wildcard: all task events

boards.>            ← Could add board events
users.>             ← Could add user events
```

**Benefits:**
- **Decoupled**: Services don't know about each other
- **Scalable**: Add subscribers without changing publishers
- **Flexible**: Different services subscribe to different topics

### Database Schema
```sql
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    board_id BIGINT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    completed BOOLEAN DEFAULT FALSE,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_tasks_board_id ON tasks(board_id);
CREATE INDEX idx_tasks_completed ON tasks(completed);
```

**Design decisions:**
- SERIAL for auto-incrementing IDs
- BIGINT for foreign keys (allows large scale)
- Indexes on common query filters
- Timestamps for audit trail

---

## 💻 Development

### Project Structure
```
taskboard/
├── proto/task/v1/          # gRPC API definitions
│   ├── task.proto
│   └── task_grpc.pb.go
├── cmd/
│   ├── api-gateway/
│   │   ├── main.go
│   │   └── Dockerfile
│   └── task-service/
│       ├── main.go
│       └── Dockerfile
├── internal/
│   ├── database/               # Database connection & schema
│   ├── gateway/
│   │   ├── grpcclient/         # gRPC client
│   │   ├── handlers/           # HTTP handlers
│   │   └── websocket/          # WebSocket hub & clients
│   └── task/
│       ├── repository/         # Data access layer
│       └── service/            # Business logic
├── deploy/
│   ├── environments/           # Environment configs
│   │   ├── dev.yaml
│   │   └── prod.yaml
│   └── helm/                   # Helm charts
│       ├── postgres/
│       ├── nats/
│       ├── task-service/
│       └── api-gateway/
├── scripts/
│   ├── deploy.sh               # Build & deploy everything
│   └── destroy.sh              # Teardown cluster
├── test/
│   └── taskboard-test.html     # WebSocket test page
├── helmfile.yaml.gotmpl        # Helmfile configuration
└── README.md
```

### Building Services Manually
```bash
# Build task-service
docker build -t taskboard-task-service:latest -f cmd/task-service/Dockerfile .

# Build api-gateway
docker build -t taskboard-api-gateway:latest -f cmd/api-gateway/Dockerfile .

# Tag and push to local registry
docker tag taskboard-task-service:latest localhost:5010/taskboard-task-service:latest
docker push localhost:5010/taskboard-task-service:latest

docker tag taskboard-api-gateway:latest localhost:5010/taskboard-api-gateway:latest
docker push localhost:5010/taskboard-api-gateway:latest

# Redeploy with new images
export REGISTRY_IP=$(docker inspect registry -f '{{range $net, $config := .NetworkSettings.Networks}}{{if eq $net "taskboard"}}{{$config.IPAddress}}{{end}}{{end}}')
helmfile sync
```

### Local Development (Outside Kubernetes)

Run services locally for faster development:

**Terminal 1 - PostgreSQL:**
```bash
docker run --rm --name postgres \
  -e POSTGRES_USER=taskboard \
  -e POSTGRES_PASSWORD=taskboard \
  -e POSTGRES_DB=taskboard \
  -p 5432:5432 postgres:15
```

**Terminal 2 - NATS:**
```bash
docker run --rm --name nats -p 4222:4222 nats:2.10
```

**Terminal 3 - Task Service:**
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=taskboard
export DB_PASSWORD=taskboard
export DB_NAME=taskboard
export NATS_URL=nats://localhost:4222
export PORT=50051

go run cmd/task-service/main.go
```

**Terminal 4 - API Gateway:**
```bash
export HTTP_PORT=8080
export TASK_SERVICE_ADDR=localhost:50051
export NATS_URL=nats://localhost:4222

go run cmd/api-gateway/main.go
```

---

## 🐛 Troubleshooting

### Cluster Not Ready

**Check node status:**
```bash
kubectl get nodes

# If NotReady, check Talos health:
talosctl health --nodes 10.5.0.2
```

**Pods not scheduling:**
```bash
kubectl describe pod <pod-name> -n taskboard

# Common fix: Remove control plane taint
kubectl taint nodes --all node-role.kubernetes.io/control-plane:NoSchedule-
```

### Image Pull Failures

**Check registry IP:**
```bash
kubectl get deployment task-service -n taskboard -o jsonpath='{.spec.template.spec.containers[0].image}'

# Should show: 10.5.0.x:5000/taskboard-task-service:latest
# NOT: 172.17.0.2:5000/...
```

**Fix registry IP:**
```bash
export REGISTRY_IP=$(docker inspect registry -f '{{range $net, $config := .NetworkSettings.Networks}}{{if eq $net "taskboard"}}{{$config.IPAddress}}{{end}}{{end}}')

helmfile sync
```

### WebSocket Connection Failed

**Check endpoint exists:**
```bash
curl -i http://localhost:8080/ws

# Should return: HTTP/1.1 426 Upgrade Required (this is correct!)
```

**Check logs:**
```bash
kubectl logs -n taskboard -l app=api-gateway | grep -E "WebSocket|Hub|subscribed"

# Should see:
# ✅ WebSocket Hub started
# ✅ WebSocket endpoint registered
# ✅ Hub subscribed to tasks.* events
```

### Events Not Appearing

**Check NATS connections:**
```bash
# API Gateway
kubectl logs -n taskboard -l app=api-gateway | grep NATS
# Should see: Connected to NATS successfully

# Task Service
kubectl logs -n taskboard -l app=task-service | grep NATS
# Should see: Connected to NATS successfully
```

**Check event flow:**
```bash
# Publishing
kubectl logs -n taskboard -l app=task-service | grep "📤"
# Should see: 📤 Published event: tasks.created

# Broadcasting
kubectl logs -n taskboard -l app=api-gateway | grep "📨"
# Should see: 📨 Received NATS event on tasks.created
```

### Complete Reset

If all else fails, start fresh:
```bash
# Destroy everything
./scripts/destroy.sh

# Wait for cleanup
sleep 10

# Deploy again
./scripts/deploy.sh
```

---

## 📊 Monitoring

### System Health
```bash
# Pod status
kubectl get pods -n taskboard

# Resource usage
kubectl top pods -n taskboard

# Service endpoints
kubectl get endpoints -n taskboard

# Recent events
kubectl get events -n taskboard --sort-by='.lastTimestamp'
```

### Live Logs
```bash
# API Gateway logs
kubectl logs -n taskboard -l app=api-gateway -f

# Task Service logs
kubectl logs -n taskboard -l app=task-service -f

# PostgreSQL logs
kubectl logs -n taskboard -l app=taskboard-db -f

# NATS logs
kubectl logs -n taskboard nats-0 -f
```

### Metrics

**WebSocket connections:**
```bash
kubectl logs -n taskboard -l app=api-gateway | grep "Total clients"
```

**Event throughput:**
```bash
kubectl logs -n taskboard -l app=task-service | grep "📤" | wc -l
```

---

## 🎓 Next Steps

### Feature Enhancements

- [ ] Add authentication (JWT)
- [ ] Implement boards CRUD
- [ ] Add user management
- [ ] Implement task assignment
- [ ] Add file attachments
- [ ] Implement search functionality
- [ ] Add email notifications
- [ ] Implement audit logs
- [ ] Add unit and integration tests

### Production Readiness

- [ ] Add Prometheus metrics
- [ ] Implement distributed tracing (Jaeger)
- [ ] Set up log aggregation (Loki)
- [ ] Add comprehensive health checks
- [ ] Implement rate limiting
- [ ] Add circuit breakers
- [ ] Set up automated backups
- [ ] Implement disaster recovery

---

## 📝 License

MIT License - see LICENSE file for details

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Commit your changes: `git commit -m 'Add amazing feature'`
4. Push to the branch: `git push origin feature/amazing-feature`
5. Open a Pull Request

---

## 👥 Author

**Paul Ohrt** - [@zaouldyeck](https://github.com/zaouldyeck)

---

## 🙏 Acknowledgments

Built with:
- [Go](https://golang.org/) - Programming language
- [gRPC](https://grpc.io/) - RPC framework
- [NATS](https://nats.io/) - Message broker
- [PostgreSQL](https://www.postgresql.org/) - Database
- [Kubernetes](https://kubernetes.io/) - Container orchestration
- [Talos Linux](https://www.talos.dev/) - Secure Kubernetes OS
- [Helm](https://helm.sh/) - Kubernetes package manager
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket library

---

⭐ **If this project helped you, please give it a star!**
