#!/usr/bin/env bash
# Build & install the ZlefRemote xfce4-panel plugin from source.
#
#   ./install.sh            # system-wide install (uses sudo) — recommended
#   ./install.sh --user     # per-user install into ~/.local (no root)
#   ./install.sh --uninstall [--user]
#   ./install.sh --update [--user]   # fetch the latest release and reinstall
#
# After installing: right-click the Xfce panel → "Panel" → "Add New Items…" →
# search "ZlefRemote".
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
cd "$here"

MODE=system
ACTION=install
for a in "$@"; do
  case "$a" in
    --user)      MODE=user ;;
    --system)    MODE=system ;;
    --uninstall) ACTION=uninstall ;;
    --update)    ACTION=update ;;
    -h|--help)   sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "unknown option: $a" >&2; exit 2 ;;
  esac
done

# --update: fetch the latest release tarball and re-run its installer.
# (Debian/Ubuntu users on the apt repo just use `apt upgrade` instead.)
if [ "$ACTION" = update ]; then
  url="https://remote.zlef.fr/download/zlefremote-xfce-plugin.tar.gz"
  echo "Fetching latest from $url …"
  tmp="$(mktemp -d)"
  curl -fsSL "$url" -o "$tmp/p.tgz" || { echo "download failed" >&2; exit 1; }
  tar xzf "$tmp/p.tgz" -C "$tmp"
  exec sh "$tmp/zlefremote-xfce-plugin/install.sh" "--$MODE"
fi

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1"; return 1; }; }

check_deps() {
  local ok=1
  need cc || need gcc || ok=0
  need make || ok=0
  need pkg-config || ok=0
  if [ "$ok" = 1 ] && ! pkg-config --exists libxfce4panel-2.0 gtk+-3.0; then
    ok=0
  fi
  if [ "$ok" != 1 ]; then
    cat >&2 <<'EOF'

Missing build dependencies. On Debian/Ubuntu:
  sudo apt-get install build-essential pkg-config libgtk-3-dev libxfce4panel-2.0-dev libxfce4util-dev
On Fedora:
  sudo dnf install gcc make pkgconf-pkg-config gtk3-devel xfce4-panel-devel
On Arch:
  sudo pacman -S base-devel gtk3 xfce4-panel
EOF
    exit 1
  fi
}

# resolve the bundled agent binary (next to this script in a release tarball,
# or fall back to the repo dist/ when run from a checkout)
agent_src() {
  for c in "$here/zlefremote-agent" \
           "$here/../dist/zlefremote-agent-linux-amd64"; do
    [ -x "$c" ] && { echo "$c"; return; }
  done
}

if [ "$MODE" = user ]; then
  PREFIX="$HOME/.local"; LIBDIR="$HOME/.local/lib"; DATADIR="$HOME/.local/share"
  AGENT_DIR="$HOME/.local/share/zlefremote"; SUDO=""
else
  PREFIX="/usr"; LIBDIR="/usr/lib"; DATADIR="/usr/share"
  AGENT_DIR="/usr/local/lib/zlefremote"
  SUDO=""; [ "$(id -u)" -ne 0 ] && SUDO="sudo"
fi

if [ "$ACTION" = uninstall ]; then
  $SUDO make uninstall PREFIX="$PREFIX" LIBDIR="$LIBDIR" DATADIR="$DATADIR"
  $SUDO rm -f "$AGENT_DIR/zlefremote-agent"
  echo "Uninstalled. Restart the panel:  xfce4-panel -r"
  exit 0
fi

check_deps
echo "Building plugin…"
make clean >/dev/null 2>&1 || true
make

echo "Installing plugin ($MODE)…"
$SUDO make install PREFIX="$PREFIX" LIBDIR="$LIBDIR" DATADIR="$DATADIR"

AG="$(agent_src || true)"
if [ -n "${AG:-}" ]; then
  echo "Installing agent → $AGENT_DIR/zlefremote-agent"
  $SUDO install -d "$AGENT_DIR"
  $SUDO install -m 0755 "$AG" "$AGENT_DIR/zlefremote-agent"
else
  echo "NOTE: no agent binary bundled. Put 'zlefremote-agent' on your PATH or set"
  echo "      \$ZLEFREMOTE_AGENT to its location. Download it from https://remote.zlef.fr"
fi

cat <<EOF

Done. Now:
  1) xfce4-panel -r            # reload the panel
  2) Right-click the panel → Panel → Add New Items… → "ZlefRemote"
EOF
