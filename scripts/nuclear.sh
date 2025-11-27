#!/bin/bash
echo "☢️  NUCLEAR CLEANUP - Forcing everything down..."

# 1. Kill ALL Talos/taskboard containers immediately
echo "  - Force stopping containers..."
docker ps -a --format '{{.Names}}' | grep -E '(talos|taskboard)' | xargs -r docker rm -f

# 2. Remove ALL related networks
echo "  - Removing networks..."
docker network ls --format '{{.Name}}' | grep -E '(talos|taskboard)' | xargs -r docker network rm 2>/dev/null || true

# 3. Clean up ALL Talos contexts
echo "  - Cleaning Talos contexts..."
talosctl config contexts | awk 'NR>1 {print $2}' | xargs -r -I {} talosctl config remove {}

# 4. Clean up ALL kubectl contexts
echo "  - Cleaning kubectl contexts..."
for i in {1..20}; do
    kubectl config delete-context admin@taskboard-$i 2>/dev/null || true
    kubectl config delete-cluster taskboard-$i 2>/dev/null || true
    kubectl config delete-user admin@taskboard-$i 2>/dev/null || true
done
kubectl config delete-context admin@taskboard 2>/dev/null || true
kubectl config delete-cluster taskboard 2>/dev/null || true
kubectl config delete-user admin@taskboard 2>/dev/null || true

# 5. Remove ALL Talos state
echo "  - Removing Talos state..."
rm -rf ~/.talos/clusters/*

# 6. Clean Docker system
echo "  - Docker system prune..."
docker system prune -f

echo "✅ Nuclear cleanup complete!"
echo ""
echo "Verifying cleanup:"
docker ps -a | grep -E '(talos|taskboard)' && echo "⚠️  Still some containers!" || echo "✅ No containers"
docker network ls | grep -E '(talos|taskboard)' && echo "⚠️  Still some networks!" || echo "✅ No networks"
