#!/bin/bash
set -e

# Ensure we're in the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Deploying Taskboard...${NC}"

# Set KUBECONFIG to default location
export KUBECONFIG="${HOME}/.kube/config"

# Step 1: Check if Talos cluster exists
echo -e "\n${YELLOW}📋 Checking Talos cluster status...${NC}"

if talosctl cluster show --provisioner docker 2>/dev/null | grep -q "taskboard"; then
    echo "✅ Talos cluster 'taskboard' exists"
    talosctl kubeconfig --nodes 10.5.0.2 --force
else
    # Clean up any existing resources first
    echo -e "\n${YELLOW}🧹 Cleaning up any existing resources...${NC}"
    docker stop taskboard-controlplane-1 2>/dev/null || true
    docker rm taskboard-controlplane-1 2>/dev/null || true
    docker network rm taskboard 2>/dev/null || true
    
    # Clean up contexts
    for i in {1..20}; do
        talosctl config remove taskboard-$i 2>/dev/null || true
        kubectl config delete-context admin@taskboard-$i 2>/dev/null || true
        kubectl config delete-cluster taskboard-$i 2>/dev/null || true
        kubectl config delete-user admin@taskboard-$i 2>/dev/null || true
    done
    talosctl config remove taskboard 2>/dev/null || true
    kubectl config delete-context admin@taskboard 2>/dev/null || true
    kubectl config delete-cluster taskboard 2>/dev/null || true
    kubectl config delete-user admin@taskboard 2>/dev/null || true
    
    rm -rf ~/.talos/clusters/taskboard
    echo "✅ Cleanup complete"
    
    # Create cluster
    echo -e "\n${YELLOW}🔧 Creating Talos cluster...${NC}"
    talosctl cluster create \
        --provisioner docker \
        --name taskboard \
        --workers 0 \
        --controlplanes 1 \
        --wait \
        --wait-timeout 10m || {
            echo -e "${RED}❌ Failed to create cluster${NC}"
            exit 1
        }
    
    echo "✅ Talos cluster created"
fi

# Step 2: Configure kubectl access
echo -e "\n${YELLOW}🔧 Configuring cluster access...${NC}"

# Get fresh kubeconfig
talosctl kubeconfig --nodes 10.5.0.2 --force

# Fix server endpoint to use forwarded port
CONTROL_PLANE_IP="10.5.0.2"
FORWARDED_PORT=$(docker port taskboard-controlplane-1 6443/tcp 2>/dev/null | cut -d: -f2)

if [ -n "$FORWARDED_PORT" ]; then
    kubectl config set-cluster taskboard --server=https://127.0.0.1:${FORWARDED_PORT}
    echo "  - API server: https://127.0.0.1:${FORWARDED_PORT}"
else
    echo -e "${RED}❌ Could not detect forwarded port${NC}"
    exit 1
fi

# Wait for cluster to be ready
echo "⏳ Waiting for cluster to be ready..."
MAX_RETRIES=60
RETRY_COUNT=0
while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if kubectl get nodes &>/dev/null; then
        kubectl wait --for=condition=ready node --all --timeout=120s && break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $((RETRY_COUNT % 5)) -eq 0 ]; then
        echo "  Attempt $RETRY_COUNT/$MAX_RETRIES..."
    fi
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}❌ Cluster not ready after $MAX_RETRIES attempts${NC}"
    exit 1
fi

echo "✅ Cluster is ready"

# Step 3: Configure cluster
echo -e "\n${YELLOW}⚙️  Configuring cluster...${NC}"

# Remove control-plane taint
kubectl taint nodes --all node-role.kubernetes.io/control-plane- 2>/dev/null || true

# Create and label taskboard namespace
kubectl create namespace taskboard 2>/dev/null || true
kubectl label namespace taskboard pod-security.kubernetes.io/enforce=privileged --overwrite

# Install storage provisioner if needed
if ! kubectl get storageclass local-path &>/dev/null; then
    echo "  - Installing storage provisioner..."
    kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.30/deploy/local-path-storage.yaml
    kubectl patch namespace local-path-storage -p '{"metadata":{"labels":{"pod-security.kubernetes.io/enforce":"privileged"}}}'
    kubectl wait --for=condition=ready pod -l app=local-path-provisioner -n local-path-storage --timeout=60s 2>/dev/null || true
fi

echo "✅ Cluster configured"

# Step 4: Setup Docker registry
echo -e "\n${YELLOW}🐳 Setting up Docker registry...${NC}"

