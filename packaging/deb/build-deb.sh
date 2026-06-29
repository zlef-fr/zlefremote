#!/usr/bin/env bash
# Build the zlefremote-xfce-plugin .deb (prebuilt .so + .desktop + icon + the
# bundled Linux agent). Prints the path to the built .deb.
#
#   ./build-deb.sh [OUTDIR]        # default OUTDIR = repo dist/
#   VERSION=1.0.1 ./build-deb.sh
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$here/../.." && pwd)"
PLUGIN="$ROOT/panel-plugin"
VERSION="${VERSION:-1.0.0}"
ARCH=amd64
OUT="${1:-$ROOT/dist}"
mkdir -p "$OUT"

# fresh plugin build
make -C "$PLUGIN" clean >/dev/null
make -C "$PLUGIN" >/dev/null

AGENT="$ROOT/dist/zlefremote-agent-linux-amd64"
[ -x "$AGENT" ] || { echo "missing agent binary: $AGENT (run agent/build.sh)" >&2; exit 1; }

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
plugdir="$stage/usr/lib/x86_64-linux-gnu/xfce4/panel/plugins"
deskdir="$stage/usr/share/xfce4/panel/plugins"
icondir="$stage/usr/share/icons/hicolor"
agentdir="$stage/usr/lib/zlefremote"
docdir="$stage/usr/share/doc/zlefremote-xfce-plugin"
mkdir -p "$plugdir" "$deskdir" "$icondir" "$agentdir" "$docdir" "$stage/DEBIAN"

install -m644 "$PLUGIN/libzlefremote.so"      "$plugdir/libzlefremote.so"
install -m644 "$PLUGIN/zlefremote.desktop.in" "$deskdir/zlefremote.desktop"
# full hicolor tree: PNG sizes (render with no SVG loader) + scalable SVG
( cd "$PLUGIN/icons/hicolor" && find . -type f -exec install -Dm644 '{}' "$icondir/{}" \; )
install -m755 "$AGENT"                         "$agentdir/zlefremote-agent"
install -m644 "$PLUGIN/README.md"              "$docdir/README.md"

isize=$(du -ks "$stage" | cut -f1)
cat > "$stage/DEBIAN/control" <<EOF
Package: zlefremote-xfce-plugin
Version: $VERSION
Architecture: $ARCH
Maintainer: ZlefRemote <hello@zlef.fr>
Depends: libgtk-3-0 | libgtk-3-0t64, libxfce4panel-2.0-4, libglib2.0-0 | libglib2.0-0t64
Section: x11
Priority: optional
Homepage: https://remote.zlef.fr
Installed-Size: $isize
Description: ZlefRemote panel plugin for Xfce
 Start a ZlefRemote session and show the pairing QR straight from the Xfce
 panel - turn your phone into a wireless trackpad & keyboard, over local
 Wi-Fi or end-to-end encrypted from anywhere. Bundles the ZlefRemote agent.
EOF

cat > "$stage/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || true
fi
echo "ZlefRemote installed. Reload the panel:  xfce4-panel -r"
echo "then right-click the panel - Add New Items... - \"ZlefRemote\"."
exit 0
EOF
chmod 755 "$stage/DEBIAN/postinst"

( cd "$stage" && find usr -type f -exec md5sum {} + > DEBIAN/md5sums )

deb="$OUT/zlefremote-xfce-plugin_${VERSION}_${ARCH}.deb"
fakeroot dpkg-deb --build --root-owner-group "$stage" "$deb" >/dev/null
echo "$deb"
