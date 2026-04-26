
.PHONY: test unit e2e goldens \
	k3d-start k3d-stop k3d-restart k3d-delete \
	argocd-up argocd-down argocd-restart \
	argocd-portforward argocd-portforward-stop \
	argocd-git-daemon argocd-git-daemon-stop \
	argocd-password argocd-login

# Default parallelism for tests
PARALLEL ?= 8

# k3d/ArgoCD defaults
K3D_CLUSTER ?= argocd-demo
ARGOCD_NAMESPACE ?= argocd
ARGOCD_PORT ?= 8080
ARGOCD_HOST ?= localhost

# Local git daemon defaults — serves this repo to the k3d cluster via
# host.k3d.internal so ArgoCD can resolve it without any remote push.
# The daemon exposes all repos under GIT_DAEMON_BASE_PATH.
GIT_DAEMON_PORT ?= 9418
GIT_DAEMON_BASE_PATH ?= $(abspath $(CURDIR)/..)

# Run all tests, including e2e (unix-only), with parallelism and no cache.
test:
	go test -tags e2e ./... -v -count=1 -parallel $(PARALLEL)

# Run only unit tests (no e2e).
unit:
	go test ./... -v -count=1 -parallel $(PARALLEL)

# Run only e2e tests.
e2e:
	go test -tags e2e ./e2e -v -count=1 -parallel $(PARALLEL)

# Regenerate golden snapshots for app tests.
goldens:
	UPDATE_GOLDEN=1 go test ./cmd/app -run TestGolden_ -v

# --- k3d cluster management ---

# Start k3d cluster (create if missing)
k3d-start:
	@if k3d cluster list | grep -q $(K3D_CLUSTER); then \
		echo "Starting cluster $(K3D_CLUSTER)..."; \
		k3d cluster start $(K3D_CLUSTER); \
	else \
		echo "Cluster $(K3D_CLUSTER) not found. Creating minimal cluster..."; \
		k3d cluster create $(K3D_CLUSTER) \
		  --servers 1 \
		  --agents 0 \
		  --k3s-arg "--disable=traefik@server:0" \
		  --k3s-arg "--disable=metrics-server@server:0"; \
	fi

# Stop k3d cluster (no delete)
k3d-stop:
	@echo "Stopping cluster $(K3D_CLUSTER)..."
	@k3d cluster stop $(K3D_CLUSTER)

# Restart k3d cluster
k3d-restart: k3d-stop k3d-start
	@echo "Cluster $(K3D_CLUSTER) restarted."

# Delete k3d cluster (destroy)
k3d-delete:
	@echo "Deleting cluster $(K3D_CLUSTER)..."
	@k3d cluster delete $(K3D_CLUSTER)

# --- ArgoCD convenience targets ---

# Full setup: cluster + ArgoCD + apps + login + port-forward
# To test the rollouts demo against an unmerged branch, run `make
# argocd-git-daemon` first and edit argocd/apps-rollouts.yaml accordingly.
argocd-up:
	@./argocd/setup-fixed.sh
	@$(MAKE) --no-print-directory argocd-login

# Stop ArgoCD port-forward, git daemon, and cluster (non-destructive)
argocd-down: argocd-portforward-stop argocd-git-daemon-stop
	@$(MAKE) --no-print-directory k3d-stop

# Start a local `git daemon` serving this repo so ArgoCD in k3d can clone
# it via `git://host.k3d.internal/<repo>` — enables fully offline testing
# without pushing to GitHub. PID written to .argocd-gitdaemon.pid.
argocd-git-daemon:
	@if [ -f .argocd-gitdaemon.pid ] && kill -0 $$(cat .argocd-gitdaemon.pid) 2>/dev/null; then \
		echo "git daemon already running (PID $$(cat .argocd-gitdaemon.pid))."; \
		exit 0; \
	fi
	@# Free the port if a stale daemon is bound
	@fuser -k -TERM $(GIT_DAEMON_PORT)/tcp 2>/dev/null || true
	@rm -f .argocd-gitdaemon.pid
	@sleep 0.2
	@echo "Starting git daemon on :$(GIT_DAEMON_PORT) serving $(GIT_DAEMON_BASE_PATH) ..."
	@setsid -f git daemon --reuseaddr --listen=0.0.0.0 --port=$(GIT_DAEMON_PORT) \
		--base-path=$(GIT_DAEMON_BASE_PATH) --export-all --informative-errors \
		>/tmp/argocd-gitdaemon.log 2>&1 < /dev/null & \
		echo $$! > .argocd-gitdaemon.pid
	@# Wait until the daemon is accepting connections (bash required for /dev/tcp)
	@for i in $$(seq 1 40); do \
		if bash -c "exec 3<>/dev/tcp/127.0.0.1/$(GIT_DAEMON_PORT); exec 3<&-" 2>/dev/null; then \
			echo "git daemon ready (PID $$(cat .argocd-gitdaemon.pid))."; \
			exit 0; \
		fi; \
		sleep 0.25; \
	done; \
	echo "git daemon failed to start; see /tmp/argocd-gitdaemon.log" >&2; \
	exit 1

