# 🐙 Argonaut — Argo CD TUI (Go)

This is the Go (Bubble Tea) port of Argonaut. It mirrors the UX and features of the TypeScript Ink app, with a native terminal implementation and some Go‑specific enhancements.

Below is a copy of the main README for quick reference, followed by Go‑specific notes and configuration for the diff toolchain.

---

## 📦 Prerequisites

- Argo CD CLI installed
- (Optional) delta for enhanced diffs

## ⚡ Quickstart

```bash
# Log in to your Argo CD server
argocd login

# Start Argonaut (Go)
go-app/app
```

---

## ✨ Highlights

- Instant app browsing with live updates
- Scoped navigation: clusters → namespaces → projects → apps
- Command palette (`:`) for actions: `sync`, `diff`, `rollback`, `resources`, etc.
- Live resources view per app with health & sync status
- Diff integration with configurable formatter/viewer (see below)
- Guided rollback with revision metadata and progress streaming
- Keyboard‑only workflow with Vim‑style navigation

---

## Diff Formatter and Interactive Viewer

You can control how diffs are displayed using two environment variables. This separates non‑interactive “formatting” (pretty printing unified diffs) from interactive visual tools that take over the terminal.

- `ARGONAUT_DIFF_FORMATTER` (non‑interactive)
  - A command that reads unified diff from stdin and writes formatted output to stdout.
  - The formatted output is then displayed via Argonaut’s built‑in pager (ov).
  - Defaults to `delta --side-by-side --line-numbers --navigate --paging=never --width=$COLUMNS` if `delta` is available.
  - Example values:
    - `delta --side-by-side --line-numbers --paging=never`
    - `diff-so-fancy`

- `ARGONAUT_DIFF_VIEWER` (interactive)
  - An interactive command that replaces the terminal temporarily. Use `{left}` and `{right}` placeholders for the temp file paths containing live/desired manifests.
  - Examples:
    - `vimdiff {left} {right}`
    - `meld {left} {right}` (GUI, when available)

Behavior:
- If `ARGONAUT_DIFF_VIEWER` is set, Argonaut runs it and restores the TTY on exit.
- Otherwise Argonaut pipes the unified diff through `ARGONAUT_DIFF_FORMATTER` (or delta, if present) and shows formatted output in ov.
- Width is propagated to the formatter via `--width=$COLUMNS` and the `COLUMNS` env var to ensure full‑width output when piping.

Notes:
- The internal pager uses deterministic Vim‑style keys. To avoid conflicts, OV defaults are disabled and Argonaut installs its own keymap: `h/j/k/l`, `g`/`G`, `/`, `q`.

---

## Keyboard Shortcuts (Go pager/OV)

- `j`/`k` → down/up
- `h`/`l` → left/right
- `g`/`G` → top/bottom
- `/` → search, `n`/`N` to navigate results (OV built‑in)
- `q` → exit pager

---

## Docker

Build locally and run:

```bash
docker build -t argonaut-go .
docker run --rm -it -v ~/.config/argocd:/root/.config/argocd:ro argonaut-go
```

