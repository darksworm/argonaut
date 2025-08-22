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
for cmd in curl tar install awk; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: $cmd is required but not installed." >&2
    exit 1
  fi
done

REPO="darksworm/argonaut"
BIN="argonaut"
VERSION="${1:-}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
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

if [ -z "$VERSION" ]; then
  # use wget to fetch the latest version from GitHub
  VERSION="$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" \
    | awk -F'"' '/"tag_name":/ { print $4; exit }' | cut -c 2-)"
fi

filename="${BIN}-${VERSION}-${os}-${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/v${VERSION}/${filename}"

echo "Downloading $url..."
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

wget -O "$tmp/$filename" "$url"
tar -xzf "$tmp/$filename" -C "$tmp"
install -m 0755 "$tmp/$BIN" "$INSTALL_DIR/$BIN"
echo "Installed $BIN to $INSTALL_DIR"