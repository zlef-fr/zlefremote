#!/usr/bin/env bash
# Copy the phone client into the agent's embed dir and vendor the design tokens
# so LAN mode is fully self-contained (works with no internet on the phone).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
src="$here/../public/app"
dst="$here/web"

rm -rf "$dst"
mkdir -p "$dst"
cp -r "$src/." "$dst/"

# vendor design tokens locally and point the embedded client at them.
# (portable in-place edit — BSD/macOS `sed -i` differs from GNU, so avoid it)
if curl -fsS https://da.zlef.fr/tokens.css -o "$dst/tokens.css" 2>/dev/null; then
  sed 's#https://da.zlef.fr/tokens.css#/app/tokens.css#g' "$dst/index.html" > "$dst/index.html.tmp"
  mv "$dst/index.html.tmp" "$dst/index.html"
  echo "vendored da tokens.css into embed"
else
  echo "WARN: could not fetch da tokens.css — embed will rely on CDN"
fi
echo "synced phone client → agent/web"
