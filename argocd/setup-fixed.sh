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

# 4) Get admin password (for display at the end)
PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || true)

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

# 9) Apply the app-of-apps configuration
echo "Creating app-of-apps..."
kubectl apply -f "$SCRIPT_DIR/app-of-apps.yaml"

echo ""
echo "✅ Setup complete!"
echo ""
echo "ArgoCD UI: https://localhost:8080 (start port-forward with: make argocd-portforward)"
echo "Username: admin"
echo "Password: $PASS"
echo ""
echo "Applications (OutOfSync - manual sync required):"
kubectl -n argocd get applications.argoproj.io -o name
echo ""
echo "To log in to argocd CLI:          make argocd-login"
echo "To sync all apps:                 argocd app sync --all"
echo "To cleanup everything:            k3d cluster delete argocd-demo"