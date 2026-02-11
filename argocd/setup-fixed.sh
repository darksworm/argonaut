#!/bin/bash
# Fixed ArgoCD setup script - handles errors and uses correct paths

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "🚀 Setting up lightweight ArgoCD demo..."

# 1) Check if cluster exists, create if not
if k3d cluster list | grep -q argocd-demo; then
    echo "Cluster argocd-demo already exists, using it..."
    kubectl config use-context k3d-argocd-demo
else
    echo "Creating single k3d cluster..."
    k3d cluster create argocd-demo \
      --servers 1 \
      --agents 0 \
      --k3s-arg "--disable=traefik@server:0" \
      --k3s-arg "--disable=metrics-server@server:0"
fi

# 2) Install or verify ArgoCD
if kubectl get ns argocd &>/dev/null; then
    echo "ArgoCD namespace exists, checking installation..."
else
    echo "Installing ArgoCD..."
    kubectl create ns argocd
fi

# Always apply to ensure latest version
# Use --server-side to avoid the 262KB annotation limit on large CRDs (applicationsets.argoproj.io)
kubectl -n argocd apply --server-side --force-conflicts -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 3) Wait for ArgoCD to be ready
echo "Waiting for ArgoCD to be ready..."
kubectl -n argocd rollout status deploy/argocd-server --timeout=300s

# 4) Port-forward (kill existing if any)
echo "Setting up port-forward..."
pkill -f "port-forward.*argocd-server" || true
sleep 2
kubectl -n argocd port-forward svc/argocd-server 8080:443 >/dev/null 2>&1 &
PF_PID=$!
echo "Port-forward PID: $PF_PID"
sleep 3

# 5) Get admin password and login
PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)
echo "Admin password: $PASS"

# 6) Login with argocd CLI
echo "Logging into ArgoCD..."
argocd login localhost:8080 --username admin --password "$PASS" --insecure --grpc-web

# 7) Apply projects if they exist
if [ -f "$SCRIPT_DIR/projects.yaml" ]; then
    echo "Creating projects..."
    kubectl apply -f "$SCRIPT_DIR/projects.yaml"
else
    echo "No projects.yaml found, using default project"
fi

# 8) Apply the working apps configuration
echo "Creating applications..."
kubectl apply -f "$SCRIPT_DIR/apps-working.yaml"

echo ""
echo "✅ Setup complete!"
echo ""
echo "ArgoCD UI: https://localhost:8080"
echo "Username: admin"
echo "Password: $PASS"
echo ""
echo "Apps created (OutOfSync - manual sync required):"
argocd app list --output name
echo ""
echo "To sync all apps: argocd app sync --all"
echo "To sync one app: argocd app sync test-dev"
echo "To list apps: argocd app list"
echo ""
echo "To cleanup everything: k3d cluster delete argocd-demo"