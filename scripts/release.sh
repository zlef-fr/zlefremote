#!/usr/bin/env bash
# ZlefRemote — build the signed Android APK and publish it for the landing page
# and the in-app updater:
#
#   dist/app/zlefremote-<versionCode>.apk   served at /app/zlefremote-<code>.apk
#   dist/app/manifest.json                  served at /api/app/release
#
# Gradle is memory-hungry and this host runs a full fleet, so the build goes
# through `limit` (cgroup scope: overflow kills the build, never the host).
#
#   bash scripts/release.sh
set -euo pipefail
cd "$(dirname "$0")/.."

export PATH="/opt/flutter/bin:/root/bin:$PATH"

ENV_FILE=${ZLEFREMOTE_ENV:-/root/.env.zlefremote}
if [ ! -f "$ENV_FILE" ]; then
  echo "missing $ENV_FILE (signing credentials)" >&2
  exit 1
fi
# shellcheck disable=SC1090
. "$ENV_FILE"

VERSION_LINE=$(grep -E '^version:' app/pubspec.yaml)
VERSION_NAME=${VERSION_LINE#version: }
VERSION_NAME=${VERSION_NAME%%+*}
VERSION_CODE=${VERSION_LINE##*+}

echo "Building ZlefRemote v$VERSION_NAME ($VERSION_CODE)…"

# key.properties is generated per build and removed afterwards, so the signing
# password never sits in the working tree.
cat > app/android/key.properties <<EOF
storeFile=$ZLEFREMOTE_KEYSTORE
storePassword=$ZLEFREMOTE_STORE_PASSWORD
keyAlias=$ZLEFREMOTE_KEY_ALIAS
keyPassword=$ZLEFREMOTE_KEY_PASSWORD
EOF
trap 'rm -f app/android/key.properties' EXIT

# One universal APK, minus x86_64: that ABI is emulators only and was 24 MB of
# the 68 MB first build. arm64-v8a + armeabi-v7a covers every real phone, and a
# single file is what a "download" link on the landing page can actually serve
# (the browser can't tell us the device's ABI).
(cd app && limit -c 4 -m 6G -- flutter build apk --release \
  --target-platform android-arm64,android-arm)

APK=app/build/app/outputs/flutter-apk/app-release.apk
OUT=dist/app
mkdir -p "$OUT"
cp "$APK" "$OUT/zlefremote-$VERSION_CODE.apk.tmp"
SHA=$(sha256sum "$OUT/zlefremote-$VERSION_CODE.apk.tmp" | cut -d' ' -f1)
SIZE=$(stat -c%s "$OUT/zlefremote-$VERSION_CODE.apk.tmp")
mv "$OUT/zlefremote-$VERSION_CODE.apk.tmp" "$OUT/zlefremote-$VERSION_CODE.apk"

# keep only the current build plus the one before it (this host watches disk)
ls -1t "$OUT"/zlefremote-*.apk 2>/dev/null | tail -n +3 | xargs -r rm -f

cat > "$OUT/manifest.json.tmp" <<EOF
{
  "versionName": "$VERSION_NAME",
  "versionCode": $VERSION_CODE,
  "url": "https://remote.zlef.fr/app/zlefremote-$VERSION_CODE.apk",
  "sha256": "$SHA",
  "size": $SIZE,
  "minSdk": 24,
  "releasedAt": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
mv "$OUT/manifest.json.tmp" "$OUT/manifest.json"

# The signing certificate's SHA-256 is what Android checks against
# /.well-known/assetlinks.json before it lets pairing links open the app.
FINGERPRINT=$(keytool -list -v -keystore "$ZLEFREMOTE_KEYSTORE" \
  -alias "$ZLEFREMOTE_KEY_ALIAS" -storepass "$ZLEFREMOTE_STORE_PASSWORD" \
  | awk -F': ' '/SHA256:/ {print $2; exit}')

echo
echo "Published v$VERSION_NAME ($VERSION_CODE)"
echo "  sha256      $SHA"
echo "  size        $SIZE bytes"
echo "  signing key $FINGERPRINT"
echo
echo "Check that assetlinks.json carries that fingerprint:"
echo "  curl -s https://remote.zlef.fr/.well-known/assetlinks.json"
