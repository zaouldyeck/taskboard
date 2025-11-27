#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo -e "${RED}🗑️  Destroying Taskboard deployment...${NC}"

# Set KUBECONFIG to default location
export KUBECONFIG="${HOME}/.kube/config"

# Parse command line arguments
KEEP_REGISTRY=false
KEEP_CLUSTER=false

for arg in "$@"; do
    case $arg in
        --keep-registry)
            KEEP_REGISTRY=true
            shift
            ;;
        --keep-cluster)
            KEEP_CLUSTER=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --keep-registry    Keep the Docker registry running"
            echo "  --keep-cluster     Keep the Talos cluster running"
            echo "  --help            Show this help message"
            exit 0
            ;;
    esac
done

# Step 1: Destroy Helmfile deployments
echo -e "\n${YELLOW}📦 Destroying Helm releases...${NC}"

if kubectl get nodes &>/dev/null 2>&1; then
    echo "  - Cluster is reachable"
    if helmfile list 2>/dev/null | grep -q "taskboard"; then
        timeout 60s helmfile destroy || echo "  ⚠️  Helmfile destroy timed out"
        echo "✅ Helm releases destroyed (db, task-service, api-gateway)"
    else
        echo "  (no releases found)"
    fi
else
    echo "  ⚠️  Cluster unreachable, skipping Helm cleanup"
fi

# If keeping cluster, stop here
if [ "$KEEP_CLUSTER" = true ]; then
    echo -e "\n${GREEN}✅ Cleanup complete (cluster preserved)${NC}"
    exit 0
fi

# Step 2: Destroy Talos cluster
echo -e "\n${YELLOW}☸️  Destroying Talos cluster...${NC}"
if talosctl cluster show --provisioner docker 2>/dev/null | grep -q "taskboard"; then
    talosctl cluster destroy --name taskboard --provisioner docker
    echo "✅ Talos cluster destroyed"
else
    echo "  (no cluster found)"
fi

# Step 3: Clean up Docker resources
echo -e "\n${YELLOW}🧹 Cleaning up Docker resources...${NC}"

# Remove containers
docker ps -a --format '{{.Names}}' | grep -E "(talos.*taskboard|taskboard.*controlplane|taskboard.*worker)" | xargs -r docker rm -f 2>/dev/null || true

# Remove network
if docker network ls --format '{{.Name}}' | grep -q "^taskboard$"; then
    docker network rm taskboard 2>/dev/null || true
fi

echo "✅ Docker resources cleaned"

# Step 4: Clean up contexts
echo -e "\n${YELLOW}🧹 Cleaning up contexts...${NC}"

# Remove numbered contexts (from failed attempts)
for i in {1..20}; do
    talosctl config remove taskboard-$i 2>/dev/null || true
    kubectl config delete-context admin@taskboard-$i 2>/dev/null || true
    kubectl config delete-cluster taskboard-$i 2>/dev/null || true
    kubectl config delete-user admin@taskboard-$i 2>/dev/null || true
done

# Remove main context
talosctl config remove taskboard 2>/dev/null || true
kubectl config delete-context admin@taskboard 2>/dev/null || true
kubectl config delete-cluster taskboard 2>/dev/null || true
kubectl config delete-user admin@taskboard 2>/dev/null || true

# Remove state directory
rm -rf ~/.talos/clusters/taskboard

echo "✅ Contexts cleaned"

# Step 5: Handle registry
if [ "$KEEP_REGISTRY" = true ]; then
    echo -e "\n${GREEN}📦 Keeping Docker registry${NC}"
else
    echo -e "\n${YELLOW}🐳 Removing Docker registry...${NC}"
    if docker ps -a --format '{{.Names}}' | grep -q "^registry$"; then
        docker stop registry 2>/dev/null || true
        docker rm registry 2>/dev/null || true
        echo "✅ Registry removed"
    else
        echo "  (no registry found)"
    fi
fi

# Step 6: Check for orphaned volumes
echo -e "\n${YELLOW}🗂️  Checking for orphaned volumes...${NC}"
ORPHANED=$(docker volume ls -q -f dangling=true)
if [ -n "$ORPHANED" ]; then
    echo "  Found orphaned volumes:"
    echo "$ORPHANED" | sed 's/^/    /'
    read -p "  Remove them? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "$ORPHANED" | xargs docker volume rm
        echo "  ✅ Volumes removed"
    fi
else
    echo "  (no orphaned volumes)"
fi

echo -e "\n${GREEN}✅ Cleanup complete!${NC}"
echo ""
echo -e "${YELLOW}📝 Summary:${NC}"
echo "  - Helm releases: destroyed"
echo "    • PostgreSQL database"
echo "    • Task service (gRPC)"
echo "    • API Gateway (HTTP)"
if [ "$KEEP_CLUSTER" = false ]; then
    echo "  - Talos cluster: destroyed"
    echo "  - Docker resources: removed"
    echo "  - Contexts: cleaned"
fi
if [ "$KEEP_REGISTRY" = true ]; then
    echo "  - Docker registry: preserved"
else
    echo "  - Docker registry: removed"
fi
echo ""
echo -e "${YELLOW}💡 Next steps:${NC}"
echo "  To redeploy: ./scripts/deploy.sh"
if [ "$KEEP_REGISTRY" = true ]; then
    echo "  To remove registry: docker stop registry && docker rm registry"
fi
