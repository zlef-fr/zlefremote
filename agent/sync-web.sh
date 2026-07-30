#!/usr/bin/env bash
# Copy the phone client into the agent's embed dir and vendor the design tokens
# so LAN mode is fully self-contained (works with no internet on the phone).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
src="$here/../public/app"
desk="$here/../public/desk"
dst="$here/web"

rm -rf "$dst"
mkdir -p "$dst"
cp -r "$src/." "$dst/"
# the desktop remote rides along: LAN mode should offer both UIs, and it reuses
# the phone client's transport/crypto modules under /app/js/.
cp -r "$desk" "$dst/desk"

# vendor design tokens locally and point the embedded client at them.
# (portable in-place edit — BSD/macOS `sed -i` differs from GNU, so avoid it)
if curl -fsS https://da.zlef.fr/tokens.css -o "$dst/tokens.css" 2>/dev/null; then
  for f in "$dst/index.html" "$dst/desk/index.html"; do
    sed 's#https://da.zlef.fr/tokens.css#/app/tokens.css#g' "$f" > "$f.tmp"
    mv "$f.tmp" "$f"
  done
  echo "vendored da tokens.css into embed"
else
  echo "WARN: could not fetch da tokens.css — embed will rely on CDN"
fi
echo "synced phone + desktop clients → agent/web"
