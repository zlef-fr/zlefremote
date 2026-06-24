#!/usr/bin/env bash
# Build the ZlefRemote agent. Real OS input control needs the `robotgo` build
# tag + a C toolchain and platform headers; without the tag it builds a stub
# (logs input) that still exercises the relay/LAN/QR/E2EE pipeline anywhere.
#
# Native builds (recommended — robotgo is CGO and builds per-OS):
#   Linux  : sudo apt-get install gcc libx11-dev libxtst-dev libxkbcommon-dev xorg-dev
#   macOS  : xcode-select --install
#   Windows: a gcc (e.g. mingw-w64 / msys2)
#
# Usage:
#   ./build.sh            # real agent for the current OS  (-tags robotgo)
#   ./build.sh stub       # portable stub for the current OS (no CGO)
#
# Build setting — bake the anonymous usage ping OFF (no opt-out needed at runtime):
#   TELEMETRY=off ./build.sh
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here"
"$here/sync-web.sh"

mkdir -p ../dist
os="$(go env GOOS)"; arch="$(go env GOARCH)"
ext=""; [ "$os" = "windows" ] && ext=".exe"
out="../dist/zlefremote-agent-${os}-${arch}${ext}"

ldflags="-s -w"
if [ "${TELEMETRY:-on}" = "off" ]; then
  ldflags="$ldflags -X main.telemetryDefault=off"
  echo "telemetry: compiled OFF"
fi

if [ "${1:-real}" = "stub" ]; then
  echo "building STUB → $out"
  CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "$out" .
else
  echo "building REAL (robotgo) → $out"
  CGO_ENABLED=1 go build -tags robotgo -trimpath -ldflags "$ldflags" -o "$out" .
fi
echo "done: $out"
ls -lh "$out"
