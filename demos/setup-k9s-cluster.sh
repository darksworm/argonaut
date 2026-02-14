#!/usr/bin/env bash
# Creates a lightweight k3d cluster with dummy workloads matching the demo apps.
# The exported kubeconfig has its context renamed so that findK9sContext()
# auto-matches the ArgoCD cluster name "prod-us-east-1".
#
# Usage: ./setup-k9s-cluster.sh <cluster-name> <kubeconfig-output-path>

set -euo pipefail

CLUSTER="${1:?Usage: $0 <cluster-name> <kubeconfig-path>}"
KUBECONFIG_OUT="${2:?Usage: $0 <cluster-name> <kubeconfig-path>}"

# App name → namespace (must match buildDemoApps() in cmd/demo/main.go)
declare -A APPS=(
  [payment-api]=payments
  [user-service]=users
  [frontend-web]=frontend
  [cart-service]=cart
  [notification-worker]=notifications
  [config-server]=platform
  [redis-cache]=cache
  [ingress-controller]=ingress
)

# ArgoCD cluster name used in the demo data
TARGET_CONTEXT="prod-us-east-1"

echo "==> Ensuring k3d cluster '${CLUSTER}' exists..."
if k3d cluster list 2>/dev/null | grep -q "${CLUSTER}"; then
  echo "    Cluster already exists, starting if stopped..."
  k3d cluster start "${CLUSTER}" 2>/dev/null || true
else
  echo "    Creating cluster..."
  k3d cluster create "${CLUSTER}" \
    --servers 1 \
    --agents 0 \
    --k3s-arg "--disable=traefik@server:0" \
    --k3s-arg "--disable=metrics-server@server:0" \
    --no-lb
fi

echo "==> Waiting for node to be ready..."
kubectl --context "k3d-${CLUSTER}" wait --for=condition=Ready node --all --timeout=60s

echo "==> Creating namespaces and deployments..."
for APP in "${!APPS[@]}"; do
  NS="${APPS[$APP]}"
  kubectl --context "k3d-${CLUSTER}" create namespace "${NS}" 2>/dev/null || true
  kubectl --context "k3d-${CLUSTER}" -n "${NS}" create deployment "${APP}" \
    --image=nginx:alpine --replicas=1 2>/dev/null || true
  # Reset replicas to 1 in case a previous run scaled them up
  kubectl --context "k3d-${CLUSTER}" -n "${NS}" scale deployment/"${APP}" --replicas=1
done

echo "==> Waiting for all deployments to roll out..."
for APP in "${!APPS[@]}"; do
  NS="${APPS[$APP]}"
  kubectl --context "k3d-${CLUSTER}" -n "${NS}" rollout status deployment/"${APP}" --timeout=120s
done

echo "==> Exporting kubeconfig..."
k3d kubeconfig get "${CLUSTER}" > "${KUBECONFIG_OUT}"

# Rename context from k3d-<cluster> to match the ArgoCD cluster name
K3D_CONTEXT="k3d-${CLUSTER}"
echo "==> Renaming context '${K3D_CONTEXT}' → '${TARGET_CONTEXT}'..."
KUBECONFIG="${KUBECONFIG_OUT}" kubectl config rename-context "${K3D_CONTEXT}" "${TARGET_CONTEXT}"

echo "==> Done. Kubeconfig written to ${KUBECONFIG_OUT} with context '${TARGET_CONTEXT}'"
