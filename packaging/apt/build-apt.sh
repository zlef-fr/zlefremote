#!/usr/bin/env bash
# Build a signed flat APT repository under dist/apt/ so Debian/Ubuntu users can
#   apt install zlefremote-xfce-plugin   (and get updates via apt upgrade).
#
# The signing key is persisted outside the repo (ZR_APT_GNUPGHOME, default
# /root/.config/zlefremote-apt/gnupg) and reused across rebuilds. Only the
# PUBLIC key is published (dist/apt/zlefremote.gpg).
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$here/../.." && pwd)"
APT="$ROOT/dist/apt"
VERSION="${VERSION:-1.0.0}"
KEYMAIL="apt@zlef.fr"
KEYNAME="ZlefRemote APT (package signing)"

export GNUPGHOME="${ZR_APT_GNUPGHOME:-/root/.config/zlefremote-apt/gnupg}"
mkdir -p "$GNUPGHOME"; chmod 700 "$GNUPGHOME"

# 1. signing key (generate once, reuse forever)
if ! gpg --list-secret-keys "$KEYMAIL" >/dev/null 2>&1; then
  echo "generating apt signing key…"
  cat > "$GNUPGHOME/keygen.$$" <<EOF
%no-protection
Key-Type: eddsa
Key-Curve: ed25519
Subkey-Type: ecdh
Subkey-Curve: cv25519
Name-Real: $KEYNAME
Name-Email: $KEYMAIL
Expire-Date: 0
%commit
EOF
  gpg --batch --generate-key "$GNUPGHOME/keygen.$$"
  rm -f "$GNUPGHOME/keygen.$$"
fi
KEYID=$(gpg --list-keys --with-colons "$KEYMAIL" | awk -F: '/^pub/{print $5; exit}')
echo "signing key: $KEYID"

# 2. (re)build repo tree + the .deb into the pool
rm -rf "$APT"
pool="$APT/pool/main/z/zlefremote-xfce-plugin"
mkdir -p "$pool"
VERSION="$VERSION" "$ROOT/packaging/deb/build-deb.sh" "$pool" >/dev/null
echo "deb: $(ls "$pool")"

# 3. Packages index
comp="$APT/dists/stable/main/binary-amd64"
mkdir -p "$comp"
( cd "$APT" && dpkg-scanpackages --multiversion pool /dev/null > "$comp/Packages" )
gzip -9c "$comp/Packages" > "$comp/Packages.gz"

# 4. Release (with checksums of the indices) + signatures
dist="$APT/dists/stable"
apt-ftparchive \
  -o APT::FTPArchive::Release::Origin=ZlefRemote \
  -o APT::FTPArchive::Release::Label=ZlefRemote \
  -o APT::FTPArchive::Release::Suite=stable \
  -o APT::FTPArchive::Release::Codename=stable \
  -o APT::FTPArchive::Release::Architectures=amd64 \
  -o APT::FTPArchive::Release::Components=main \
  -o APT::FTPArchive::Release::Description="ZlefRemote apt repository" \
  release "$dist" > "$dist/Release"

gpg --batch --yes --default-key "$KEYID" -abs  -o "$dist/Release.gpg" "$dist/Release"
gpg --batch --yes --default-key "$KEYID" --clearsign -o "$dist/InRelease" "$dist/Release"

# 5. publish the public key (binary keyring for apt signed-by, + armored copy)
gpg --export "$KEYID"          > "$APT/zlefremote.gpg"
gpg --armor --export "$KEYID"  > "$APT/zlefremote.asc"

echo "repo ready at $APT"
find "$APT" -type f | sed "s#$APT/#  #"
