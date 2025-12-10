# 🚀 Taskboard - Real-Time Microservices Application

A production-ready, real-time task management system built with Go microservices, Kubernetes, NATS messaging, and WebSocket for live updates.

## 📋 Table of Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Detailed Setup](#detailed-setup)
- [Testing](#testing)
- [Architecture Deep Dive](#architecture-deep-dive)
- [Development](#development)
- [Troubleshooting](#troubleshooting)

---

## 🏗️ Architecture

### System Overview
```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                             │
│  ┌─────────────┐              ┌──────────────────┐        │
│  │  REST API   │              │   WebSocket      │        │
│  │  (HTTP)     │              │   (Real-time)    │        │
│  └──────┬──────┘              └────────┬─────────┘        │
└─────────┼────────────────────────────────┼─────────────────┘
          │                                │
          ▼                                ▼
┌─────────────────────────────────────────────────────────────┐
│                    API Gateway                              │
│  ┌──────────────────────────────────────────────────────┐ │
│  │  - HTTP → gRPC translation                           │ │
│  │  - WebSocket Hub (manages connections)               │ │
│  │  - CORS handling                                      │ │
│  └──────────────────────────────────────────────────────┘ │
└────────┬──────────────────────────────────┬───────────────┘
         │ gRPC                             │ NATS
         │                                  │ subscription
         ▼                                  │
┌─────────────────────────────┐            │
│      Task Service           │            │
│  ┌──────────────────────┐  │            │
│  │  - Business logic    │  │            │
│  │  - gRPC server       │  │            │
│  │  - Event publishing ─┼──┼────────────┘
│  └──────────┬───────────┘  │
└─────────────┼───────────────┘
              │ SQL
              ▼
┌─────────────────────────────┐       ┌──────────────────┐
│      PostgreSQL             │       │      NATS        │
│  - Task storage             │       │  - Message bus   │
│  - Persistent data          │       │  - Pub/Sub       │
└─────────────────────────────┘       └──────────────────┘
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
```bash
# Docker (for Talos cluster and registry)
docker --version  # Should be 20.10+

# Go (for building services)
go version  # Should be 1.21+

# Helm (for Kubernetes package management)
helm version  # Should be 3.12+

# Helmfile (for managing multiple Helm releases)
helmfile --version  # Should be 0.157+

# kubectl (for Kubernetes management)
kubectl version  # Should be 1.28+

# talosctl (for Talos cluster management)
talosctl version  # Should be 1.6+
```

### Installation Instructions

<details>
<summary><b>macOS</b></summary>
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
</details>

<details>
<summary><b>Linux</b></summary>
```bash
# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
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
</details>

---

## 🚀 Quick Start

Get the entire system running in 5 minutes:
```bash
# 1. Clone repository
git clone https://github.com/zaouldyeck/taskboard.git
cd taskboard

# 2. Create Talos cluster
./scripts/create-cluster.sh

# 3. Deploy everything
./scripts/deploy.sh

# 4. Test it works
./scripts/test.sh
```

**Open browser to test page to see real-time updates!**

---

## 📖 Detailed Setup

### Step 1: Create Talos Kubernetes Cluster

Talos is a modern, secure, and minimal Kubernetes distribution.
```bash
# Run the cluster creation script
./scripts/create-cluster.sh
```

**What this does:**
1. Creates Docker network (`taskboard`)
2. Generates Talos configuration
3. Spins up Talos control plane node
4. Bootstraps Kubernetes
5. Installs Helm controller
6. Configures kubectl context

**Verify cluster:**
```bash
kubectl get nodes
# Should show:
# NAME                 STATUS   ROLES           AGE   VERSION
# talos-demo-cluster   Ready    control-plane   2m    v1.29.0
```

---

### Step 2: Set Up Docker Registry

We need a local registry for our container images:
```bash
./scripts/registry.sh
```

**What this does:**
1. Creates persistent volume for images
2. Starts registry on port 5010
3. Connects to `taskboard` network
4. Configures registry for Kubernetes access

**Verify registry:**
```bash
curl http://localhost:5010/v2/_catalog
# Should return: {"repositories":[]}
```

---

### Step 3: Deploy Infrastructure

Deploy database, message broker, and services:
```bash
# Set registry IP (required for Helmfile)
export REGISTRY_IP=$(docker inspect registry -f '{{range $net, $config := .NetworkSettings.Networks}}{{if eq $net "taskboard"}}{{$config.IPAddress}}{{end}}{{end}}')

echo "Registry IP: $REGISTRY_IP"  # Should show 10.5.0.x

# Deploy everything
./scripts/deploy.sh
```

**Deployment order:**
1. **PostgreSQL** - Database for task storage
2. **NATS** - Message broker for events
3. **Task Service** - Business logic + gRPC API
4. **API Gateway** - HTTP REST + WebSocket

**What happens:**
1. Builds Docker images for services
2. Pushes to local registry
3. Deploys via Helmfile in dependency order
4. Waits for all pods to be ready

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

### Step 4: Test the System

#### A. Test REST API
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

# Get task
curl http://localhost:8080/api/tasks/1 | jq .

# List tasks
curl "http://localhost:8080/api/tasks?board_id=1" | jq .

# Update task
curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"completed": true}' | jq .

# Delete task
curl -X DELETE http://localhost:8080/api/tasks/1
```

#### B. Test Real-Time WebSocket

1. **Copy test page:**
```bash
cp test/taskboard-test.html /tmp/
```

2. **Open in browser:**
```bash
# macOS
open /tmp/taskboard-test.html

# Linux
xdg-open /tmp/taskboard-test.html

# Or manually open: file:///tmp/taskboard-test.html
```

3. **You should see:**
   - ✅ Green "Connected to WebSocket" status
   - Empty events area

4. **Create a task:**
   - Type task title
   - Click "Create Task"
   - **Watch event appear instantly!**

5. **Test multi-user:**
   - Open test page in 2 browser windows side-by-side
   - Create task in window 1
   - **See it appear in window 2 instantly!** ✨

#### C. Test Event Flow

**Terminal 1 - Watch events:**
```bash
kubectl logs -n taskboard -l app=task-service -f | grep "📤"
```

**Terminal 2 - Watch broadcasts:**
```bash
kubectl logs -n taskboard -l app=api-gateway -f | grep "📨"
```

**Terminal 3 - Create task:**
```bash
curl -X POST http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"board_id": 1, "title": "Event test", "created_by": 100}'
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

---

## 🎯 Architecture Deep Dive

### Why This Architecture?

#### Event-Driven Design

**Traditional approach (polling):**
```
Browser: "Any updates?" → Server: "No"
Browser: "Any updates?" → Server: "No"
Browser: "Any updates?" → Server: "No"
Browser: "Any updates?" → Server: "Yes, here's 1 update"

Problems:
- Wastes bandwidth (many empty requests)
- High latency (only checks every N seconds)
- Server load (constant requests)
```

**Our approach (WebSocket + events):**
```
Browser ←─WebSocket─→ Server
                       ↑
                     NATS event arrives
                       ↓
               Instant push to browser

Benefits:
- Minimal bandwidth (only send actual updates)
- Low latency (instant delivery)
- Lower server load (persistent connections)
```

#### Microservices Benefits

**Separation of Concerns:**
- **Task Service**: Business logic only (doesn't care about HTTP/WebSocket)
- **API Gateway**: Protocol translation (doesn't care about business logic)
- **NATS**: Message routing (doesn't care about content)

**Independent Scaling:**
```bash
# Scale API Gateway (more WebSocket connections)
kubectl scale deployment api-gateway -n taskboard --replicas=5

# Scale Task Service (more gRPC capacity)
kubectl scale deployment task-service -n taskboard --replicas=3
```

**Technology Flexibility:**
- Want to add mobile app? Just consume NATS events
- Want to add analytics? Subscribe to NATS topics
- Want to replace Go with Rust? One service at a time

### WebSocket Implementation

#### Hub Pattern

The Hub acts as a central switchboard:
```go
type Hub struct {
    clients    map[*Client]bool  // All connected browsers
    register   chan *Client      // New connections
    unregister chan *Client      // Disconnections
    broadcast  chan []byte       // Messages to send
}
```

**Why this pattern?**
- **Thread-safe**: Single goroutine manages clients map
- **Non-blocking**: Channels buffer messages
- **Efficient broadcasting**: Send to all clients in one loop

#### Ping/Pong Keepalive

WebSocket connections use ping/pong to detect dead connections:
```
Every 54 seconds:
  Server → Ping → Client
  Client → Pong → Server

If no pong in 60 seconds:
  Connection considered dead
  Server closes it
```

**Why?**
- Mobile devices can drop connections silently
- Firewalls can close idle connections
- Prevents resource leaks

### NATS Messaging

#### Pub/Sub Pattern

**Publisher (Task Service):**
```go
nats.Publish("tasks.created", eventJSON)
```

**Subscriber (API Gateway):**
```go
nats.Subscribe("tasks.>", handleEvent)  // Wildcard subscription
```

**Benefits:**
- Decoupled: Services don't know about each other
- Scalable: Add more subscribers without changing publishers
- Flexible: Different services can subscribe to different topics

#### Subject Hierarchy
```
tasks.created       ← Task created events
tasks.updated       ← Task updated events
tasks.deleted       ← Task deleted events
tasks.>             ← All task events (wildcard)

boards.created      ← Could add board events
boards.>            ← All board events

users.>             ← Could add user events
```

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
- `SERIAL` for auto-incrementing IDs
- `BIGINT` for foreign keys (allows large scale)
- Indexes on common query filters
- Timestamps for audit trail

---

## 💻 Development

### Project Structure
```
taskboard/
├── api/
│   └── proto/
│       └── task/
│           └── v1/
│               ├── task.proto           # gRPC API definition
│               └── task_grpc.pb.go      # Generated code
├── cmd/
│   ├── api-gateway/
│   │   ├── main.go                      # Gateway entrypoint
│   │   └── Dockerfile
│   └── task-service/
│       ├── main.go                      # Service entrypoint
│       └── Dockerfile
├── internal/
│   ├── database/
│   │   ├── postgres.go                  # DB connection
│   │   └── schema.go                    # Schema initialization
│   ├── gateway/
│   │   ├── grpcclient/                  # gRPC client
│   │   ├── handlers/                    # HTTP handlers
│   │   └── websocket/                   # WebSocket hub & clients
│   │       ├── hub.go
│   │       └── client.go
│   └── task/
│       ├── repository/                  # Data access layer
│       │   └── postgres.go
│       └── service/                     # Business logic
│           └── service.go
├── deploy/
│   ├── environments/
│   │   ├── dev.yaml                     # Dev config
│   │   └── prod.yaml                    # Prod config
│   └── helm/
│       ├── postgres/                    # Postgres chart
│       ├── nats/                        # NATS chart
│       ├── task-service/                # Task service chart
│       └── api-gateway/                 # Gateway chart
├── scripts/
│   ├── create-cluster.sh                # Talos cluster setup
│   ├── registry.sh                      # Registry setup
│   ├── deploy.sh                        # Build & deploy
│   ├── destroy.sh                       # Teardown
│   └── test.sh                          # Integration tests
├── test/
│   └── taskboard-test.html              # WebSocket test page
├── helmfile.yaml.gotmpl                 # Helmfile configuration
├── go.mod
├── go.sum
└── README.md
```

### Building Services
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
```

### Updating Protobuf

If you modify `api/proto/task/v1/task.proto`:
```bash
# Install protoc compiler
brew install protobuf  # macOS
# or
apt-get install protobuf-compiler  # Linux

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/proto/task/v1/task.proto
```

### Local Development

Run services locally (outside Kubernetes):
```bash
# Terminal 1 - PostgreSQL
docker run --rm --name postgres \
  -e POSTGRES_USER=taskboard \
  -e POSTGRES_PASSWORD=taskboard \
  -e POSTGRES_DB=taskboard \
  -p 5432:5432 postgres:15

# Terminal 2 - NATS
docker run --rm --name nats -p 4222:4222 nats:2.10

# Terminal 3 - Task Service
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=taskboard
export DB_PASSWORD=taskboard
export DB_NAME=taskboard
export NATS_URL=nats://localhost:4222
export PORT=50051

go run cmd/task-service/main.go

# Terminal 4 - API Gateway
export HTTP_PORT=8080
export TASK_SERVICE_ADDR=localhost:50051
export NATS_URL=nats://localhost:4222

go run cmd/api-gateway/main.go
```

### Running Tests
```bash
# Unit tests
go test ./...

# Integration tests
./scripts/test.sh

# Load testing
# Install hey: go install github.com/rakyll/hey@latest
hey -n 1000 -c 10 -m POST \
  -H "Content-Type: application/json" \
  -d '{"board_id":1,"title":"Load test","created_by":100}' \
  http://localhost:8080/api/tasks
```

---

## 🐛 Troubleshooting

### Cluster Issues

**Problem: Nodes not ready**
```bash
kubectl get nodes
# Shows: NotReady

# Check Talos health
talosctl health --nodes 10.5.0.2

# Check kubelet logs
talosctl logs --nodes 10.5.0.2 kubelet
```

**Problem: Pods not scheduling**
```bash
kubectl describe pod <pod-name> -n taskboard

# Common issue: Control plane taint
kubectl taint nodes --all node-role.kubernetes.io/control-plane:NoSchedule-
```

### Image Pull Issues

**Problem: ImagePullBackOff**
```bash
kubectl describe pod <pod-name> -n taskboard

# Check which registry IP is being used
kubectl get deployment <name> -n taskboard -o jsonpath='{.spec.template.spec.containers[0].image}'

# Should show: 10.5.0.x:5000/...
# NOT: 172.17.0.2:5000/...

# Fix: Use correct registry IP
export REGISTRY_IP=$(docker inspect registry -f '{{range $net, $config := .NetworkSettings.Networks}}{{if eq $net "taskboard"}}{{$config.IPAddress}}{{end}}{{end}}')
helmfile sync
```

### WebSocket Issues

**Problem: WebSocket won't connect**
```bash
# Check endpoint exists
curl -i http://localhost:8080/ws
# Should return: HTTP/1.1 426 Upgrade Required

# Check logs
kubectl logs -n taskboard -l app=api-gateway | grep -i websocket

# Should see:
# ✅ WebSocket Hub started
# ✅ WebSocket endpoint registered
```

**Problem: Events not appearing**
```bash
# Check NATS connection
kubectl logs -n taskboard -l app=api-gateway | grep NATS
# Should see: Connected to NATS successfully

kubectl logs -n taskboard -l app=task-service | grep NATS
# Should see: Connected to NATS successfully

# Check event publishing
kubectl logs -n taskboard -l app=task-service | grep "📤"
# Should see: 📤 Published event: tasks.created

# Check event reception
kubectl logs -n taskboard -l app=api-gateway | grep "📨"
# Should see: 📨 Received NATS event on tasks.created
```

### Database Issues

**Problem: Connection refused**
```bash
# Check PostgreSQL is running
kubectl get pods -n taskboard -l app=taskboard-db

# Check service
kubectl get svc -n taskboard taskboard-db-postgres

# Test connection from task-service pod
kubectl exec -it -n taskboard deploy/task-service -- sh
nc -zv taskboard-db-postgres 5432
```

### Complete Restart

If all else fails:
```bash
# Destroy everything
./scripts/destroy.sh

# Wait for cleanup
sleep 10

# Start fresh
./scripts/create-cluster.sh
./scripts/deploy.sh
```

---

## 📊 Monitoring

### Check System Health
```bash
# All pods running?
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
# API Gateway
kubectl logs -n taskboard -l app=api-gateway -f

# Task Service
kubectl logs -n taskboard -l app=task-service -f

# PostgreSQL
kubectl logs -n taskboard -l app=taskboard-db -f

# NATS
kubectl logs -n taskboard nats-0 -f
```

### Metrics

Check connected WebSocket clients:
```bash
kubectl logs -n taskboard -l app=api-gateway | grep "Total clients"
```

Check event throughput:
```bash
kubectl logs -n taskboard -l app=task-service | grep "📤" | wc -l
```

---

## 🎓 Learning Resources

### Concepts to Understand

1. **Kubernetes Fundamentals**
   - Pods, Deployments, Services
   - ConfigMaps, Secrets
   - Namespaces, Labels, Selectors

2. **Microservices Patterns**
   - API Gateway pattern
   - Event-driven architecture
   - Pub/Sub messaging

3. **Real-Time Systems**
   - WebSocket protocol
   - Connection lifecycle
   - Keepalive mechanisms

4. **gRPC**
   - Protocol Buffers
   - Service definitions
   - Streaming

5. **Container Orchestration**
   - Helm charts
   - Helmfile
   - Infrastructure as Code

### Next Steps

**Enhance the system:**
- [ ] Add authentication (JWT)
- [ ] Implement boards CRUD
- [ ] Add user management
- [ ] Implement task assignment
- [ ] Add file attachments
- [ ] Implement search
- [ ] Add notifications
- [ ] Implement audit logs

**Production readiness:**
- [ ] Add Prometheus metrics
- [ ] Implement distributed tracing (Jaeger)
- [ ] Set up log aggregation (Loki)
- [ ] Add health checks
- [ ] Implement rate limiting
- [ ] Add circuit breakers
- [ ] Set up backup/restore
- [ ] Implement disaster recovery

---

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 👥 Authors

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

