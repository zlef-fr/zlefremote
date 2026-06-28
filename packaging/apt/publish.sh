#!/usr/bin/env bash
# Build the zlefremote-xfce-plugin .deb and publish it to the zlef-wide apt
# repository at apt.zlef.fr. Users then update via `apt upgrade`.
#
# The signing + repo metadata live in the central `apt` project; this just
# builds the package and hands it to that repo's add tool.
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$here/../.." && pwd)"
ADD="${ZLEF_APT_ADD:-/root/projects/apt/bin/zlef-apt-add.sh}"

tmp="$(mktemp -d)"
deb="$("$ROOT/packaging/deb/build-deb.sh" "$tmp")"
echo "built: $deb"

if [ -x "$ADD" ]; then
  "$ADD" "$deb"
  echo "published to apt.zlef.fr"
else
  echo "central publisher not found at $ADD" >&2
  echo "copy $deb to the apt host and run: zlef-apt-add <deb>" >&2
  exit 1
fi
