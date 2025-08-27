#!/usr/bin/env sh
#                                                     __
#    _____ _______  ____   ____   ____ _____   __ ___/  |_
#    \__  \\_  __ \/ ___\ /  _ \ /    \\__  \ |  |  \   __\
#     / __ \|  | \/ /_/  >  <_> )   |  \/ __ \|  |  /|  |
#    (____  /__|  \___  / \____/|___|  (____  /____/ |__|
#         \/     /_____/             \/     \/
#                        Installer
set -eu

# check that all required commands are available
for cmd in curl tar awk; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: $cmd is required but not installed." >&2
    exit 1
  fi
done

# Optional commands that we have fallbacks for
for cmd in mktemp install; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Warning: $cmd not found, using fallback method" >&2
  fi
done

REPO="darksworm/argonaut"
BIN="argonaut"
VERSION="${1:-}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s | tr 'A-Z' 'a-z')
case "$os" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $os" >&2; exit 1 ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

# Detect libc implementation (musl vs glibc) for Linux systems
libc_suffix=""
if [ "$os" = "linux" ]; then
  # Try to detect musl libc
  if ldd --version 2>&1 | grep -q musl; then
    libc_suffix="-musl"
  elif [ -f /lib/libc.musl-x86_64.so.1 ] || [ -f /lib/libc.musl-aarch64.so.1 ]; then
    libc_suffix="-musl"
  elif getconf GNU_LIBC_VERSION >/dev/null 2>&1; then
    # This is glibc, no suffix needed
    libc_suffix=""
  else
    # Fallback: check if we're running on Alpine (common musl distro)
    if [ -f /etc/alpine-release ]; then
      libc_suffix="-musl"
    fi
  fi
fi

if [ -z "$VERSION" ]; then
  # use curl to fetch the latest version from GitHub
  VERSION="$(curl -s -L "https://api.github.com/repos/$REPO/releases/latest" \
    | awk -F'"' '/"tag_name":/ { print $4; exit }' | cut -c 2-)"
  
  if [ -z "$VERSION" ]; then
    echo "Error: Failed to fetch latest version from GitHub API" >&2
    exit 1
  fi
fi

filename="${BIN}-${VERSION}-${os}-${arch}${libc_suffix}.tar.gz"
url="https://github.com/${REPO}/releases/download/v${VERSION}/${filename}"

# Inform user about detected system
if [ "$os" = "linux" ]; then
  if [ -n "$libc_suffix" ]; then
    echo "Detected musl libc system, downloading musl variant..."
  else
    echo "Detected glibc system, downloading standard variant..."
  fi
fi

echo "Downloading $url..."
# Create temporary directory with fallback for systems without mktemp
if command -v mktemp >/dev/null 2>&1; then
  tmp="$(mktemp -d)"
else
  tmp="/tmp/argonaut-install-$$"
  mkdir -p "$tmp"
fi
trap 'rm -rf "$tmp"' EXIT INT TERM

# Download with fallback for musl systems
if ! curl -s -L -o "$tmp/$filename" "$url" || [ ! -s "$tmp/$filename" ]; then
  if [ "$os" = "linux" ] && [ -n "$libc_suffix" ]; then
    echo "Musl variant not found, falling back to standard glibc version..."
    filename="${BIN}-${VERSION}-${os}-${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/v${VERSION}/${filename}"
    echo "Downloading $url..."
    if ! curl -s -L -o "$tmp/$filename" "$url" || [ ! -s "$tmp/$filename" ]; then
      echo "Error: Failed to download $filename" >&2
      exit 1
    fi
  else
    echo "Error: Failed to download $filename" >&2
    exit 1
  fi
fi

tar -xzf "$tmp/$filename" -C "$tmp"

# Install binary with fallback for systems without install command
if command -v install >/dev/null 2>&1; then
  install -m 0755 "$tmp/$BIN" "$INSTALL_DIR/$BIN"
else
  cp "$tmp/$BIN" "$INSTALL_DIR/$BIN"
  chmod 755 "$INSTALL_DIR/$BIN"
fi

echo "Installed $BIN to $INSTALL_DIR"