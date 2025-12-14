#!/usr/bin/env bash
set -euo pipefail

FILE="flake.nix"

if [[ ! -f "$FILE" ]]; then
  echo "Cannot find $FILE. Run this script from the repository root." >&2
  exit 1
fi

# 1) Force a mismatch so Nix tells us the correct hash
perl -0777 -i -pe 's/vendorHash\s*=\s*".*?";/vendorHash = "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=";/' "$FILE"

# 2) Build and capture the expected hash from stderr
out="$( (nix build .#default -L 2>&1 || true) )"

# Nix typically prints: got:    sha256-...
new_hash="$(printf "%s" "$out" | sed -n 's/.*got:\s*\(sha256-[A-Za-z0-9+/=]\+\).*/\1/p' | tail -n1)"

if [[ -z "${new_hash}" ]]; then
  echo "Could not extract vendorHash. Build output was:" >&2
  echo "$out" >&2
  exit 1
fi

# Escape slashes for Perl's substitution delimiter
escaped_hash=${new_hash//\//\\/}

# 3) Write the correct hash back
perl -0777 -i -pe "s/vendorHash\\s*=\\s*\".*?\";/vendorHash = \"${escaped_hash}\";/" "$FILE"

echo "Updated vendorHash to: ${new_hash}"
