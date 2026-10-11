#!/usr/bin/env bash
# Build an MCP bundle (.mcpb) for the current version.
#
# A bundle is a zip holding the manifest and the server binaries. MCPB has a
# "binary" server type, and its guidance prefers static linking, which is what a
# Go build with CGO_ENABLED=0 gives - so a Go server needs no runtime and no
# container to be installable.
#
# Usage: mcpb/build.sh [version] [output-dir]
#
# The version defaults to the git tag if the commit is tagged, otherwise to the
# short hash, so a bundle built from a branch is never mistaken for a release. CI
# passes the tag explicitly, so the bundle and the binaries report the same one.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(dirname "$here")"
out="${1:-}"
dest="${2:-$root}"

if [ -z "$out" ]; then
  if out="$(git -C "$root" describe --tags --exact-match 2>/dev/null)"; then
    out="${out#v}"
  else
    out="0.0.0+$(git -C "$root" rev-parse --short HEAD)"
  fi
fi

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

mkdir -p "$work/server"

# One binary per platform the manifest declares. The manifest's mcp_config names
# server/imagor-mcp and overrides the name on Windows, so the pair below is what
# it expects to find.
echo "building $out"
(cd "$root" && CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o "$work/server/imagor-mcp"     ./cmd/imagor-mcp)
(cd "$root" && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o "$work/server/imagor-mcp.exe" ./cmd/imagor-mcp)

# the manifest wants semver, the release asset wants the tag as written
semver="${out#v}"
python3 - "$here/manifest.json" "$work/manifest.json" "$semver" <<'PY'
import json, sys
src, dst, version = sys.argv[1], sys.argv[2], sys.argv[3]
m = json.load(open(src))
m["version"] = version
json.dump(m, open(dst, "w"), indent=2)
print(f"  manifest version {version}")
PY

cp "$here/.mcpbignore" "$work/.mcpbignore"

cd "$work"
# the CLI validates the manifest against the schema before packing, so a bundle
# that packs is a bundle whose manifest is well formed
npx --yes @anthropic-ai/mcpb pack

artifact="$(ls "$work"/*.mcpb | head -1)"
mkdir -p "$dest"
cp "$artifact" "$dest/imagor-mcp-${out}.mcpb"
echo
echo "  imagor-mcp-${out}.mcpb  $(du -h "$dest/imagor-mcp-${out}.mcpb" | cut -f1)"
sha256sum "$dest/imagor-mcp-${out}.mcpb" 2>/dev/null || shasum -a 256 "$dest/imagor-mcp-${out}.mcpb"
