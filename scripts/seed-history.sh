#!/usr/bin/env bash
# Seed an Argo CD app with a real deployment history for testing the
# rollback view: a local git repo (served to the k3d cluster by the git
# daemon from `make argocd-git-daemon`) gets a chain of commits whose
# manifests genuinely differ, and the app is synced at each revision so
# every history entry has a real sha, commit metadata, and a real diff.
set -euo pipefail

APP=${HISTORY_APP:-history-demo}
DEPTH=${HISTORY_DEPTH:-6}
NAMESPACE=history-demo

# The git daemon serves the parent directory of the argonaut repo.
BASE_DIR=$(cd "$(dirname "$0")/../.." && pwd)
REPO_NAME=argonaut-history-repo
REPO_DIR="$BASE_DIR/$REPO_NAME"
REPO_URL="git://host.k3d.internal/$REPO_NAME"

# Fresh repo every run so revisions and history entries line up.
rm -rf "$REPO_DIR"
mkdir -p "$REPO_DIR/manifests"
cd "$REPO_DIR"
git init -q -b main

subject() {
  case $(( $1 % 6 )) in
    0) echo "feat: initial web deployment" ;;
    1) echo "feat: expose web through a service" ;;
    2) echo "fix: scale up after load spike (#$((100 + $1)))" ;;
    3) echo "chore: rotate motd for release r$1" ;;
    4) echo "feat: enable feature banner" ;;
    5) echo "refactor: tune replica count down" ;;
  esac
}

write_manifests() {
  local i=$1
  cat > manifests/deployment.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  labels:
    app: web
    release: "r$i"
spec:
  replicas: $(( i % 3 + 1 ))
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
        release: "r$i"
    spec:
      containers:
        - name: web
          image: nginx:1.27-alpine
          env:
            - name: RELEASE
              value: "r$i"
            - name: FEATURE_BANNER
              value: "$([ $(( i % 2 )) -eq 1 ] && echo on || echo off)"
EOF
  cat > manifests/configmap.yaml <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: web-config
data:
  release: "r$i"
  motd: "Welcome to release r$i"
EOF
  # A resource that appears mid-history, so older entries diff as a removal
  if [ "$i" -ge 2 ]; then
    cat > manifests/service.yaml <<EOF
apiVersion: v1
kind: Service
metadata:
  name: web
spec:
  selector:
    app: web
  ports:
    - port: 80
      targetPort: 80
EOF
  fi
}

for i in $(seq 0 $(( DEPTH - 1 ))); do
  write_manifests "$i"
  git add -A
  git commit -q -m "$(subject "$i")" -m "Release r$i of the demo web stack.

Signed-off-by: Argonaut Demo <demo@argonaut.local>"
done

echo "Seeded $REPO_DIR with $DEPTH commits."

argocd app create "$APP" \
  --repo "$REPO_URL" \
  --path manifests \
  --dest-server https://kubernetes.default.svc \
  --dest-namespace "$NAMESPACE" \
  --sync-option CreateNamespace=true \
  --project default \
  --revision-history-limit 100 \
  --upsert

for rev in $(git log --format=%H | tac); do
  echo "Syncing $APP @ $rev ..."
  argocd app sync "$APP" --revision "$rev" --prune >/dev/null
done

echo "Done — '$APP' has $DEPTH history entries with real diffs."
echo "Tip: 'argocd app set $APP --sync-policy automated' to test the auto-sync warning."
