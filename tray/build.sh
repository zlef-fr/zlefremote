#!/usr/bin/env bash
# Build the ZlefRemote Windows tray app.
#
# Unlike the agent (CGO/robotgo, one native build per OS), the tray is pure
# syscalls — it cross-compiles from anywhere with CGO_ENABLED=0, which is why
# the release binaries are produced right here on the Linux host.
#
# Usage:
#   ./build.sh                 # amd64 + arm64 → ../dist/
#   ./build.sh amd64           # one arch
#   ./build.sh --no-resources  # skip the icon/manifest .syso (offline builds)
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here"

archs=()
resources=1
for a in "$@"; do
  case "$a" in
    --no-resources) resources=0 ;;
    amd64|arm64|386) archs+=("$a") ;;
    *) echo "unknown argument: $a" >&2; exit 2 ;;
  esac
done
[ ${#archs[@]} -eq 0 ] && archs=(amd64 arm64)

mkdir -p ../dist assets

# The .ico feeds both the exe resource and the NSIS installer.
go run ./tools/mkico assets/icon-16.png assets/icon-24.png assets/icon-32.png \
  assets/icon-48.png assets/icon-64.png -o assets/zlefremote.ico

for arch in "${archs[@]}"; do
  syso="rsrc_windows_${arch}.syso"
  rm -f "$syso"
  if [ "$resources" = 1 ]; then
    # Icon (Explorer/Alt-Tab/installer) + manifest (asInvoker, per-monitor DPI).
    # rsrc is a build-time tool only; nothing links against it.
    go run github.com/akavel/rsrc@v0.10.2 \
      -ico assets/zlefremote.ico \
      -manifest zlefremote-tray.exe.manifest \
      -arch "$arch" -o "$syso"
  fi

  out="../dist/zlefremote-tray-windows-${arch}.exe"
  echo "building → $out"
  # -H windowsgui: no console window when double-clicked or auto-started.
  GOOS=windows GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "-s -w -H windowsgui" -o "$out" .
  rm -f "$syso"
  ls -lh "$out"
done
