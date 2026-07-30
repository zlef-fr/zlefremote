#!/usr/bin/env bash
# Build the ZlefRemote desktop client (drive another machine from a laptop).
#
# The UI is Ebitengine. On Linux and macOS that needs cgo + the usual GL/X
# headers; on Windows it does NOT (Ebitengine talks to the OS through purego),
# so the Windows binary cross-builds from here.
#
#   Linux  : sudo apt-get install gcc libc6-dev libgl1-mesa-dev libx11-dev \
#            libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev
#   macOS  : xcode-select --install
#
# Usage:
#   ./build.sh                 # current OS/arch
#   ./build.sh windows amd64   # cross-build (Windows only — no cgo needed)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here"

os="${1:-$(go env GOOS)}"
arch="${2:-$(go env GOARCH)}"
ext=""; [ "$os" = "windows" ] && ext=".exe"
mkdir -p ../dist
out="../dist/zlefremote-desktop-${os}-${arch}${ext}"

cgo=1
ldflags="-s -w"
if [ "$os" = "windows" ]; then
  cgo=0
  # no console window behind the app
  ldflags="$ldflags -H windowsgui"
fi

echo "building $os/$arch (CGO_ENABLED=$cgo) → $out"
CGO_ENABLED=$cgo GOOS="$os" GOARCH="$arch" \
  go build -trimpath -ldflags "$ldflags" -o "$out" .
echo "done: $out"
ls -lh "$out"
