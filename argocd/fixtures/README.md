# Sync-option fixtures

Three small Argo CD Applications for exercising sync options against the local
k3d Argo CD.

```bash
make argocd-up
make argocd-git-daemon
./argocd/fixtures/seed-sync-fixtures.sh
```

`seed-sync-fixtures.sh` reuses `scripts/seed-history.sh`'s mechanism: it builds a
throwaway git repo next to the argonaut checkout (`argonaut-sync-fixtures-repo`),
which the `git daemon` from `make argocd-git-daemon` exports, so the cluster
clones it as `git://host.k3d.internal/argonaut-sync-fixtures-repo` with no push
to any remote. Re-running the script deletes the old Applications with
`argocd app delete --yes`, which cascades to managed resources by default,
then rebuilds the repo and syncs the two prune apps. The schema-error app is
left unsynced. The script stops if an old Application remains after 60 seconds,
before rebuilding the repository or applying new Applications.

The script derives the daemon base path from `git rev-parse --show-toplevel`.
Outside a checkout, set `GIT_DAEMON_BASE_PATH`.

The real e2e harness accepts only loopback endpoints (`localhost`, `127.0.0.1`,
or `::1`) and honors the CLI context's `insecure` setting. Use the local context
created by `make argocd-login` for the demo's self-signed certificate.

Run the seed script's isolated regression tests without a cluster:

```bash
python3 -B -m unittest discover -s argocd/fixtures -p '*_test.py'
```

## The fixtures

| App | Namespace | What it demonstrates |
|---|---|---|
| `prune-demo` | `sync-prune-demo` | A live ConfigMap with no git counterpart — `sync --prune` really deletes something |
| `prune-confirm-demo` | `sync-confirm-demo` | `sync --prune` parks in `Running` awaiting confirmation |
| `schema-error-demo` | `sync-invalid-demo` | `sync --dry-run` fails with a validation message |

All three follow the existing conventions: `project: default`,
`destination.server: https://kubernetes.default.svc`, manual sync
(`syncPolicy.automated: null`), `CreateNamespace=true` — same shape as
`argocd/apps-hang.yaml`.

### 1. Something to prune

The seed repo has two commits. The first contains `keep-me` **and**
`prune-me`; the second deletes `prune-me`. The script syncs the app at the
**first** commit, but the Application's `targetRevision` is `main`. So
`prune-me` is live in the cluster and absent from the tracked revision — Argo CD
marks the app OutOfSync and offers it as a prune candidate.

```bash
argocd app sync prune-demo            # stays OutOfSync, prune skipped
argocd app sync prune-demo --prune    # deletes prune-me, app goes Synced
```

This "manifest set shrinks between commits" trick is the same one
`seed-history.sh` already uses for `manifests/service.yaml` (present only from
commit 2 onward), just run backwards.

### 2. Prune stuck awaiting confirmation

Same setup, plus `argocd.argoproj.io/sync-options: Prune=confirm` on the orphan
ConfigMap. A prune task has no target object, so Argo CD reads the sync option
off the **live** resource — which is why the annotation has to be present in the
commit the app is synced at, not added later.

```bash
argocd app sync prune-confirm-demo --prune --timeout 60   # blocks
argocd app get prune-confirm-demo                         # operation phase: Running
argocd app confirm-deletion prune-confirm-demo            # releases it
```

Notes:
- Without `--prune` the gate never triggers — the app just stays OutOfSync.
- Confirming is also possible from the UI ("Confirm Pruning") or by annotating
  the **Application** with `argocd.argoproj.io/deletion-approved: <RFC3339 ts>`.

### 3. Schema error

`manifests/schema-error/deployment.yaml` says `reploicas` instead of `replicas`.
Argo CD's dry-run apply hits the API server, which rejects it:

```
Deployment in version "v1" cannot be handled as a Deployment:
strict decoding error: unknown field "spec.reploicas"
```

```bash
argocd app sync schema-error-demo --dry-run   # SyncFailed, no cluster changes
argocd app sync schema-error-demo             # same failure, recorded in status
```

The script leaves this app unsynced so the first sync attempt is the failing one.

Use the **plain** sync when the failure has to be visible in argonaut: a real
sync definitely records `SyncFailed` in `status.operationState`, which is what
the TUI renders. Whether `--dry-run` persists an operation result there too, or
only prints to the CLI, was not verified — see below.

## Version requirements

- **`Prune=confirm` (fixture 2) needs Argo CD ≥ 2.14**, not 3.1 as the task
  stated — upstream introduced it in the 2.14 release. This box currently runs
  **v3.5.1**, so it is well covered.
- `setup-fixed.sh` installs from
  `https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml`,
  a **moving target**. It cannot pin a version, so the fixture's floor is a
  documented assumption rather than an enforced one. It only breaks if `stable`
  ever regresses below 2.14, which it will not.
- Fixtures 1 and 3 have no meaningful version floor.
- The local `argocd` CLI is v3.4.2, one minor behind the server. `app
  confirm-deletion` is present in it (verified via `argocd app --help`).

## Verified vs. assumed

Verified against the running cluster:
- Server image `quay.io/argoproj/argocd:v3.5.1`; CLI `v3.4.2`.
- `kubectl apply --dry-run=server` rejects `spec.reploicas` with the strict
  decoding error quoted above. (`--dry-run=client` silently accepts it — client
  side validation is not enough, but Argo CD's sync does a server-side dry run.)
- `argocd app confirm-deletion` exists; `objRequiresPruneConfirmation` and
  `WithPruneConfirmed` are present in the shipped binary.

Assumed, not verified end to end:
- That the whole seed script runs green — it was syntax-checked only, never
  executed, since that would mutate the live cluster and create a sibling repo
  directory.
- The exact wording of the `argocd.argoproj.io/deletion-approved` annotation
  against 3.5.1 (taken from upstream docs, not from this install).
- That `--prune` on `prune-confirm-demo` parks in `Running` rather than failing
  outright. This is the documented behaviour but was not run here.
- That a **dry-run** sync writes `SyncFailed` into `status.operationState`. Only
  the API server's rejection of the manifest was verified, not what Argo CD
  persists. A plain sync is the safe variant for anything reading app status.