# Create registry if not exists
if ! docker ps | grep -q "^.*registry.*$"; then
    if docker ps -a --format '{{.Names}}' | grep -q "^registry$"; then
        docker start registry
    else
        docker run -d -p 5010:5000 --restart=always --name registry registry:2
    fi
fi

# Create and connect networks
docker network create taskboard 2>/dev/null || true
docker network connect taskboard registry 2>/dev/null || true

# Get registry IPs
REGISTRY_IPS=$(docker inspect registry | jq -r '.[0].NetworkSettings.Networks[].IPAddress' | grep -v '^$' | sort -u)
export REGISTRY_IP=$(echo "$REGISTRY_IPS" | head -1)

echo "✅ Registry ready at ${REGISTRY_IP}:5000 (localhost:5010)"

# Step 5: Configure Talos for insecure registry
echo -e "\n${YELLOW}🔧 Configuring Talos for insecure registry...${NC}"
echo "  - Registry IPs: $REGISTRY_IPS"

# Create patch file
cat > /tmp/registry-patch.yaml << PATCH_EOF
machine:
  registries:
    config:
PATCH_EOF

for IP in $REGISTRY_IPS; do
    cat >> /tmp/registry-patch.yaml << PATCH_EOF
      "${IP}:5000":
        tls:
          insecureSkipVerify: true
PATCH_EOF
done

cat >> /tmp/registry-patch.yaml << PATCH_EOF
    mirrors:
PATCH_EOF

for IP in $REGISTRY_IPS; do
    cat >> /tmp/registry-patch.yaml << PATCH_EOF
      "${IP}:5000":
        endpoints:
          - "http://${IP}:5000"
PATCH_EOF
done

# Apply patch
talosctl --nodes $CONTROL_PLANE_IP patch machineconfig -p @/tmp/registry-patch.yaml
sleep 10

echo "✅ Registry configuration applied"

# Step 6: Build and push images
echo -e "\n${YELLOW}📦 Building and pushing images...${NC}"

echo "  - Building task-service..."
docker build -t taskboard-task-service:latest -f cmd/task-service/Dockerfile . || {
    echo -e "${RED}❌ Failed to build task-service${NC}"
    exit 1
}
docker tag taskboard-task-service:latest localhost:5010/taskboard-task-service:latest
docker push localhost:5010/taskboard-task-service:latest

echo "  - Building api-gateway..."
docker build -t taskboard-api-gateway:latest -f cmd/api-gateway/Dockerfile . || {
    echo -e "${RED}❌ Failed to build api-gateway${NC}"
    exit 1
}
docker tag taskboard-api-gateway:latest localhost:5010/taskboard-api-gateway:latest
docker push localhost:5010/taskboard-api-gateway:latest

echo "✅ Images pushed to registry"

# Step 7: Deploy with Helmfile
echo -e "\n${YELLOW}⎈ Deploying with Helmfile...${NC}"
helmfile sync || {
    echo -e "${RED}❌ Helmfile deployment failed${NC}"
    exit 1
}

# Step 8: Wait for pods
echo -e "\n${YELLOW}⏳ Waiting for pods to be ready...${NC}"
kubectl wait --for=condition=ready pod -l app=taskboard-db -n taskboard --timeout=120s 2>/dev/null || true
kubectl wait --for=condition=ready pod -l app=task-service -n taskboard --timeout=120s 2>/dev/null || true
kubectl wait --for=condition=ready pod -l app=api-gateway -n taskboard --timeout=120s 2>/dev/null || true

# Step 9: Display status
echo -e "\n${GREEN}✅ Deployment complete!${NC}"
echo ""
echo -e "${BLUE}📊 Status:${NC}"
kubectl get pods -n taskboard
echo ""
echo -e "${BLUE}🌐 Services:${NC}"
kubectl get svc -n taskboard
echo ""
echo -e "${GREEN}💡 To access the API Gateway:${NC}"
echo "  kubectl port-forward -n taskboard svc/api-gateway 8080:8080"
echo ""
echo -e "${GREEN}💡 Then test:${NC}"
echo "  curl http://localhost:8080/health"
echo "  curl http://localhost:8080/api/tasks?board_id=1"
echo ""
echo -e "${BLUE}📝 Useful commands:${NC}"
echo "  # View gateway logs"
echo "  kubectl logs -n taskboard -l app=api-gateway -f"
echo ""
echo "  # View task service logs"
echo "  kubectl logs -n taskboard -l app=task-service -f"
echo ""
echo "  # Check database"
echo "  kubectl exec -it -n taskboard taskboard-db-0 -- psql -U taskboard"
