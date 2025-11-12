#!/bin/bash
set -e

echo "🚀 Deploying Taskboard..."

# Get registry IP
export REGISTRY_IP=$(docker inspect registry -f '{{.NetworkSettings.Networks.taskboard.IPAddress}}')
echo "Registry IP: $REGISTRY_IP"

# Build and push image
echo "📦 Building image..."
docker build -t taskboard-task-service:scratch -f cmd/task-service/Dockerfile .
docker tag taskboard-task-service:scratch localhost:5010/taskboard-task-service:scratch
docker push localhost:5010/taskboard-task-service:scratch

# Deploy with Helmfile
echo "☸️  Deploying to Kubernetes..."
helmfile sync

echo "✅ Deployment complete!"
echo ""
echo "📊 Status:"
kubectl get pods -n taskboard
echo ""
echo "🔌 To access the service:"
echo "  kubectl port-forward -n taskboard svc/taskboard-task-service 50051:50051"
