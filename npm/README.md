## Argonaut is no longer available in NPM as of v2.0.0

Please use one of the alternate installation options:

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

---

If for some reason you still want to use the NPM package, you can install v1.16.0 (the last version published to NPM) with:

```bash
npm install -g argonaut-cli@1.6.0
```