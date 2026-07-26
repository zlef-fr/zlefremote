#!/usr/bin/env bash
# Package ZlefRemote for Windows: the NSIS installer plus a portable zip.
#
# Both bundle the tray app AND the agent, because a tray with no agent can do
# nothing (the app can download one itself, but shipping it is the point of an
# installer). The agent binaries come from the release CI (dist/), the tray is
# cross-built here — see tray/build.sh.
#
# Usage:
#   packaging/windows/build.sh              # amd64
#   VERSION=1.0.1 packaging/windows/build.sh amd64
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/../.." && pwd)"
arch="${1:-amd64}"
version="${VERSION:-1.0.0}"

tray="$root/dist/zlefremote-tray-windows-${arch}.exe"
agent="$root/dist/zlefremote-agent-windows-${arch}.exe"

[ -f "$tray" ]  || { echo "missing $tray — run tray/build.sh first" >&2; exit 1; }
[ -f "$agent" ] || { echo "missing $agent — grab it from the release CI" >&2; exit 1; }

echo "→ installer (NSIS, ${arch}, v${version})"
makensis -V2 -DVERSION="$version" -DARCH="$arch" "$here/zlefremote.nsi"

echo "→ portable zip"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$tmp/ZlefRemote"
cp "$tray"  "$tmp/ZlefRemote/ZlefRemote.exe"
cp "$agent" "$tmp/ZlefRemote/zlefremote-agent.exe"
cp "$here/README-windows.txt" "$tmp/ZlefRemote/README.txt"
zip="$root/dist/zlefremote-windows-portable-${arch}.zip"
rm -f "$zip"
(cd "$tmp" && zip -q -r "$zip" ZlefRemote)

ls -lh "$root/dist/zlefremote-setup-windows-${arch}.exe" "$zip"
