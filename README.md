# 🐙 Argonaut — Argo CD TUI

> 💥
> &nbsp; v2 - rewritten in golang!

[![NPM Downloads](https://img.shields.io/npm/dm/argonaut-cli?style=flat-square&label=npm+downloads)](https://www.npmjs.com/package/argonaut-cli)
[![Github Downloads](https://img.shields.io/github/downloads/darksworm/argonaut/total?style=flat-square&label=github+downloads)](https://github.com/darksworm/argonaut/releases/latest)
[![License](https://img.shields.io/github/license/darksworm/argonaut?style=flat-square)](https://github.com/darksworm/argonaut/blob/main/LICENSE)
[![codecov](https://img.shields.io/codecov/c/github/darksworm/argonaut?token=4MYA3DR30R&style=flat-square)](https://codecov.io/github/darksworm/argonaut)
[![Mutation testing badge](https://img.shields.io/endpoint?style=flat-square&url=https%3A%2F%2Fbadge-api.stryker-mutator.io%2Fgithub.com%2Fdarksworm%2Fargonaut%2Fmain)](https://dashboard.stryker-mutator.io/reports/github.com/darksworm/argonaut/main)

Argonaut is a keyboard-first terminal UI for **Argo CD**, built with **Bubbletea**. Browse apps, scope by clusters/namespaces/projects, stream live resource status, trigger syncs, inspect diffs in your favorite pager, and roll back safely — all without leaving your terminal.

> ❤️ 🐶
> &nbsp;Inspired by the great UX of [k9s](https://k9scli.io) — but for Argo CD.

---

## 📦 Prerequisites

- [**Argo CD CLI**](https://argo-cd.readthedocs.io/en/stable/cli_installation/) installed
- [**Delta**](https://dandavison.github.io/delta/installation.html) installed for enhanced diffs (optional, falls back to `git`)

---

## 🚀 Installation methods

<details>
  <summary><strong>Install Script (Linux/macOS)</strong></summary>

```bash
curl -sSL https://raw.githubusercontent.com/darksworm/argonaut/main/install.sh | sh
```

The install script automatically detects your system (including musl vs glibc on Linux) and downloads the appropriate binary from the latest release.

You can also install a specific version:
```bash
curl -sSL https://raw.githubusercontent.com/darksworm/argonaut/main/install.sh | sh -s -- v1.13.0
```
</details>

<details>
  <summary><strong>Homebrew (Linux/MacOS)</strong></summary>

```bash
brew tap darksworm/homebrew-tap
brew install darksworm/tap/argonaut
```
</details>

<details>
  <summary><strong>AUR (Arch User Repository)</strong></summary>

```bash
yay -S argonaut-bin
```
</details>

<details>
  <summary><strong>Docker</strong></summary>

Pull the image:
```bash
docker pull ghcr.io/darksworm/argonaut:latest
```

Run with mounted Argo CD config:
```bash
docker run -it --rm \
  -v ~/.config/argocd:/root/.config/argocd:ro \
  ghcr.io/darksworm/argonaut:latest
```

The container needs access to your Argo CD configuration for authentication. The `:ro` flag mounts it as read-only for security.
</details>

[//]: # (</details>)

[//]: # (<details>)

[//]: # (  <summary><strong>NUR &#40;Nix User Repository&#41;</strong></summary>)

[//]: # ()
[//]: # (```bash)

[//]: # (nix-env -iA nur.repos.darksworm.argonaut)

[//]: # (```)

[//]: # (</details>)

<details>
  <summary><strong>Download a binary</strong></summary>

You can download binaries and packages in from the [**latest release**](https://github.com/darksworm/argonaut/releases/latest).

</details>

## ⚡ Quickstart
```bash
# Log in to your Argo CD server
argocd login

# Start Argonaut
argonaut
```

---

## ✨ Highlights

- **Instant app browsing** with live updates (NDJSON streams)
- **Scoped navigation**: clusters → namespaces → projects → apps
- **Command palette** (`:`) for actions: `sync`, `diff`, `rollback`, `resources`, etc.
- **Live resources view** per app with health & sync status
- **External diff integration**: prefers `delta`, falls back to `git --no-index diff | less`
- **Guided rollback** with revision metadata and progress streaming
- **Keyboard-only workflow** with Vim-like navigation

---

## 📸 Screenshots

### **Live Apps**  
<img src="assets/argonaut_apps.png" alt="Apps list"/>

### **Sync**
<img src="assets/argonaut_sync.png" alt="Sync apps"/>

### **Live Resources**
<img src="assets/argonaut_resources.png" alt="Resources view"/>

### **Diff**  
<img src="assets/argonaut_diff.png" alt="External diff"/>

### **Rollback**  
<img src="assets/argonaut_rollback.png" alt="Rollback flow"/>

### **Delete apps**  
<img src="assets/argonaut_delete_apps.png" alt="Delete apps"/>

### **Enjoy colorful themes**  
<img src="assets/argonaut_themes.gif" alt="Many themes to choose from"/>
## Advanced Features

### Client certificate authentication
Argonaut supports client certificate authentication. You just need to pass a couple arguments to the argonaut command:

```bash
argonaut --client-cert=/path/to/cert --client-cert-key=/path/to/key
```

### Self-signed certificates
If your Argo CD server uses a self-signed certificate, you can provide a custom CA certificate to trust:

```bash
argonaut --ca-cert=/path/to/ca.crt
