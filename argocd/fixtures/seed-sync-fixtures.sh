#!/usr/bin/env bash
# Seed three Argo CD apps for testing sync options, using the same mechanism as
# scripts/seed-history.sh: a local git repo served to the k3d cluster by the git
# daemon from `make argocd-git-daemon`.
#
# The prune fixtures work by making the manifest set shrink between two commits.
# The apps are synced at the FIRST commit (which contains an extra ConfigMap),
# while their targetRevision tracks `main` (which no longer has it). That leaves
# a live resource with no git counterpart — a genuine prune candidate.
set -euo pipefail

FIXTURE_DIR=$(cd "$(dirname "$0")" && pwd)
# The git daemon serves the parent directory of the argonaut repo. When this
# script lives inside the checkout that is derivable; otherwise set the env var.
BASE_DIR=${GIT_DAEMON_BASE_PATH:-}
if [ -z "$BASE_DIR" ]; then
  TOPLEVEL=$(git -C "$FIXTURE_DIR" rev-parse --show-toplevel 2>/dev/null || true)
  [ -n "$TOPLEVEL" ] && BASE_DIR=$(dirname "$TOPLEVEL")
fi
if [ -z "$BASE_DIR" ] || [ ! -d "$BASE_DIR" ]; then
  echo "Cannot locate the git daemon base path. Set GIT_DAEMON_BASE_PATH to the" >&2
  echo "directory the daemon serves (the parent of the argonaut checkout)." >&2
  exit 1
fi

# Argo CD reaches the host's git daemon by name only when k3d registered
# host.k3d.internal in CoreDNS. Older clusters have no such entry, so fall back
# to the docker network gateway, which is the host as seen from inside.
GIT_HOST=${GIT_HOST:-}
if [ -z "$GIT_HOST" ]; then
  if kubectl -n kube-system get cm coredns -o jsonpath='{.data.NodeHosts}' 2>/dev/null | grep -q host.k3d.internal; then
    GIT_HOST=host.k3d.internal
  else
    GIT_HOST=$(docker network inspect "k3d-${K3D_CLUSTER:-argocd-demo}" \
      --format '{{range .IPAM.Config}}{{.Gateway}}{{end}}' 2>/dev/null)
  fi
fi
if [ -z "$GIT_HOST" ]; then
  echo "Cannot work out how the cluster reaches this host; set GIT_HOST." >&2
  exit 1
fi
echo "Cluster will clone from git://$GIT_HOST/ ..."

REPO_NAME=argonaut-sync-fixtures-repo
REPO_DIR="$BASE_DIR/$REPO_NAME"
GIT_DAEMON_PORT=${GIT_DAEMON_PORT:-9418}

if ! argocd account get-user-info >/dev/null 2>&1; then
  echo "Argo CD is not reachable — run 'make argocd-up' first." >&2
  exit 1
fi
if ! bash -c "exec 3<>/dev/tcp/127.0.0.1/$GIT_DAEMON_PORT; exec 3<&-" 2>/dev/null; then
  echo "No git daemon on :$GIT_DAEMON_PORT — run 'make argocd-git-daemon' first." >&2
  exit 1
fi

APPS=(prune-demo prune-confirm-demo schema-error-demo)

# Fresh repo AND fresh apps every run: a previous run may already have pruned
# the orphan, and old apps would still reference commits this reset destroys.
for app in "${APPS[@]}"; do
  if argocd app get "$app" >/dev/null 2>&1; then
    echo "Deleting previous '$app' ..."
    argocd app delete "$app" --yes >/dev/null
  fi
done
for app in "${APPS[@]}"; do
  for _ in $(seq 1 60); do
    argocd app get "$app" >/dev/null 2>&1 || break
    sleep 1
  done
done

rm -rf "$REPO_DIR"
mkdir -p "$REPO_DIR"
cp -R "$FIXTURE_DIR/manifests/." "$REPO_DIR/"
cd "$REPO_DIR"
git init -q -b main
git config user.name "Argonaut Demo"
git config user.email "demo@argonaut.local"

git add -A
git commit -q -m "feat: initial fixture manifests"
ORPHAN_REV=$(git rev-parse HEAD)

git rm -q prune/configmap-orphan.yaml prune-confirm/configmap-orphan.yaml
git commit -q -m "chore: drop the orphan config maps"

echo "Seeded $REPO_DIR (orphans present at $ORPHAN_REV, gone at HEAD)."

sed "s|git://host.k3d.internal/|git://$GIT_HOST/|g" "$FIXTURE_DIR/apps-sync-options.yaml" \
  | kubectl apply -f -

# Sync the prune fixtures at the commit that still has the orphan, so it lands
# in the cluster. The apps then track `main` and go OutOfSync with a prunable
# resource. schema-error-demo is left unsynced on purpose.
for app in prune-demo prune-confirm-demo; do
  echo "Syncing $app @ $ORPHAN_REV ..."
  argocd app sync "$app" --revision "$ORPHAN_REV" >/dev/null
done

cat <<EOF

Done. Try:

  # 1) a prune with something real to delete
  argocd app sync prune-demo --prune

  # 2) a prune that gets stuck Running awaiting confirmation
  argocd app sync prune-confirm-demo --prune --timeout 60   # blocks
  argocd app get prune-confirm-demo                         # phase: Running
  argocd app confirm-deletion prune-confirm-demo            # releases it

  # 3) a dry-run sync that fails validation
  argocd app sync schema-error-demo --dry-run

Re-run this script to reset all three fixtures.
EOF
