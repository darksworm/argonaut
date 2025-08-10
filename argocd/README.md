# Argo CD multi‑cluster demo (k3d)

Spin up Argo CD in a **mgmt** cluster and manage two workload clusters — **cluster-magic** (Harry Potter) and **cluster-scifi** (Hitchhiker’s Guide) — all inside Docker via **k3d**. This version creates **4 projects**, **8 namespaces** (dev/staging/qa/prod per theme), and **10 apps per project** (40 apps total).

---

## Stack

* Docker Desktop
* kubectl
* k3d (K3s in Docker)
* Argo CD CLI (`argocd`)

> Tested on macOS. Works similarly on Linux/Windows; networking note below.

---

## Quick start (copy/paste)

```bash
# 0) Install deps (brew shown)
brew install k3d argocd kubernetes-cli

# 1) Create clusters (1 server + 1 agent each)
for c in mgmt eu us; do k3d cluster create "$c" --agents 1 --servers 1; done

# 2) Install Argo CD into mgmt
kubectl --context k3d-mgmt create ns argocd
kubectl --context k3d-mgmt -n argocd apply -f \
  https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl --context k3d-mgmt -n argocd rollout status deploy/argocd-server

# 3) Port-forward & login
kubectl --context k3d-mgmt -n argocd port-forward svc/argocd-server 8080:443 >/dev/null 2>&1 &
PASS=$(kubectl --context k3d-mgmt -n argocd get secret argocd-initial-admin-secret \
  -o jsonpath='{.data.password}' | base64 -d)
argocd login localhost:8080 --username admin --password "$PASS" --insecure --grpc-web

# 4) Register workload clusters with friendly names
argocd cluster add k3d-eu --name cluster-magic --yes
argocd cluster add k3d-us --name cluster-scifi --yes
argocd cluster list

# 5) (Optional but often needed) Patch Argo CD’s stored server URLs so pods can reach them
#    Argo CD pods can reach your host via host.k3d.internal inside Docker's network.
for NAME in cluster-magic cluster-scifi; do
  CTX=$( [ "$NAME" = cluster-magic ] && echo k3d-eu || echo k3d-us )
  PORT=$(kubectl config view --raw -o jsonpath="{.clusters[?(@.name=='$CTX')].cluster.server}" | sed -E 's#.*:([0-9]+)$#\1#')
  SEC=$(kubectl --context k3d-mgmt -n argocd get secret \
      -l "argocd.argoproj.io/secret-type=cluster,cluster-name=$NAME" \
      -o jsonpath="{.items[0].metadata.name}")
  NEW=https://host.k3d.internal:$PORT
  kubectl --context k3d-mgmt -n argocd patch secret "$SEC" --type=json \
    -p="[{\"op\":\"replace\",\"path\":\"/data/server\",\"value\":\"$(echo -n $NEW | base64 -w0 2>/dev/null || echo -n $NEW | base64)\"}]"
done
kubectl --context k3d-mgmt -n argocd rollout restart deploy/argocd-repo-server deploy/argocd-server
kubectl --context k3d-mgmt -n argocd rollout restart statefulset/argocd-application-controller

# 6) Create namespaces on the workload clusters (4 per theme)
# cluster-magic (k3d-eu)
for ns in hogwarts-{dev,staging,qa,prod} diagon-{dev,staging,qa,prod}; do \
  kubectl --context k3d-eu create ns $ns; done
# cluster-scifi (k3d-us)
for ns in hitchhiker-{dev,staging,qa,prod} megadodo-{dev,staging,qa,prod}; do \
  kubectl --context k3d-us create ns $ns; done

# 7) add apps
kubectl --context k3d-mgmt apply -f projects.yaml

# 8) Seed many apps via one ApplicationSet (10 per project = 40 apps)
kubectl --context k3d-mgmt apply -f apps.yaml

# 9) Verify
argocd app list
argocd cluster list
```

Open the UI at [https://localhost:8080](https://localhost:8080) (ignore TLS warning if you used `--insecure`). You should see **2 clusters**, **4 projects**, **8 namespaces**, and **40 apps** syncing across dev/staging/qa/prod.

---

## Networking notes (why step 5 exists)

* `k3d` exposes each cluster’s API on your **host** as `https://0.0.0.0:<random-port>`.
* Argo CD runs **inside** `k3d-mgmt` (a container) and cannot reach `0.0.0.0:<port>`.
* Inside Docker/k3d networks the host is reachable as `host.k3d.internal`. Step 5 replaces Argo CD’s stored server URL so **pods** can talk to the workload clusters.
* **Do not** rewrite your host kubeconfig to `host.k3d.internal`; your host can’t resolve it. Keep host kubeconfig as-is; patch **only** the Argo CD cluster secrets.

On Docker Desktop for macOS/Windows, `host.docker.internal` can also work, but `host.k3d.internal` is consistent for k3d.

---

## Troubleshooting

**`connect: connection refused` / `0.0.0.0:PORT` in Argo CD cluster details**

* You forgot step 5. Patch secrets to `https://host.k3d.internal:PORT` and restart Argo CD components.

**`permission denied` when `argocd cluster rm`**

* Log in as `admin`, or delete the secret directly:

  ```bash
  kubectl --context k3d-mgmt -n argocd delete secret <cluster-secret-name>
  ```

**`host.k3d.internal: no such host` from your host shell**

* Expected. Your host can’t resolve that name. Only containers/pods can.

**List cluster secrets**

```bash
kubectl --context k3d-mgmt -n argocd get secret -l argocd.argoproj.io/secret-type=cluster -o name
```

**Show friendly names + servers Argo CD stored**

```bash
kubectl --context k3d-mgmt -n argocd get secret -l argocd.argoproj.io/secret-type=cluster \
  -o jsonpath='{range .items[*]}{.data.name|base64decode}{" -> "}{.data.server|base64decode}{"\n"}{end}'
```

---

## Cleanup

```bash
k3d cluster delete mgmt eu us
rm -f projects.yaml apps.yaml
```

---

## Extending

* Add more themed apps by appending elements under `generators.list.elements`.
* Swap `source` to your own repos/paths or Helm charts.
* If you want a **lot** of pretend clusters without networking tweaks, run multiple **vcluster** instances inside `k3d-mgmt` and add them to Argo CD by name.