# Stop the local git daemon
argocd-git-daemon-stop:
	@if [ -f .argocd-gitdaemon.pid ]; then \
		PID=$$(cat .argocd-gitdaemon.pid); \
		echo "Stopping git daemon PID $$PID..."; \
		kill $$PID 2>/dev/null || true; \
		rm -f .argocd-gitdaemon.pid; \
	else \
		echo "No git daemon PID file; freeing port $(GIT_DAEMON_PORT) if bound..."; \
		fuser -k -TERM $(GIT_DAEMON_PORT)/tcp 2>/dev/null || true; \
	fi

# Restart ArgoCD environment (stop, then full setup again)
argocd-restart:
	@$(MAKE) --no-print-directory argocd-down
	@$(MAKE) --no-print-directory argocd-up

# Start ArgoCD port-forward in background; writes PID to .argocd-portforward.pid.
# Uses setsid so the kubectl process is detached into its own session/process group
# and survives the make recipe's subshell exit. Waits until the port is accepting
# connections before returning, so dependents (argocd-login) don't race.
argocd-portforward:
	@# Clean up any existing port-forward by PID file. Intentionally do NOT
	@# use `pkill -f <pattern>` here: make runs each recipe line via `sh -c "..."`,
	@# so the parent shell's cmdline contains the pkill pattern literally and
	@# pkill ends up killing its own parent (visible as "Terminated" on the recipe).
	@if [ -f .argocd-portforward.pid ]; then \
		kill -TERM $$(cat .argocd-portforward.pid) 2>/dev/null || true; \
		rm -f .argocd-portforward.pid; \
	fi
	@# If anything is still bound to the port (stale / untracked process), free it.
	@fuser -k -TERM $(ARGOCD_PORT)/tcp 2>/dev/null || true
	@sleep 0.3
	@echo "Port-forwarding ArgoCD on https://$(ARGOCD_HOST):$(ARGOCD_PORT) ..."
	@setsid -f kubectl -n $(ARGOCD_NAMESPACE) port-forward svc/argocd-server $(ARGOCD_PORT):443 \
		>/tmp/argocd-portforward.log 2>&1 < /dev/null & \
		echo $$! > .argocd-portforward.pid
	@# Wait up to ~15s for the local listener to accept connections
	@for i in $$(seq 1 60); do \
		if curl -sk -o /dev/null --max-time 1 https://$(ARGOCD_HOST):$(ARGOCD_PORT); then \
			echo "Port-forward ready (PID $$(cat .argocd-portforward.pid))."; \
			exit 0; \
		fi; \
		sleep 0.25; \
	done; \
	echo "Port-forward failed to become ready; see /tmp/argocd-portforward.log" >&2; \
	exit 1

# Stop ArgoCD port-forward (uses PID file; falls back to `fuser -k` on the port
# since `pkill -f` self-terminates when invoked from a make recipe — see the
# comment in argocd-portforward for details).
argocd-portforward-stop:
	@if [ -f .argocd-portforward.pid ]; then \
		PID=$$(cat .argocd-portforward.pid); \
		echo "Stopping port-forward PID $$PID..."; \
		kill $$PID 2>/dev/null || true; \
		rm -f .argocd-portforward.pid; \
	else \
		echo "No PID file; freeing port $(ARGOCD_PORT) if bound..."; \
		fuser -k -TERM $(ARGOCD_PORT)/tcp 2>/dev/null || true; \
	fi

# Echo the initial ArgoCD admin password (reads from k3d-managed cluster)
argocd-password:
	@PASS=$$(kubectl -n $(ARGOCD_NAMESPACE) get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' 2>/dev/null \
		| (base64 -d 2>/dev/null || base64 --decode 2>/dev/null || base64 -D 2>/dev/null)); \
	if [ -n "$$PASS" ]; then \
		echo "$$PASS"; \
	else \
		echo "Could not read ArgoCD admin password. Is the cluster up and ArgoCD installed?" >&2; \
		exit 1; \
	fi

# Log in to ArgoCD using the initial admin password from the cluster
# Ensures port-forward is running first
argocd-login: argocd-portforward
	@echo "Waiting for ArgoCD server rollout..."
	@kubectl -n $(ARGOCD_NAMESPACE) rollout status deploy/argocd-server --timeout=180s
	@PASS=$$(kubectl -n $(ARGOCD_NAMESPACE) get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' 2>/dev/null \
		| (base64 -d 2>/dev/null || base64 --decode 2>/dev/null || base64 -D 2>/dev/null)); \
	if [ -z "$$PASS" ]; then \
		echo "Could not read ArgoCD admin password. Is ArgoCD installed?" >&2; \
		exit 1; \
	fi; \
	echo "Logging into ArgoCD at $(ARGOCD_HOST):$(ARGOCD_PORT) ..."; \
	argocd login $(ARGOCD_HOST):$(ARGOCD_PORT) --username admin --password "$$PASS" --insecure --grpc-web >/dev/null && echo "Logged in as admin."
