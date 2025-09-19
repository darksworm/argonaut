#!/bin/bash
# Lightweight ArgoCD setup - single cluster, minimal resources

set -e

echo "🚀 Setting up lightweight ArgoCD demo..."

# 1) Create single k3d cluster with minimal resources
echo "Creating single k3d cluster..."
k3d cluster create argocd-demo \
  --servers 1 \
  --agents 0 \
  --k3s-arg "--disable=traefik@server:0" \
  --k3s-arg "--disable=metrics-server@server:0"

# 2) Install ArgoCD
echo "Installing ArgoCD..."
kubectl create ns argocd
kubectl -n argocd apply -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 3) Wait for ArgoCD to be ready
echo "Waiting for ArgoCD to be ready..."
kubectl -n argocd rollout status deploy/argocd-server --timeout=300s

# 4) Port-forward
echo "Starting port-forward..."
kubectl -n argocd port-forward svc/argocd-server 8080:443 >/dev/null 2>&1 &
PF_PID=$!
echo "Port-forward PID: $PF_PID"
sleep 3

# 5) Get admin password and login
PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)
echo "Admin password: $PASS"

# 6) Login with argocd CLI
argocd login localhost:8080 --username admin --password "$PASS" --insecure --grpc-web

# 7) Register the in-cluster as both "cluster-magic" and "cluster-scifi"
# (same cluster, different names for demo purposes)
kubectl config set-context --current --namespace=argocd
argocd cluster add k3d-argocd-demo --name cluster-magic --in-cluster --yes
argocd cluster add k3d-argocd-demo --name cluster-scifi --in-cluster --yes

# 8) Create namespaces
echo "Creating namespaces..."
for ns in hogwarts-{dev,staging,qa,prod} diagon-{dev,staging,qa,prod} \
          hitchhiker-{dev,staging,qa,prod} megadodo-{dev,staging,qa,prod}; do
  kubectl create ns $ns --dry-run=client -o yaml | kubectl apply -f -
done

# 9) Apply projects
echo "Creating projects..."
kubectl apply -f projects.yaml

# 10) Apply lightweight apps (using pause containers)
echo "Creating apps with pause containers..."
cat > /tmp/apps-ultra-light.yaml << 'EOF'
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: ultra-light-demo
  namespace: argocd
spec:
  generators:
  - list:
      elements:
      # Minimal set for testing - 8 apps total
      - {project: hogwarts, namespace: hogwarts-dev, service: owlery}
      - {project: hogwarts, namespace: hogwarts-prod, service: spellbook}
      - {project: diagon, namespace: diagon-dev, service: ollivanders}
      - {project: diagon, namespace: diagon-prod, service: gringotts}
      - {project: hitchhiker, namespace: hitchhiker-dev, service: heart-of-gold}
      - {project: hitchhiker, namespace: hitchhiker-prod, service: babel-fish}
      - {project: megadodo, namespace: megadodo-dev, service: guide}
      - {project: megadodo, namespace: megadodo-prod, service: towel}
  template:
    metadata:
      name: '{{project}}-{{service}}'
    spec:
      project: '{{project}}'
      syncPolicy:
        automated: null  # Manual sync only
      source:
        repoURL: https://github.com/argoproj/argocd-example-apps
        path: .
        targetRevision: HEAD
        directory:
          recurse: false
        # Override with inline manifests
        kustomize:
          patches:
          - patch: |-
              apiVersion: v1
              kind: ConfigMap
              metadata:
                name: {{service}}-config
              data:
                app: {{service}}
                project: {{project}}
            target:
              kind: ConfigMap
              name: '.*'
          - patch: |-
              apiVersion: apps/v1
              kind: Deployment
              metadata:
                name: {{service}}
              spec:
                replicas: 1
                selector:
                  matchLabels:
                    app: {{service}}
                template:
                  metadata:
                    labels:
                      app: {{service}}
                  spec:
                    containers:
                    - name: app
                      image: gcr.io/google_containers/pause:3.9
                      resources:
                        limits:
                          cpu: "2m"
                          memory: "2Mi"
            target:
              kind: Deployment
              name: '.*'
      destination:
        name: cluster-magic  # All apps go to same cluster
        namespace: '{{namespace}}'
EOF

kubectl apply -f /tmp/apps-ultra-light.yaml

echo "✅ Setup complete!"
echo ""
echo "ArgoCD UI: https://localhost:8080"
echo "Username: admin"
echo "Password: $PASS"
echo ""
echo "To sync all apps: argocd app sync -l argocd.argoproj.io/instance"
echo "To list apps: argocd app list"
echo ""
echo "To cleanup: k3d cluster delete argocd-demo"